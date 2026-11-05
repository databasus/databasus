package rclone

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/operations"

	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/storages/internal/providers/connections"
)

// configuredRemote is one rclone storage while a call runs: the settings it was called with, the
// storage whose remote it uses and the provider that keeps the remotes.
type configuredRemote struct {
	provider  *Provider
	settings  Settings
	storageID uuid.UUID
}

const (
	rcloneOperationTimeout = 30 * time.Second
	rcloneDeleteTimeout    = 30 * time.Second
)

func (r *configuredRemote) saveFile(ctx context.Context, logger *slog.Logger, fileName string, file io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger.InfoContext(ctx, "starting to save file to rclone storage", slog.Any("file_name", fileName))

	remoteFs, releaseRemote, err := r.acquireFs(ctx)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to create rclone filesystem",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to create rclone filesystem: %w", err)
	}
	defer releaseRemote()

	filePath := r.getFilePath(fileName)
	logger.DebugContext(
		ctx,
		"uploading file via rclone",
		slog.Any("file_name", fileName),
		slog.Any("file_path", filePath),
	)

	_, err = operations.Rcat(ctx, remoteFs, filePath, io.NopCloser(file), time.Now().UTC(), nil)
	if err != nil {
		select {
		case <-ctx.Done():
			logger.InfoContext(ctx, "rclone upload cancelled", slog.Any("file_name", fileName))
			return ctx.Err()
		default:
			logger.ErrorContext(
				ctx,
				"failed to upload file via rclone",
				slog.Any("file_name", fileName),
				slog.Any("error", err),
			)
			return fmt.Errorf("failed to upload file via rclone: %w", err)
		}
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to rclone storage",
		slog.Any("file_name", fileName),
		slog.Any("file_path", filePath),
	)
	return nil
}

func (r *configuredRemote) openFile(ctx context.Context, logger *slog.Logger, fileName string) (io.ReadCloser, error) {
	remoteFs, releaseRemote, err := r.acquireFs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create rclone filesystem: %w", err)
	}

	filePath := r.getFilePath(fileName)

	obj, err := remoteFs.NewObject(ctx, filePath)
	if err != nil {
		releaseRemote()
		return nil, fmt.Errorf("failed to get object from rclone: %w", err)
	}

	fileReader := io_utils.NewResumingReader(io_utils.ResumingReaderSpec{
		StreamCtx:  ctx,
		Logger:     logger,
		FileName:   fileName,
		TotalBytes: obj.Size(),
		OpenAtOffset: func(attemptCtx context.Context, offsetBytes int64) (io.ReadCloser, error) {
			if offsetBytes == 0 {
				return obj.Open(attemptCtx)
			}

			return obj.Open(attemptCtx, &fs.RangeOption{Start: offsetBytes, End: -1})
		},
		IsRetryableError: isRetryableRcloneError,
	})

	return &remoteHoldingReader{ReadCloser: fileReader, releaseRemote: releaseRemote}, nil
}

// The stream keeps reading through the remote after GetFile returns, so the remote stays in use
// until the caller closes the stream.
type remoteHoldingReader struct {
	io.ReadCloser
	releaseRemote func()
}

func (r *remoteHoldingReader) Close() error {
	defer r.releaseRemote()

	return r.ReadCloser.Close()
}

func isRetryableRcloneError(err error) bool {
	return !errors.Is(err, fs.ErrorObjectNotFound) &&
		!errors.Is(err, fs.ErrorDirNotFound) &&
		!errors.Is(err, fs.ErrorNotAFile) &&
		!errors.Is(err, fs.ErrorPermissionDenied)
}

