package ftp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/textproto"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jlaffaye/ftp"

	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

// ftpServer is one FTP storage while a call runs: the settings it was called with, the storage whose
// sessions it uses and the provider that keeps them.
type ftpServer struct {
	provider  *Provider
	settings  Settings
	storageID uuid.UUID
}

const (
	ftpConnectTimeout     = 30 * time.Second
	ftpTestConnectTimeout = 10 * time.Second
	ftpDeleteTimeout      = 30 * time.Second
)

func (s *ftpServer) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.InfoContext(
		ctx,
		"starting to save file to FTP storage",
		slog.Any("file_name", fileName),
		slog.Any("host", s.settings.Host),
	)

	lease, err := s.checkoutSession(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "failed to connect to FTP", slog.Any("file_name", fileName), slog.Any("error", err))
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}

	filePath := s.getFilePath(fileName)
	logger.DebugContext(ctx, "uploading file to FTP", slog.Any("file_name", fileName), slog.Any("file_path", filePath))

	err = lease.GetSession().serverConn.Stor(filePath, &contextReader{ctx: ctx, reader: file})
	lease.Release(ignoreFtpReply(err))

	if err != nil {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "FTP upload cancelled", slog.Any("file_name", fileName))
			return ctx.Err()
		default:
			logger.ErrorContext(
				ctx,
				"failed to upload file to FTP",
				slog.Any("file_name", fileName),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to upload file to FTP: %w", err)
		}
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to FTP storage",
		slog.Any("file_name", fileName),
		slog.Any("file_path", filePath),
	)
	return nil
}

func (s *ftpServer) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	filePath := s.getFilePath(fileName)

	totalBytes, err := lease.GetSession().serverConn.FileSize(filePath)
	if err != nil {
		// A server that does not implement SIZE still serves the file, so an unsupported command
		// costs the completeness check rather than the whole restore. Any other reply, 550 above
		// all, means the file itself is unreachable.
		if !isUnsupportedFtpCommand(err) {
			lease.Release(ignoreFtpReply(err))

			return nil, fmt.Errorf("failed to stat file on FTP: %w", err)
		}

		logger.WarnContext(
			ctx,
			"storage does not support the FTP SIZE command",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)

		totalBytes = io_utils.UnknownTotalBytes
	}

	lease.Release(nil)

	return io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: totalBytes,
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			return s.retrieveFileFromOffset(attemptCtx, filePath, offsetBytes)
		},
	}), nil
}

func (s *ftpServer) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ftpDeleteTimeout)
	defer cancel()

	lease, err := s.checkoutSession(deleteCtx)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}

	err = lease.GetSession().serverConn.Delete(s.getFilePath(fileName))
	lease.Release(ignoreFtpReply(err))

	if err != nil && !isFtpFileUnavailable(err) {
		return fmt.Errorf("failed to delete file from FTP: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from ftp storage", slog.Any("file_name", fileName))

	return nil
}

func (s *ftpServer) testConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ftpTestConnectTimeout)
	defer cancel()

	session, err := s.connectWithContext(ctx, ftpTestConnectTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect to FTP: %w", err)
	}
	defer closeSession(session)

	if s.settings.Path != "" {
		if err := s.ensureDirectory(session.serverConn, s.settings.Path); err != nil {
			return fmt.Errorf("failed to access or create path '%s': %w", s.settings.Path, err)
		}
	}

	return nil
}

