package sftp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

// sftpServer is one SFTP storage while a call runs: the settings it was called with, the storage
// whose sessions it uses and the provider that keeps them.
type sftpServer struct {
	provider  *Provider
	settings  Settings
	storageID uuid.UUID
}

const (
	sftpConnectTimeout     = 30 * time.Second
	sftpTestConnectTimeout = 10 * time.Second
	sftpDeleteTimeout      = 30 * time.Second
)

func (s *sftpServer) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.InfoContext(
		ctx,
		"starting to save file to SFTP storage",
		slog.Any("file_name", fileName),
		slog.Any("host", s.settings.Host),
	)

	lease, err := s.checkoutSession(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect to SFTP", slog.Any("file_name", fileName), slog.Any("error", err))
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	filePath := s.getFilePath(fileName)
	logger.DebugContext(ctx, "uploading file to SFTP", slog.Any("file_name", fileName), slog.Any("file_path", filePath))

	err = writeRemoteFile(lease.GetSession().sftpClient, filePath, &contextReader{ctx: ctx, reader: file})
	lease.Release(err)

	if err != nil {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "SFTP upload cancelled", slog.Any("file_name", fileName))
			return ctx.Err()
		default:
			logger.ErrorContext(
				ctx,
				"failed to upload file to SFTP",
				slog.Any("file_name", fileName),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to upload file to SFTP: %w", err)
		}
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to SFTP storage",
		slog.Any("file_name", fileName),
		slog.Any("file_path", filePath),
	)
	return nil
}

// The plain copy ends in the library's ReadFrom, which for a reader of unknown size, such as the
// compression pipe, writes one 32 KB block at a time and waits for each answer. Over a distant
// server that is hundreds of round trips per WAL segment. The concurrent path keeps blocks in flight
// whatever the reader's size. A failed write may leave holes in the remote file, and the caller
// discards the attempt as it does any failed write.
func writeRemoteFile(client *sftp.Client, filePath string, source io.Reader) error {
	remoteFile, err := client.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}

	if _, err := remoteFile.ReadFromWithConcurrency(source, 0); err != nil {
		_ = remoteFile.Close()
		return err
	}

	if err := remoteFile.Close(); err != nil {
		return fmt.Errorf("failed to close remote file: %w", err)
	}

	return nil
}

func (s *sftpServer) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	filePath := s.getFilePath(fileName)

	fileInfo, err := lease.GetSession().sftpClient.Stat(filePath)
	lease.Release(ignoreNotExist(err))

	if err != nil {
		return nil, fmt.Errorf("failed to stat file on SFTP: %w", err)
	}

	totalBytes := fileInfo.Size()

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: totalBytes,
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return s.openFileAtOffset(attemptCtx, filePath, offsetBytes)
		},
	}), nil
}

func (s *sftpServer) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sftpDeleteTimeout)
	defer cancel()

	lease, err := s.checkoutSession(deleteCtx)
	if err != nil {
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	err = ignoreNotExist(lease.GetSession().sftpClient.Remove(s.getFilePath(fileName)))
	lease.Release(err)

	if err != nil {
		return fmt.Errorf("failed to delete file from SFTP: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from sftp storage", slog.Any("file_name", fileName))

	return nil
}

func (s *sftpServer) testConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sftpTestConnectTimeout)
	defer cancel()

	client, sshConn, err := s.connectWithContext(ctx, sftpTestConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to SFTP: %w", err)
	}
	defer func() {
		_ = client.Close()
		_ = sshConn.Close()
	}()

	if s.settings.Path != "" {
		if err := s.ensureDirectory(client, s.settings.Path); err != nil {
			return fmt.Errorf("failed to access or create path '%s': %w", s.settings.Path, err)
		}
	}

	return nil
}

