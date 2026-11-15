package catalog

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/instancebackups/internal/tokens"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

const downloadTokenLifetime = 5 * time.Minute

// IssuedDownload is a single-use token for one stored copy.
type IssuedDownload struct {
	Token     string
	FileName  string
	ExpiresAt time.Time
}

// Download is an opened stored copy, still encrypted as its storage holds it. Closing File ends it.
type Download struct {
	File      io.ReadCloser
	FileName  string
	SizeBytes int64
}

// CreateDownloadToken lets one request download a stored copy of the backup within five minutes.
func (s *Service) CreateDownloadToken(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
	copyID uuid.UUID,
) (IssuedDownload, error) {
	rootAdmin, err := s.users.RequireRootAdmin(ctx, actor)
	if err != nil {
		return IssuedDownload{}, err
	}

	if err := s.pause.Check(); err != nil {
		return IssuedDownload{}, err
	}

	storedCopy, err := s.getDownloadableCopy(ctx, copyID)
	if err != nil {
		return IssuedDownload{}, err
	}

	if storedCopy.BackupID != backupID {
		return IssuedDownload{}, ErrCopyNotFound
	}

	var issued tokens.IssuedToken

	err = s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		issued, err = s.tokens.Issue(txCtx, tokens.Issuance{
			Purpose:  tokens.PurposeInstanceBackupDownload,
			UserID:   rootAdmin.ID,
			CopyID:   &storedCopy.ID,
			Lifetime: downloadTokenLifetime,
		})
		if err != nil {
			return err
		}

		return s.auditLog.Record(txCtx, audit.Event{
			Author:      rootAdmin.GetAuditAuthor(),
			WorkspaceID: nil,
			Message:     "Root admin requested a download of a Databasus backup from " + storedCopy.StorageName,
		})
	})
	if err != nil {
		return IssuedDownload{}, err
	}

	return IssuedDownload{Token: issued.Token, FileName: *storedCopy.FileName, ExpiresAt: issued.ExpiresAt}, nil
}

// OpenDownload spends the token and opens the stored file. A token stays spent when the storage
// cannot serve the file, because a new one is cheap and a retried one could be replayed.
func (s *Service) OpenDownload(ctx context.Context, tokenText string) (Download, error) {
	if err := s.pause.Check(); err != nil {
		return Download{}, err
	}

	token, isConsumed, err := s.tokens.Consume(ctx, tokenText, tokens.PurposeInstanceBackupDownload)
	if err != nil {
		return Download{}, err
	}

	if !isConsumed || token.CopyID == nil {
		return Download{}, ErrTokenNotUsable
	}

	downloader, isFound, err := s.users.GetUser(ctx, token.UserID)
	if err != nil {
		return Download{}, err
	}

	if !isFound || !downloader.IsRootAdmin {
		return Download{}, ErrTokenNotUsable
	}

	storedCopy, err := s.getDownloadableCopy(ctx, *token.CopyID)
	if err != nil {
		return Download{}, err
	}

	backup, _, err := s.backups.get(ctx, storedCopy.BackupID)
	if err != nil {
		return Download{}, err
	}

	file, err := s.openStoredFile(ctx, storedCopy)
	if err != nil {
		return Download{}, err
	}

	err = s.auditLog.Record(ctx, audit.Event{
		Author:      downloader.GetAuditAuthor(),
		WorkspaceID: nil,
		Message:     "Root admin downloaded a Databasus backup from " + storedCopy.StorageName,
	})
	if err != nil {
		_ = file.Close()

		return Download{}, err
	}

	return Download{File: file, FileName: *storedCopy.FileName, SizeBytes: getSizeBytes(backup)}, nil
}

func (s *Service) getDownloadableCopy(ctx context.Context, copyID uuid.UUID) (Copy, error) {
	storedCopy, isFound, err := s.copies.get(ctx, copyID)
	if err != nil {
		return Copy{}, err
	}

	if !isFound || (storedCopy.Status != CopyStatusStored) {
		return Copy{}, ErrCopyNotFound
	}

	if !storedCopy.IsDownloadable() {
		return Copy{}, ErrCopyUnavailable
	}

	return storedCopy, nil
}

func (s *Service) openStoredFile(ctx context.Context, storedCopy Copy) (io.ReadCloser, error) {
	reader, err := s.storages.GetStorageReader(ctx, *storedCopy.StorageID)
	if err != nil {
		return nil, err
	}

	file, err := reader.OpenFile(ctx, *storedCopy.FileName)
	if errors.Is(err, storages.ErrStoredFileNotFound) {
		return nil, ErrCopyUnavailable
	}

	return file, err
}

func getSizeBytes(backup Backup) int64 {
	if backup.SizeBytes == nil {
		return 0
	}

	return *backup.SizeBytes
}
