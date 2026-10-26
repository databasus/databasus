package sftp

import (
	"context"
	"fmt"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

type sftpSession struct {
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// The SFTP client's Close waits for its receive loop, which only ends once the SSH connection is
// gone, so the SSH side closes first or a session stuck on a silent server never closes.
func closeSession(session *sftpSession) {
	_ = session.sshClient.Close()
	_ = session.sftpClient.Close()
}

// The SFTP client takes no context, so the probe deadline is enforced by closing the session.
func probeSession(ctx context.Context, session *sftpSession) error {
	stopCloseOnDeadline := context.AfterFunc(ctx, func() {
		closeSession(session)
	})
	defer stopCloseOnDeadline()

	_, err := session.sftpClient.Getwd()

	return err
}

// Credential fields stay in their stored, encrypted form: they change whenever the user saves new
// credentials, and the fingerprint never holds a plain secret.
func (s *sftpServer) getSessionKey() connections.Key {
	return connections.Key{
		StorageID: s.storageID,
		Fingerprint: connections.Fingerprint(
			s.settings.Host, s.settings.Port, s.settings.Username, s.settings.Password, s.settings.PrivateKey,
			s.settings.Path,
		),
	}
}

func (s *sftpServer) checkoutSession(ctx context.Context) (*connections.Lease[*sftpSession], error) {
	return s.provider.sessionPool.Checkout(ctx, s.getSessionKey(), s.dialSession)
}

// The target directory is ensured once per session rather than walked on every write.
func (s *sftpServer) dialSession(ctx context.Context) (*sftpSession, error) {
	sftpClient, sshClient, err := s.connectWithContext(ctx, sftpConnectTimeout)
	if err != nil {
		return nil, err
	}

	session := &sftpSession{sshClient: sshClient, sftpClient: sftpClient}

	if s.settings.Path != "" {
		if err := s.ensureDirectory(sftpClient, s.settings.Path); err != nil {
			closeSession(session)
			return nil, fmt.Errorf("failed to ensure directory: %w", err)
		}
	}

	return session, nil
}
