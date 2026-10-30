package containers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/hirochachacha/go-smb2"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	readinessProbeInterval = 200 * time.Millisecond
	readinessProbeTimeout  = 5 * time.Second
)

// Docker's port proxy accepts on the mapped port before the service inside listens, so a port
// check alone hands out a connection that is reset or closed during the handshake. A readiness
// probe connects from the host through the mapped port, the way the tests do, and succeeds only
// once the service answers its protocol.
type readinessProbe func(ctx context.Context, address string) error

type probeStrategy struct {
	port           string
	startupTimeout time.Duration
	probe          readinessProbe
}

func forProbe(port string, startupTimeout time.Duration, probe readinessProbe) wait.Strategy {
	return &probeStrategy{port: port, startupTimeout: startupTimeout, probe: probe}
}

// forGreeting waits until the service on port sends a first line that starts with greetingPrefix,
// such as the SSH identification line or the FTP 220 reply.
func forGreeting(port string, startupTimeout time.Duration, greetingPrefix string) wait.Strategy {
	return forProbe(port, startupTimeout, func(ctx context.Context, address string) error {
		return readGreeting(ctx, address, greetingPrefix)
	})
}

// forSmbShare waits until an SMB session logged in with the given credentials mounts the share.
func forSmbShare(port string, startupTimeout time.Duration, username, password, share string) wait.Strategy {
	return forProbe(port, startupTimeout, func(ctx context.Context, address string) error {
		return mountSmbShare(ctx, address, username, password, share)
	})
}

func (s *probeStrategy) WaitUntilReady(ctx context.Context, target wait.StrategyTarget) error {
	ctx, cancel := context.WithTimeout(ctx, s.startupTimeout)
	defer cancel()

	ticker := time.NewTicker(readinessProbeInterval)
	defer ticker.Stop()

	for {
		lastErr := s.probeOnce(ctx, target)
		if lastErr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s did not become ready: %w", s.port, lastErr)
		case <-ticker.C:
		}
	}
}

func (s *probeStrategy) probeOnce(ctx context.Context, target wait.StrategyTarget) error {
	host, err := target.Host(ctx)
	if err != nil {
		return fmt.Errorf("get container host: %w", err)
	}

	mappedPort, err := target.MappedPort(ctx, s.port)
	if err != nil {
		return fmt.Errorf("get mapped port: %w", err)
	}

	probeCtx, cancel := context.WithTimeout(ctx, readinessProbeTimeout)
	defer cancel()

	return s.probe(probeCtx, net.JoinHostPort(host, strconv.Itoa(int(mappedPort.Num()))))
}

func readGreeting(ctx context.Context, address, greetingPrefix string) error {
	conn, err := dialWithDeadline(ctx, address)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	greeting, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return fmt.Errorf("read greeting: %w", err)
	}

	if !strings.HasPrefix(greeting, greetingPrefix) {
		return fmt.Errorf("greeting %q does not start with %q", strings.TrimSpace(greeting), greetingPrefix)
	}

	return nil
}

func mountSmbShare(ctx context.Context, address, username, password, share string) error {
	conn, err := dialWithDeadline(ctx, address)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()

	dialer := &smb2.Dialer{Initiator: &smb2.NTLMInitiator{User: username, Password: password}}

	session, err := dialer.DialContext(ctx, conn)
	if err != nil {
		return fmt.Errorf("log in: %w", err)
	}
	defer func() { _ = session.Logoff() }()

	mountedShare, err := session.Mount(share)
	if err != nil {
		return fmt.Errorf("mount share %s: %w", share, err)
	}

	return mountedShare.Umount()
}

func dialWithDeadline(ctx context.Context, address string) (net.Conn, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			_ = conn.Close()

			return nil, fmt.Errorf("set deadline: %w", err)
		}
	}

	return conn, nil
}
