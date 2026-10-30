package nas

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"log/slog"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hirochachacha/go-smb2"

	"databasus-backend/internal/lib/background"
	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

// nasShare is one NAS storage while a call runs: the settings it was called with, the storage whose
// sessions it uses and the provider that keeps them.
type nasShare struct {
	provider  *Provider
	settings  Settings
	storageID uuid.UUID
}

const (
	nasDeleteTimeout      = 30 * time.Second
	nasTestConnectTimeout = 30 * time.Second

	// Chunk size for NAS uploads - 16MB provides good balance between
	// memory usage and upload efficiency. This creates backpressure to pg_dump
	// by only reading one chunk at a time and waiting for NAS to confirm receipt.
	nasChunkSize = 16 * 1024 * 1024
)

func (s *nasShare) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.InfoContext(
		ctx,
		"starting to save file to NAS storage",
		slog.Any("file_name", fileName),
		slog.Any("host", s.settings.Host),
	)

	lease, err := s.checkoutSession(ctx)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to create NAS session",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to create NAS session: %w", err)
	}

	filePath := s.getFilePath(fileName)
	logger.DebugContext(ctx, "creating file on NAS", slog.Any("file_name", fileName), slog.Any("file_path", filePath))

	err = s.writeNasFile(ctx, lease.GetSession().getShare(ctx), filePath, file)
	lease.Release(err)

	if err != nil {
		logger.ErrorContext(ctx, "failed to write file to NAS", slog.Any("file_name", fileName), slog.Any("error", err))
		return err
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to NAS storage",
		slog.Any("file_name", fileName),
		slog.Any("file_path", filePath),
	)
	return nil
}

func (s *nasShare) writeNasFile(ctx context.Context, share *smb2.Share, filePath string, source io.Reader) error {
	nasFile, err := share.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file on NAS: %w", err)
	}

	if _, err := copyWithContext(ctx, s.provider.chunkWrites, nasFile, source); err != nil {
		_ = nasFile.Close()
		return fmt.Errorf("failed to write file to NAS: %w", err)
	}

	if err := nasFile.Close(); err != nil {
		return fmt.Errorf("failed to close file on NAS: %w", err)
	}

	return nil
}

func (s *nasShare) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	filePath := s.getFilePath(fileName)

	fileInfo, err := lease.GetSession().getShare(ctx).Stat(filePath)
	lease.Release(ignoreNotExist(err))

	if err != nil {
		return nil, fmt.Errorf("file not found: %s", fileName)
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

func (s *nasShare) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), nasDeleteTimeout)
	defer cancel()

	lease, err := s.checkoutSession(deleteCtx)
	if err != nil {
		return fmt.Errorf("failed to create NAS session: %w", err)
	}

	err = ignoreNotExist(lease.GetSession().getShare(deleteCtx).Remove(s.getFilePath(fileName)))
	lease.Release(err)

	if err != nil {
		return fmt.Errorf("failed to delete file from NAS: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from nas storage", slog.Any("file_name", fileName))

	return nil
}

// testConnection dials its own session instead of placing one in the pool, because a connection
// test often runs for settings that are not saved yet, under a storage ID they may not have.
func (s *nasShare) testConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), nasTestConnectTimeout)
	defer cancel()

	session, err := s.dialSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to NAS: %w", err)
	}
	defer closeSession(session)

	return nil
}

func (s *nasShare) dialSmbSession(ctx context.Context, conn net.Conn) (*smb2.Session, error) {
	password, err := s.provider.fieldEncryptor.Decrypt(s.settings.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt NAS password: %w", err)
	}

	d := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     s.settings.Username,
			Password: password,
			Domain:   s.settings.Domain,
		},
	}

	session, err := d.DialContext(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SMB session: %w", err)
	}

	return session, nil
}

