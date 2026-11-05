package sshtunnel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	bastionDialTimeout      = 30 * time.Second
	defaultHandshakeTimeout = 30 * time.Second
	acceptErrorBackoff      = 100 * time.Millisecond
)

type OpenSpec struct {
	Settings Settings
	Target   Endpoint
	Logger   *slog.Logger

	// Zero or negative selects defaultHandshakeTimeout. The dialer bounds the TCP connect, but
	// nothing bounds the SSH handshake, so a bastion that accepts and then says nothing needs this.
	HandshakeTimeout time.Duration
}

// The local port stays fixed for the whole lifetime while the SSH client underneath is
// re-established on demand. The WAL supervisor copies the rewritten host and port once at startup
// and then runs for days, so a listener that moved after a network blip would leave it pointing at
// a dead port.
type Forwarder struct {
	target           Endpoint
	localEndpoint    Endpoint
	listener         net.Listener
	clientConfig     *ssh.ClientConfig
	bastionAddress   string
	logger           *slog.Logger
	handshakeTimeout time.Duration

	waitGroup sync.WaitGroup
	done      chan struct{}
	closeOnce sync.Once

	// Cancelled by Close so a redial in flight against an unreachable bastion gives up at once
	// instead of holding clientMutex for the full dial timeout.
	lifetime       context.Context
	cancelLifetime context.CancelFunc

	// Seam: a bastion whose TCP connect never completes is the only way to exercise Close
	// cancelling a redial in flight, and no real address does that reliably on every host.
	dialBastionConn func(ctx context.Context, address string) (net.Conn, error)

	clientMutex sync.Mutex
	client      *ssh.Client
	isClosed    bool
}

// Ctx bounds the first dial and handshake only. The forwarder outlives the request that opened it, so redials run
// on its own lifetime instead, and Close is what ends it.
func Open(ctx context.Context, spec OpenSpec) (*Forwarder, error) {
	authMethods, err := buildAuthMethods(spec.Settings)
	if err != nil {
		return nil, err
	}

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to open local SSH tunnel port: %w", err)
	}

	localAddress, isTCPAddress := listener.Addr().(*net.TCPAddr)
	if !isTCPAddress {
		_ = listener.Close()

		return nil, errors.New("failed to resolve the local SSH tunnel port")
	}

	bastionAddress := Endpoint{Host: spec.Settings.Host, Port: spec.Settings.Port}.String()

	// The accept and keepalive loops log from background goroutines, long after Open returned and
	// with no request scope of their own, so the tunnel they belong to has to be attached here.
	forwarderLogger := spec.Logger.With(
		slog.String("ssh_tunnel_host", bastionAddress),
		slog.String("ssh_tunnel_target", spec.Target.String()),
	)

	handshakeTimeout := spec.HandshakeTimeout
	if handshakeTimeout <= 0 {
		handshakeTimeout = defaultHandshakeTimeout
	}

	lifetime, cancelLifetime := context.WithCancel(context.WithoutCancel(ctx))

	forwarder := &Forwarder{
		target:        spec.Target,
		localEndpoint: Endpoint{Host: "127.0.0.1", Port: localAddress.Port},
		listener:      listener,
		clientConfig: &ssh.ClientConfig{
			User: spec.Settings.Username,
			Auth: authMethods,
			// Accepted risk: the bastion's host key is not pinned, so the credentials are
			// presented to whatever answers its address. Settings carry no fingerprint to
			// verify against; this matches the existing SFTP storage backend.
			// nosemgrep: go.lang.security.audit.crypto.insecure_ssh.avoid-ssh-insecure-ignore-host-key
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		},
		bastionAddress:   bastionAddress,
		logger:           forwarderLogger,
		handshakeTimeout: handshakeTimeout,
		done:             make(chan struct{}),
		lifetime:         lifetime,
		cancelLifetime:   cancelLifetime,
		dialBastionConn:  dialBastionConnOverTCP,
	}

	forwarderLogger.InfoContext(ctx, "ssh tunnel opened", slog.Int("local_port", localAddress.Port))

	if err := forwarder.dialAndCacheBastion(ctx); err != nil {
		cancelLifetime()
		_ = listener.Close()

		return nil, err
	}

	forwarder.waitGroup.Go(forwarder.runAcceptLoop)
	forwarder.waitGroup.Go(forwarder.runKeepalive)

	return forwarder, nil
}

func (f *Forwarder) GetLocalEndpoint() Endpoint {
	return f.localEndpoint
}

func (f *Forwarder) Close() {
	f.closeOnce.Do(func() {
		f.logger.InfoContext(f.lifetime, "ssh tunnel closed")

		close(f.done)
		// Before taking clientMutex: a redial holding it aborts only once its context is cancelled.
		f.cancelLifetime()
		_ = f.listener.Close()
		f.markClosedAndCloseClient()
	})

	f.waitGroup.Wait()
}

func (f *Forwarder) runAcceptLoop() {
	for {
		localConn, err := f.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			f.logger.WarnContext(f.lifetime, "ssh tunnel failed to accept a local connection", slog.Any("error", err))

			select {
			case <-f.done:
				return
			case <-time.After(acceptErrorBackoff):
				continue
			}
		}

		f.waitGroup.Go(func() { f.forwardConnection(localConn) })
	}
}

