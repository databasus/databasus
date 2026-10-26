package storages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	storage_files "databasus-backend/internal/features/storages/files"
	"databasus-backend/internal/lib/database"
	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
)

// The drain runs in the request that deletes the storage, so this is how long it
// keeps starting files. The deadline is read between files and a provider holds
// its own deadline on the call it already began, so an unreachable one costs this
// plus one provider timeout.
const storageDeletionDrainBudget = 10 * time.Second

// An audit message names at most this many of the files left behind, so a storage
// with thousands of them does not produce an unreadable event.
const maxLeftBehindFileNamesInAudit = 20

// deleteStorageAndDrainFiles deletes the storage row first and removes the files
// of its pending deletions only once the database has accepted that, so a storage
// that a backup config or a backup row still references keeps every file. The files
// are removed inside the transaction, because the list of what was left behind,
// written to the audit log, is the user's only way to find those files.
func (s *StorageService) deleteStorageAndDrainFiles(
	ctx context.Context,
	storage *Storage,
	actorUserID *uuid.UUID,
) error {
	return db.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		// The row lock makes a write that registers its file meanwhile wait for this
		// deletion, so no pending deletion is cascaded away without reaching the list.
		if err := s.storageRepository.LockByID(ctx, storage.ID); err != nil {
			return err
		}

		pendingDeletions, err := getStorageFileDependencies().Repository.FindByStorage(
			database.Conn(ctx, db.GetDb()), storage.ID,
		)
		if err != nil {
			return err
		}

		if err := s.storageRepository.Delete(ctx, storage); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrStorageInUse
			}

			return err
		}

		leftBehindFileNames := getStorageFileDeletionWorker().DrainFiles(ctx, storage_files.DrainRequest{
			Remover:          storage,
			PendingDeletions: pendingDeletions,
			Budget:           storageDeletionDrainBudget,
		})

		s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
			Message:     fmt.Sprintf("Storage deleted: %s", storage.Name),
			UserID:      actorUserID,
			WorkspaceID: &storage.WorkspaceID,
		})

		if len(leftBehindFileNames) > 0 {
			s.reportLeftBehindFiles(ctx, storage, leftBehindFileNames)
		}

		return nil
	})
}

func (s *StorageService) reportLeftBehindFiles(ctx context.Context, storage *Storage, fileNames []string) {
	namedFileNames := fileNames
	if len(namedFileNames) > maxLeftBehindFileNamesInAudit {
		namedFileNames = namedFileNames[:maxLeftBehindFileNamesInAudit]
	}

	message := fmt.Sprintf("Storage %s deleted with %d file(s) left behind: %s",
		storage.Name, len(fileNames), strings.Join(namedFileNames, ", "))
	if len(fileNames) > len(namedFileNames) {
		message += fmt.Sprintf(" and %d more", len(fileNames)-len(namedFileNames))
	}

	logger.GetLogger().WarnContext(ctx, "storage deleted with files left behind",
		"storage_id", storage.ID, "left_behind_count", len(fileNames))

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     message,
		UserID:      nil,
		WorkspaceID: &storage.WorkspaceID,
	})
}