func (s *nasShare) createConnectionWithContext(ctx context.Context) (net.Conn, error) {
	address := net.JoinHostPort(s.settings.Host, fmt.Sprintf("%d", s.settings.Port))

	dialer := &net.Dialer{
		Timeout: 30 * time.Second,
	}

	if s.settings.IsSSL {
		tlsConfig := &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ServerName:         s.settings.Host,
			InsecureSkipVerify: false,
		}
		conn, err := (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to create SSL connection to %s: %w", address, err)
		}
		return conn, nil
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection to %s: %w", address, err)
	}
	return conn, nil
}

// MkdirAll also succeeds when a concurrent session created the directory first, which happens when
// several sessions dial at once.
func (s *nasShare) ensureDirectory(share *smb2.Share, path string) error {
	path = strings.ReplaceAll(filepath.Clean(path), "\\", "/")

	if err := share.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", path, err)
	}

	return nil
}

func (s *nasShare) getFilePath(filename string) string {
	if s.settings.Path == "" {
		return filename
	}

	// Clean path and use forward slashes for SMB
	cleanPath := filepath.Clean(s.settings.Path)
	cleanPath = strings.ReplaceAll(cleanPath, "\\", "/")

	return cleanPath + "/" + filename
}

type nasFileReader struct {
	file    *smb2.File
	lease   *connections.Lease[*nasSession]
	readErr error
}

func (r *nasFileReader) Read(p []byte) (int, error) {
	readBytes, err := r.file.Read(p)
	if err != nil && !errors.Is(err, io.EOF) {
		r.readErr = err
	}

	return readBytes, err
}

func (r *nasFileReader) Close() error {
	closeErr := r.file.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("failed to close file: %w", closeErr)
	}

	r.lease.Release(errors.Join(r.readErr, closeErr))

	return closeErr
}

type writeResult struct {
	bytesWritten int
	writeErr     error
}

func copyWithContext(ctx context.Context, chunkWrites *background.Group, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, nasChunkSize)
	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, readErr := io.ReadFull(src, buf)

		if nr == 0 && readErr == io.EOF {
			break
		}

		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return written, readErr
		}

		writeResultCh := make(chan writeResult, 1)
		chunkWrites.Go(ctx, "nas chunk write", func(context.Context) error {
			nw, writeErr := dst.Write(buf[0:nr])
			writeResultCh <- writeResult{nw, writeErr}

			return nil
		})

		var nw int
		var writeErr error

		select {
		case <-ctx.Done():
			return written, ctx.Err()
		case result := <-writeResultCh:
			nw = result.bytesWritten
			writeErr = result.writeErr
		}

		if nw < 0 || nr < nw {
			nw = 0
			if writeErr == nil {
				writeErr = errors.New("invalid write result")
			}
		}

		if writeErr != nil {
			return written, writeErr
		}

		if nr != nw {
			return written, io.ErrShortWrite
		}

		written += int64(nw)

		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
	}

	return written, nil
}

// Each attempt takes its own session. A resumed read follows a dropped one, and the session that
// carried it was closed when its read failed, so the retry gets a live one.
func (s *nasShare) openFileAtOffset(
	attemptCtx context.Context,
	filePath string,
	offsetBytes int64,
) (io.ReadCloser, error) {
	lease, err := s.checkoutSession(attemptCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create NAS session: %w", err)
	}

	nasFile, err := lease.GetSession().getShare(attemptCtx).Open(filePath)
	if err != nil {
		lease.Release(ignoreNotExist(err))

		return nil, fmt.Errorf("failed to open file from NAS: %w", err)
	}

	if offsetBytes > 0 {
		if _, err := nasFile.Seek(offsetBytes, io.SeekStart); err != nil {
			_ = nasFile.Close()
			lease.Release(err)

			return nil, fmt.Errorf("failed to resume NAS read at offset %d: %w", offsetBytes, err)
		}
	}

	return &nasFileReader{file: nasFile, lease: lease}, nil
}

func ignoreNotExist(err error) error {
	if errors.Is(err, iofs.ErrNotExist) {
		return nil
	}

	return err
}
