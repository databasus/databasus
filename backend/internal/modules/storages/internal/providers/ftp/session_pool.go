package ftp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/jlaffaye/ftp"

	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

type ftpSession struct {
	serverConn  *ftp.ServerConn
	connections *trackedConnections
}

// QUIT is skipped on purpose: a session is also closed to unblock a call stuck on a silent server,
// where a polite goodbye would wait just as long.
func closeSession(session *ftpSession) {
	session.connections.closeAll()
}

// The FTP client takes no context, so the probe deadline is enforced by closing the session.
func probeSession(ctx context.Context, session *ftpSession) error {
	stopCloseOnDeadline := context.AfterFunc(ctx, func() {
		closeSession(session)
	})
	defer stopCloseOnDeadline()

	return session.serverConn.NoOp()
}

// Credential fields stay in their stored, encrypted form: they change whenever the user saves new
// credentials, and the fingerprint never holds a plain secret.
func (s *ftpServer) getSessionKey() connections.Key {
	return connections.Key{
		StorageID: s.storageID,
		Fingerprint: connections.Fingerprint(
			s.settings.Host,
			s.settings.Port,
			s.settings.Username,
			s.settings.Password,
			s.settings.IsSSL,
			s.settings.IsSkipTLSVerify,
			s.settings.Path,
		),
	}
}

func (s *ftpServer) checkoutSession(ctx context.Context) (*connections.Lease[*ftpSession], error) {
	return s.provider.sessionPool.Checkout(ctx, s.getSessionKey(), s.dialSession)
}

// The target directory is ensured once per session rather than walked on every write.
func (s *ftpServer) dialSession(ctx context.Context) (*ftpSession, error) {
	session, err := s.connectWithContext(ctx, ftpConnectTimeout)
	if err != nil {
		return nil, err
	}

	if s.settings.Path != "" {
		if err := s.ensureDirectory(session.serverConn, s.settings.Path); err != nil {
			closeSession(session)
			return nil, fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	return session, nil
}

// The client closes a data connection only when its transfer ends, and a transfer blocked on a
// silent server never ends. Every connection a session opens is tracked, so closing the session
// unblocks the control channel and any data transfer alike.
type trackedConnections struct {
	mutex      sync.Mutex
	open       map[*trackedConnection]struct{}
	isClosed   bool
	dialCount  int
	dialer     net.Dialer
	tlsConfig  *tls.Config
	controlCtx context.Context
}

type trackedConnection struct {
	net.Conn
	owner *trackedConnections
}

func (c *trackedConnection) Close() error {
	c.owner.forget(c)

	return c.Conn.Close()
}

// With a custom dial function the client upgrades only the control channel to TLS by itself, so
// data channels of a TLS session are wrapped here.
func (t *trackedConnections) dial(network, address string) (net.Conn, error) {
	t.mutex.Lock()
	isControlChannel := t.dialCount == 0
	t.dialCount++
	t.mutex.Unlock()

	var (
		rawConnection net.Conn
		err           error
	)

	if isControlChannel {
		rawConnection, err = t.dialer.DialContext(t.controlCtx, network, address)
	} else {
		rawConnection, err = t.dialer.Dial(network, address)
	}

	if err != nil {
		return nil, err
	}

	connection := &trackedConnection{Conn: rawConnection, owner: t}
	if !t.remember(connection) {
		_ = rawConnection.Close()
		return nil, errSessionClosed
	}

	if !isControlChannel && t.tlsConfig != nil {
		return tls.Client(connection, t.tlsConfig), nil
	}

	return connection, nil
}

func (t *trackedConnections) remember(connection *trackedConnection) bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if t.isClosed {
		return false
	}

	t.open[connection] = struct{}{}

	return true
}

func (t *trackedConnections) forget(connection *trackedConnection) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	delete(t.open, connection)
}

func (t *trackedConnections) closeAll() {
	t.mutex.Lock()
	t.isClosed = true
	openConnections := make([]*trackedConnection, 0, len(t.open))

	for connection := range t.open {
		openConnections = append(openConnections, connection)
	}

	t.open = map[*trackedConnection]struct{}{}
	t.mutex.Unlock()

	for _, connection := range openConnections {
		_ = connection.Conn.Close()
	}
}