func (s *ftpServer) connectWithContext(ctx context.Context, timeout time.Duration) (*ftpSession, error) {
	password, err := s.provider.fieldEncryptor.Decrypt(s.settings.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt FTP password: %w", err)
	}

	address := fmt.Sprintf("%s:%d", s.settings.Host, s.settings.Port)

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	connections := &trackedConnections{
		open:       map[*trackedConnection]struct{}{},
		dialer:     net.Dialer{Timeout: timeout},
		controlCtx: dialCtx,
	}

	dialOptions := []ftp.DialOption{ftp.DialWithDialFunc(connections.dial)}

	if s.settings.IsSSL {
		connections.tlsConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ServerName:         s.settings.Host,
			InsecureSkipVerify: s.settings.IsSkipTLSVerify,
		}
		dialOptions = append(dialOptions, ftp.DialWithExplicitTLS(connections.tlsConfig))
	}

	session := &ftpSession{connections: connections}

	// The greeting, the TLS upgrade and the login ignore the context, so a server that accepts the
	// TCP connection and then goes silent would hold the dial forever without this.
	stopCloseOnCancel := context.AfterFunc(dialCtx, func() {
		closeSession(session)
	})
	defer stopCloseOnCancel()

	session.serverConn, err = ftp.Dial(address, dialOptions...)
	if err != nil {
		closeSession(session)
		return nil, fmt.Errorf("failed to dial FTP server: %w", err)
	}

	if err := session.serverConn.Login(s.settings.Username, password); err != nil {
		closeSession(session)
		return nil, fmt.Errorf("failed to login to FTP server: %w", err)
	}

	if !stopCloseOnCancel() {
		closeSession(session)
		return nil, fmt.Errorf("failed to connect to FTP server: %w", dialCtx.Err())
	}

	return session, nil
}

func (s *ftpServer) ensureDirectory(conn *ftp.ServerConn, path string) error {
	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, "/")

	if path == "" {
		return nil
	}

	parts := strings.Split(path, "/")

	currentDir, err := conn.CurrentDir()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() {
		_ = conn.ChangeDir(currentDir)
	}()

	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}

		if err := conn.ChangeDir(part); err == nil {
			continue
		}

		// Concurrent sessions dial at once, and the one that loses the race to create the directory
		// finds it in place, so a failed MKD is judged by whether the directory can be entered.
		makeDirErr := conn.MakeDir(part)

		if err := conn.ChangeDir(part); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", part, errors.Join(makeDirErr, err))
		}
	}

	return nil
}

func (s *ftpServer) getFilePath(filename string) string {
	if s.settings.Path == "" {
		return filename
	}

	path := strings.TrimPrefix(s.settings.Path, "/")
	path = strings.TrimSuffix(path, "/")

	return path + "/" + filename
}

type ftpFileReader struct {
	response *ftp.Response
	lease    *connections.Lease[*ftpSession]
	readErr  error
}

func (r *ftpFileReader) Read(p []byte) (int, error) {
	readBytes, err := r.response.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.readErr = err
	}

	return readBytes, err
}

func (r *ftpFileReader) Close() error {
	closeErr := r.response.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close response: %w", closeErr)
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

// Each attempt takes its own session: an FTP data transfer dies together with the control channel
// that opened it, and a session whose read failed was closed, so a resumed REST gets a live one.
func (s *ftpServer) retrieveFileFromOffset(
	attemptCtx context.Context,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(attemptCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to FTP: %w", err)
	}

	response, err := lease.GetSession().serverConn.RetrFrom(filePath, uint64(offsetBytes))
	if err != nil {
		lease.Release(ignoreFtpReply(err))

		return nil, fmt.Errorf("failed to retrieve file from FTP: %w", err)
	}

	return &ftpFileReader{response: response, lease: lease}, nil
}

// A protocol reply such as 550 leaves the control channel in a clean state, so only transport
// failures make a session unusable.
func ignoreFtpReply(err error) error {
	if _, isProtocolError := errors.AsType[*textproto.Error](err); isProtocolError {
		return nil
	}

	return err
}

// 550 is the only reply an FTP server has for a missing file, and also its reply
// for a refused one, so absence is the closest reading the protocol allows.
func isFtpFileUnavailable(err error) bool {
	var protocolError *textproto.Error
	if !errors.As(err, &protocolError) {
		return false
	}

	return protocolError.Code == ftp.StatusFileUnavailable
}

func isUnsupportedFtpCommand(err error) bool {
	var protocolError *textproto.Error
	if !errors.As(err, &protocolError) {
		return false
	}

	return protocolError.Code == ftp.StatusNotImplemented ||
		protocolError.Code == ftp.StatusBadCommand ||
		protocolError.Code == ftp.StatusCommandNotImplemented
}