func (f *Forwarder) forwardConnection(localConn net.Conn) {
	targetConn, err := f.dialTarget()
	if err != nil {
		_ = localConn.Close()

		// Losing a connection to a concurrent Close is the expected shutdown race, not a fault.
		if !errors.Is(err, ErrForwarderClosed) {
			f.logger.WarnContext(f.lifetime, "ssh tunnel failed to reach the database", slog.Any("error", err))
		}

		return
	}

	// Both directions close both ends: whichever side finishes first unblocks the other, which
	// would otherwise sit in io.Copy until Close.
	closeBothEnds := func() {
		_ = localConn.Close()
		_ = targetConn.Close()
	}

	var copyGroup sync.WaitGroup

	copyGroup.Go(func() {
		defer closeBothEnds()

		_, _ = io.Copy(targetConn, localConn)
	})

	copyGroup.Go(func() {
		defer closeBothEnds()

		_, _ = io.Copy(localConn, targetConn)
	})

	copyGroup.Wait()
}

// The database client on the other end of the local port will not retry for us, so a session that
// died since the previous connection has to be replaced within this call.
func (f *Forwarder) dialTarget() (net.Conn, error) {
	client, err := f.getOrDialBastion()
	if err != nil {
		return nil, err
	}

	targetConn, err := client.Dial("tcp", f.target.String())
	if err == nil {
		return targetConn, nil
	}

	// A refused or filtered target is the database's problem, not the transport's: the bastion
	// answered. Discarding a healthy session here would re-authenticate on every connection.
	var openChannelError *ssh.OpenChannelError
	if errors.As(err, &openChannelError) {
		return nil, fmt.Errorf("failed to reach the database through the SSH tunnel host: %w", err)
	}

	f.discardClient(client)

	client, err = f.getOrDialBastion()
	if err != nil {
		return nil, err
	}

	targetConn, err = client.Dial("tcp", f.target.String())
	if err != nil {
		return nil, fmt.Errorf("failed to reach the database through the SSH tunnel host: %w", err)
	}

	return targetConn, nil
}

func (f *Forwarder) getOrDialBastion() (*ssh.Client, error) {
	f.clientMutex.Lock()
	defer f.clientMutex.Unlock()

	if f.client != nil {
		return f.client, nil
	}

	ctx, cancel := context.WithTimeout(f.lifetime, bastionDialTimeout)
	defer cancel()

	if err := f.dialAndCacheBastionLocked(ctx); err != nil {
		return nil, err
	}

	return f.client, nil
}

func (f *Forwarder) dialAndCacheBastion(ctx context.Context) error {
	f.clientMutex.Lock()
	defer f.clientMutex.Unlock()

	return f.dialAndCacheBastionLocked(ctx)
}

func (f *Forwarder) dialAndCacheBastionLocked(ctx context.Context) error {
	// A connection accepted just before Close would otherwise open a session that nothing is left
	// to close, and hold Close in its Wait until the remote peer hangs up.
	if f.isClosed {
		return ErrForwarderClosed
	}

	bastionConn, err := f.dialBastionConn(ctx, f.bastionAddress)
	if err != nil {
		f.logger.ErrorContext(ctx, "failed to dial the ssh tunnel host", slog.Any("error", err))

		return fmt.Errorf("failed to dial the SSH tunnel host: %w", err)
	}

	if err := bastionConn.SetDeadline(time.Now().Add(f.handshakeTimeout)); err != nil {
		_ = bastionConn.Close()

		return fmt.Errorf("failed to bound the SSH tunnel handshake: %w", err)
	}

	stopClosingOnEnd := context.AfterFunc(ctx, func() { _ = bastionConn.Close() })

	sshConn, channels, requests, err := ssh.NewClientConn(bastionConn, f.bastionAddress, f.clientConfig)
	if err != nil {
		stopClosingOnEnd()

		_ = bastionConn.Close()

		f.logger.ErrorContext(ctx, "failed to authenticate with the ssh tunnel host", slog.Any("error", err))

		if ctx.Err() != nil {
			err = context.Cause(ctx)
		}

		return fmt.Errorf("failed to authenticate with the SSH tunnel host: %w", err)
	}

	if !stopClosingOnEnd() {
		_ = sshConn.Close()

		return fmt.Errorf("failed to authenticate with the SSH tunnel host: %w", context.Cause(ctx))
	}

	f.logger.DebugContext(ctx, "connected to the ssh tunnel host")

	// The deadline bounded the handshake only. Forwarded traffic must not inherit it, or every
	// dump longer than the handshake bound would be cut off mid-stream.
	if err := bastionConn.SetDeadline(time.Time{}); err != nil {
		_ = sshConn.Close()

		return fmt.Errorf("failed to clear the SSH tunnel handshake deadline: %w", err)
	}

	f.client = ssh.NewClient(sshConn, channels, requests)

	return nil
}

func dialBastionConnOverTCP(ctx context.Context, address string) (net.Conn, error) {
	dialer := net.Dialer{Timeout: bastionDialTimeout}

	return dialer.DialContext(ctx, "tcp", address)
}

func (f *Forwarder) discardClient(deadClient *ssh.Client) {
	f.clientMutex.Lock()
	defer f.clientMutex.Unlock()

	if f.client != deadClient {
		return
	}

	_ = f.client.Close()
	f.client = nil
}

func (f *Forwarder) markClosedAndCloseClient() {
	f.clientMutex.Lock()
	defer f.clientMutex.Unlock()

	f.isClosed = true

	if f.client == nil {
		return
	}

	_ = f.client.Close()
	f.client = nil
}
