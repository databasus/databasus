package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/storages/internal/providers"
)

const (
	// Chunk size for local storage writes - 8MB per buffer with double-buffering
	// allows overlapped I/O while keeping total memory under 32MB.
	// Two 8MB buffers = 16MB for local storage, plus 8MB for pg_dump buffer = ~25MB total.
	localChunkSize = 8 * 1024 * 1024

	directoryPermissions = 0o755
)

type Deps struct {
	Repositories Repositories
	Folders      providers.LocalFolders
}

type Provider struct {
	repository *settingsRepository
	folders    providers.LocalFolders
}

func New(deps Deps) *Provider {
	return &Provider{repository: deps.Repositories.settings, folders: deps.Folders}
}

func (p *Provider) GetType() providers.Type {
	return providers.TypeLocal
}

func (p *Provider) GetRepository() providers.SettingsRepository {
	return p.repository
}

func (p *Provider) Validate(settings providers.Settings) error {
	_, err := providers.CastSettings[Settings](settings)

	return err
}

func (p *Provider) MergeStoredSettings(merge providers.SettingsMerge) (providers.Settings, error) {
	return providers.CastSettings[Settings](merge.Incoming)
}

func (p *Provider) SaveFile(ctx context.Context, file providers.FileAccess, content io.Reader) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	logger := file.Logger
	fileName := file.FileName

	logger.InfoContext(ctx, "starting to save file to local storage", slog.Any("file_name", fileName))

	tempFilePath := filepath.Join(p.folders.TempFolder, fileName)

	err := os.MkdirAll(filepath.Dir(tempFilePath), directoryPermissions)
	if err != nil {
		return fmt.Errorf("failed to ensure directories: %w", err)
	}

	logger.DebugContext(ctx, "creating temp file", slog.Any("file_name", fileName), slog.Any("temp_path", tempFilePath))

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to create temp file",
			slog.Any("file_name", fileName),
			slog.Any("temp_path", tempFilePath),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to create temp file: %w", err)
	}

	defer func() {
		_ = tempFile.Close()
	}()

	logger.DebugContext(ctx, "copying file data to temp file", slog.Any("file_name", fileName))

	_, err = copyWithContext(ctx, tempFile, content)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to write to temp file",
			slog.Any("file_name", fileName),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	if err = tempFile.Sync(); err != nil {
		logger.ErrorContext(ctx, "failed to sync temp file", slog.Any("file_name", fileName), slog.Any("error", err))

		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	// Close the temp file explicitly before moving it (required on Windows)
	if err = tempFile.Close(); err != nil {
		logger.ErrorContext(ctx, "failed to close temp file", slog.Any("file_name", fileName), slog.Any("error", err))

		return fmt.Errorf("failed to close temp file: %w", err)
	}

	finalPath := filepath.Join(p.folders.DataFolder, fileName)
	logger.DebugContext(
		ctx,
		"moving file from temp to final location",
		slog.Any("file_name", fileName),
		slog.Any("final_path", finalPath),
	)

	if err = os.MkdirAll(filepath.Dir(finalPath), directoryPermissions); err != nil {
		return fmt.Errorf("failed to ensure final directory: %w", err)
	}

	if err = moveFile(tempFilePath, finalPath); err != nil {
		logger.ErrorContext(
			ctx,
			"failed to move file from temp to backups",
			slog.Any("file_name", fileName),
			slog.Any("temp_path", tempFilePath),
			slog.Any("final_path", finalPath),
			slog.Any("error", err),
		)

		return fmt.Errorf("failed to move file from temp to backups: %w", err)
	}

	logger.InfoContext(
		ctx,
		"successfully saved file to local storage",
		slog.Any("file_name", fileName),
		slog.Any("final_path", finalPath),
	)

	return nil
}

func (p *Provider) OpenFile(_ context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	filePath := filepath.Join(p.folders.DataFolder, file.FileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", file.FileName)
	}

	openedFile, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return openedFile, nil
}

func (p *Provider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	// An interrupted write leaves the staging copy behind, and a cross-filesystem
	// publication can leave a partial file at the final path.
	paths := []string{
		filepath.Join(p.folders.DataFolder, file.FileName),
		filepath.Join(p.folders.TempFolder, file.FileName),
	}

	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	file.Logger.DebugContext(ctx, "deleted file from local storage", slog.Any("file_name", file.FileName))

	return nil
}

// TestConnection gives each test its own file, so two tests running at once never remove each
// other's.
func (p *Provider) TestConnection(_ context.Context, _ providers.Location) error {
	testFile := filepath.Join(p.folders.TempFolder, "test_connection_"+uuid.NewString())

	createdFile, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("failed to create test file: %w", err)
	}

	if err = createdFile.Close(); err != nil {
		return fmt.Errorf("failed to close test file: %w", err)
	}

	if err = os.Remove(testFile); err != nil {
		return fmt.Errorf("failed to remove test file: %w", err)
	}

	return nil
}

// moveFile moves a file from src to dst. It first attempts os.Rename for efficiency.
// If rename fails with a cross-device link error (EXDEV), it falls back to copy-then-delete.
// This happens when users mount temp and backups directories as separate Docker volumes
// (e.g., on Unraid with split volume mapping), causing them to reside on different filesystems.
func moveFile(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) || !errors.Is(linkErr.Err, syscall.EXDEV) {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	defer func() {
		_ = srcFile.Close()
	}()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	defer func() {
		_ = dstFile.Close()
	}()

	if _, err = io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err = dstFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	if err = os.Remove(src); err != nil {
		return fmt.Errorf("failed to remove source file: %w", err)
	}

	return nil
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, localChunkSize)

	var written int64

	for {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		nr, readErr := src.Read(buf)
		if nr > 0 {
			nw, writeErr := dst.Write(buf[:nr])

			written += int64(nw)
			if writeErr != nil {
				return written, writeErr
			}

			if nr != nw {
				return written, io.ErrShortWrite
			}
		}

		if readErr == io.EOF {
			return written, nil
		}

		if readErr != nil {
			return written, readErr
		}
	}
}
