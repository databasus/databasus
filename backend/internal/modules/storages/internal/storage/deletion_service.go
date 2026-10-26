package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

// The drain runs in the request that deletes the storage, so this is how long it keeps starting files.
// The deadline is read between files and a provider holds its own deadline on the call it already
// began, so an unreachable one costs this plus one provider timeout.
const storageDeletionDrainBudget = 10 * time.Second

// An audit message names at most this many of the files left behind, so a storage with thousands of
// them does not produce an unreadable event.
const maxLeftBehindFileNamesInAudit = 20

// DeleteStorage answers ErrStorageInUse when a backup config or a backup still references the storage,
// which the references of the backup tables enforce.
func (s *Service) DeleteStorage(ctx context.Context, actor users.Actor, storageID uuid.UUID) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		deletedStorage, err := s.getAuthorized(ctx, actor, storageID, workspaces.ActionEditContent, forUpdateRowLock)
		if err != nil {
			return err
		}

		author, err := s.audit.getAuthor(ctx, actor)
		if err != nil {
			return err
		}

		return s.deleteAndDrainFiles(ctx, deletedStorage, author)
	})
}

// OnWorkspaceDeleting deletes each storage of the workspace the way a single deletion does, in the
// workspace deletion's transaction. The files it removes already had a pending deletion, so removing
// them stays correct when the workspace deletion rolls back. A storage that still holds backups of a
// database in another workspace refuses the whole deletion with in_use.
func (s *Service) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	workspaceStorages, err := s.repository.listByWorkspace(ctx, event.WorkspaceID)
	if err != nil {
		return err
	}

	for _, workspaceStorage := range workspaceStorages {
		_, _, err := s.repository.get(ctx, workspaceStorage.ID, forUpdateRowLock)
		if err != nil {
			return err
		}

		err = s.deleteAndDrainFiles(ctx, workspaceStorage, nil)
		if errors.Is(err, ErrStorageInUse) {
			return ErrWorkspaceStorageInUse
		}

		if err != nil {
			return fmt.Errorf("delete storage %s: %w", workspaceStorage.ID, err)
		}
	}

	return nil
}

// deleteAndDrainFiles deletes the storage row first and removes the files of its pending deletions
// only once the database has accepted that, so a storage that a backup config or a backup still
// references keeps every file. The files are removed inside the transaction, because the list of what
// was left behind, written to the audit log, is the user's only way to find those files. The caller
// holds the row lock, so a write that registers its file meanwhile waits for this deletion and no
// pending deletion is cascaded away without reaching the list.
func (s *Service) deleteAndDrainFiles(ctx context.Context, deletedStorage Storage, author *audit.Author) error {
	drainRequest, err := s.fileDeletions.PrepareDrain(ctx, deletedStorage.ID, storageDeletionDrainBudget)
	if err != nil {
		return err
	}

	err = s.repository.delete(ctx, deletedStorage.ID)
	if database.IsForeignKeyViolation(err) {
		return ErrStorageInUse
	}

	if err != nil {
		return err
	}

	leftBehindFileNames := s.fileDeletions.DrainFiles(ctx, drainRequest)

	err = s.audit.recordAs(ctx, author, deletedStorage.WorkspaceID, "Storage deleted: "+deletedStorage.Name)
	if err != nil {
		return err
	}

	if len(leftBehindFileNames) == 0 {
		return nil
	}

	return s.reportLeftBehindFiles(ctx, deletedStorage, leftBehindFileNames)
}

func (s *Service) reportLeftBehindFiles(ctx context.Context, deletedStorage Storage, fileNames []string) error {
	namedFileNames := fileNames[:min(len(fileNames), maxLeftBehindFileNamesInAudit)]

	message := fmt.Sprintf("Storage %s deleted with %d file(s) left behind: %s",
		deletedStorage.Name, len(fileNames), strings.Join(namedFileNames, ", "))
	if len(fileNames) > len(namedFileNames) {
		message += fmt.Sprintf(" and %d more", len(fileNames)-len(namedFileNames))
	}

	s.logger.WarnContext(ctx, "storage deleted with files left behind",
		slog.String("storage_id", deletedStorage.ID.String()),
		slog.Int("left_behind_count", len(fileNames)))

	return s.audit.recordAs(ctx, nil, deletedStorage.WorkspaceID, message)
}