func (r *configuredRemote) deleteFile(ctx context.Context, logger *slog.Logger, fileName string) error {
	// Deletes run from cleanup paths whose caller context is often already cancelled, so the
	// operation outlives a cancellation under its own deadline and keeps the caller's values.
	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rcloneDeleteTimeout)
	defer cancel()

	remoteFs, releaseRemote, err := r.acquireFs(deleteCtx)
	if err != nil {
		return fmt.Errorf("failed to create rclone filesystem: %w", err)
	}
	defer releaseRemote()

	filePath := r.getFilePath(fileName)

	obj, err := remoteFs.NewObject(deleteCtx, filePath)
	if errors.Is(err, fs.ErrorObjectNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to look up file in rclone: %w", err)
	}

	if err := operations.DeleteFile(deleteCtx, obj); err != nil {
		return fmt.Errorf("failed to delete file via rclone: %w", err)
	}

	logger.DebugContext(ctx, "deleted file from rclone storage", slog.Any("file_name", fileName))

	return nil
}

func (r *configuredRemote) testConnection(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rcloneOperationTimeout)
	defer cancel()

	// A connection test often runs for settings that are not saved yet, so it builds its own remote
	// instead of placing one in the cache under a storage ID it may not have.
	remoteFs, err := r.buildFs(ctx)
	if err != nil {
		return fmt.Errorf("failed to create rclone filesystem: %w", err)
	}
	defer shutdownRemote(ctx, remoteFs)

	testFileID := uuid.New().String() + "-test"
	testFilePath := r.getFilePath(testFileID)
	testData := strings.NewReader("test connection")

	_, err = operations.Rcat(
		ctx,
		remoteFs,
		testFilePath,
		io.NopCloser(testData),
		time.Now().UTC(),
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to upload test file via rclone: %w", err)
	}

	obj, err := remoteFs.NewObject(ctx, testFilePath)
	if err != nil {
		return fmt.Errorf("failed to get test file from rclone: %w", err)
	}

	err = obj.Remove(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete test file from rclone: %w", err)
	}

	return nil
}

func (r *configuredRemote) acquireFs(ctx context.Context) (fs.Fs, func(), error) {
	fingerprint := connections.Fingerprint(r.settings.ConfigContent, r.settings.RemotePath)

	return r.provider.remotes.acquire(ctx, r.storageID, fingerprint, r.buildFs)
}

func (r *configuredRemote) buildFs(ctx context.Context) (fs.Fs, error) {
	configContent, err := r.provider.fieldEncryptor.Decrypt(r.settings.ConfigContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt rclone config content: %w", err)
	}

	r.provider.configMutex.Lock()
	defer r.provider.configMutex.Unlock()

	parsedConfig, err := parseConfigContent(configContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse rclone config: %w", err)
	}

	if len(parsedConfig) == 0 {
		return nil, errors.New("rclone config must contain at least one remote section")
	}

	if len(parsedConfig) > 1 {
		return nil, fmt.Errorf(
			"rclone config must contain exactly one remote section, but found %d; create a separate storage for each remote",
			len(parsedConfig),
		)
	}

	var remoteName string
	for section, values := range parsedConfig {
		remoteName = section
		for key, value := range values {
			config.FileSetValue(section, key, value)
		}
	}

	remotePath := remoteName + ":"
	if r.settings.RemotePath != "" {
		remotePath = remoteName + ":" + strings.TrimPrefix(r.settings.RemotePath, "/")
	}

	remoteFs, err := fs.NewFs(ctx, remotePath)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create rclone filesystem for remote '%s': %w",
			remoteName,
			err,
		)
	}

	return remoteFs, nil
}

func (r *configuredRemote) getFilePath(filename string) string {
	return filename
}

func parseConfigContent(content string) (map[string]map[string]string, error) {
	sections := make(map[string]map[string]string)

	var currentSection string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimPrefix(strings.TrimSuffix(line, "]"), "[")
			if sections[currentSection] == nil {
				sections[currentSection] = make(map[string]string)
			}
			continue
		}

		if currentSection != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := ""
			if len(parts) > 1 {
				value = strings.TrimSpace(parts[1])
			}
			sections[currentSection][key] = value
		}
	}

	return sections, scanner.Err()
}
