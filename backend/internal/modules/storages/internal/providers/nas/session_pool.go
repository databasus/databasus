package nas

import (
	"context"
	"fmt"
	"net"

	"github.com/hirochachacha/go-smb2"

	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

type nasSession struct {
	connection net.Conn
	session    *smb2.Session
	share      *smb2.Share
}

// Every call goes through the share bound to the caller's context, so the library itself gives up
// on a request when the caller does.
func (s *nasSession) getShare(ctx context.Context) *smb2.Share {
	return s.share.WithContext(ctx)
}

// Unmounting and logging off are skipped on purpose: a session is also closed to unblock a call
// stuck on a silent server, where each would wait just as long. The server drops the tree and the
// session together with the connection.
func closeSession(session *nasSession) {
	_ = session.connection.Close()
}

func probeSession(ctx context.Context, session *nasSession) error {
	_, err := session.getShare(ctx).Stat(".")

	return err
}

// Credential fields stay in their stored, encrypted form: they change whenever the user saves new
// credentials, and the fingerprint never holds a plain secret.
func (s *nasShare) getSessionKey() connections.Key {
	return connections.Key{
		StorageID: s.storageID,
		Fingerprint: connections.Fingerprint(
			s.settings.Host,
			s.settings.Port,
			s.settings.Share,
			s.settings.Username,
			s.settings.Password,
			s.settings.Domain,
			s.settings.IsSSL,
			s.settings.Path,
		),
	}
}

func (s *nasShare) checkoutSession(ctx context.Context) (*connections.Lease[*nasSession], error) {
	return s.provider.sessionPool.Checkout(ctx, s.getSessionKey(), s.dialSession)
}

// The share is mounted and the target directory ensured once per session rather than on every write.
func (s *nasShare) dialSession(ctx context.Context) (*nasSession, error) {
	connection, err := s.createConnectionWithContext(ctx)
	if err != nil {
		return nil, err
	}

	smbSession, err := s.dialSmbSession(ctx, connection)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}

	share, err := smbSession.WithContext(ctx).Mount(s.settings.Share)
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("failed to mount share '%s': %w", s.settings.Share, err)
	}

	session := &nasSession{connection: connection, session: smbSession, share: share}

	if s.settings.Path != "" {
		if err := s.ensureDirectory(session.getShare(ctx), s.settings.Path); err != nil {
			closeSession(session)
			return nil, fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	return session, nil
}