func (s *sftpServer) connectWithContext(ctx context.Context, timeout time.Duration) (*sftp.Client, *ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	if s.settings.Password != "" {
		password, err := s.provider.fieldEncryptor.Decrypt(s.settings.Password)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt SFTP password: %w", err)
		}
		authMethods = append(authMethods, ssh.Password(password))
	}

	if s.settings.PrivateKey != "" {
		privateKey, err := s.provider.fieldEncryptor.Decrypt(s.settings.PrivateKey)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decrypt SFTP private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey([]byte(privateKey))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Accepted risk: the storage config carries no host key fingerprint to verify against.
	// nosemgrep: go.lang.security.audit.crypto.insecure_ssh.avoid-ssh-insecure-ignore-host-key
	hostKeyCallback := ssh.InsecureIgnoreHostKey()

	config := &ssh.ClientConfig{
		User:            s.settings.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	address := fmt.Sprintf("%s:%d", s.settings.Host, s.settings.Port)

	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial SFTP server: %w", err)
	}

	// The SSH handshake and the SFTP init ignore the context, so a server that accepts the TCP
	// connection and then goes silent would hold the dial forever without this.
	stopCloseOnCancel := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopCloseOnCancel()

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create SSH connection: %w", err)
	}

	sshClient := ssh.NewClient(sshConn, chans, reqs)

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		_ = sshClient.Close()
		return nil, nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	if !stopCloseOnCancel() {
		closeSession(&sftpSession{sshClient: sshClient, sftpClient: sftpClient})
		return nil, nil, fmt.Errorf("failed to connect to SFTP server: %w", ctx.Err())
	}

	return sftpClient, sshClient, nil
}

func (s *sftpServer) ensureDirectory(client *sftp.Client, path string) error {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		if currentPath == "" {
			currentPath = "/" + part
		} else {
			currentPath = currentPath + "/" + part
		}

		if _, err := client.Stat(currentPath); err == nil {
			continue
		}

		// Concurrent sessions dial at once, and the one that loses the race to create the directory
		// finds it in place.
		if err := client.Mkdir(currentPath); err != nil {
			if info, statErr := client.Stat(currentPath); statErr == nil && info.IsDir() {
				continue
			}

			return fmt.Errorf("failed to create directory '%s': %w", currentPath, err)
		}
	}

	return nil
}

func (s *sftpServer) getFilePath(filename string) string {
	if s.settings.Path == "" {
		return filename
	}

	path := strings.TrimPrefix(s.settings.Path, "/")
	path = strings.TrimSuffix(path, "/")

	return "/" + path + "/" + filename
}

type sftpFileReader struct {
	file    *sftp.File
	lease   *connections.Lease[*sftpSession]
	readErr error
}

func (r *sftpFileReader) Read(p []byte) (int, error) {
	readBytes, err := r.file.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.readErr = err
	}

	return readBytes, err
}

func (r *sftpFileReader) Close() error {
	closeErr := r.file.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close file: %w", closeErr)
	}

	r.lease.Release(errors.Join(r.readErr, closeErr))

	return closeErr
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (n int, err error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.reader.Read(p)
	}
}

// Each attempt takes its own session. A resumed read follows a dropped one, and the session that
// carried it was closed when its read failed, so the retry gets a live one.
func (s *sftpServer) openFileAtOffset(
	attemptCtx context.Context,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(attemptCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SFTP: %w", err)
	}

	remoteFile, err := lease.GetSession().sftpClient.Open(filePath)
	if err != nil {
		lease.Release(ignoreNotExist(err))
		return nil, fmt.Errorf("failed to open file from SFTP: %w", err)
	}

	if offsetBytes > 0 {
		if _, err := remoteFile.Seek(offsetBytes, io.SeekStart); err != nil {
			_ = remoteFile.Close()
			lease.Release(err)

			return nil, fmt.Errorf("failed to resume SFTP read at offset %d: %w", offsetBytes, err)
		}
	}

	return &sftpFileReader{file: remoteFile, lease: lease}, nil
}

func ignoreNotExist(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	return err
}
