package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

const maxAbandonedFileNamesInAudit = 20

type Deletion struct {
	StorageID            uuid.UUID
	ShouldAbandonCleanup bool
}

func (s *Service) DeleteStorage(ctx context.Context, actor users.Actor, deletion Deletion) error {
	return s.transactor.Transaction(ctx, func(ctx context.Context) error {
		deletedStorage, err := s.getAuthorized(
			ctx,
			actor,
			deletion.StorageID,
			workspaces.ActionEditContent,
			forUpdateRowLock,
		)
		if err != nil {
			return err
		}

		pending, err := s.fileDeletions.GetStoragePendingDeletions(ctx, deletedStorage.ID)
		if err != nil {
			return err
		}

		// RESTRICT checks references before cleanup refusal; any refusal rolls the cascade back.
		if err := s.repository.delete(ctx, deletedStorage.ID); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrStorageInUse
			}

			return err
		}

		if len(pending) > 0 && !deletion.ShouldAbandonCleanup {
			return ErrCleanupPending
		}

		message := "Storage deleted: " + deletedStorage.Name
		if len(pending) > 0 {
			names := make([]string, 0, min(len(pending), maxAbandonedFileNamesInAudit))
			for _, obligation := range pending[:min(len(pending), maxAbandonedFileNamesInAudit)] {
				names = append(names, obligation.FileName)
			}

			message += fmt.Sprintf(
				"; automatic cleanup abandoned for %d outstanding file(s): %s",
				len(pending),
				strings.Join(names, ", "),
			)

			if len(pending) > len(names) {
				message += fmt.Sprintf(" and %d more", len(pending)-len(names))
			}
		}

		return s.audit.record(ctx, actor, deletedStorage.WorkspaceID, message)
	})
}

func (s *Service) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	workspaceStorages, err := s.repository.listByWorkspace(ctx, event.WorkspaceID)
	if err != nil {
		return err
	}

	hasPendingCleanup := false
	for _, workspaceStorage := range workspaceStorages {
		_, isFound, err := s.repository.get(ctx, workspaceStorage.ID, forUpdateRowLock)
		if err != nil {
			return err
		}

		if !isFound {
			continue
		}

		pending, err := s.fileDeletions.GetStoragePendingDeletions(ctx, workspaceStorage.ID)
		if err != nil {
			return err
		}

		hasPendingCleanup = hasPendingCleanup || len(pending) > 0

		if err := s.repository.delete(ctx, workspaceStorage.ID); err != nil {
			if database.IsForeignKeyViolation(err) {
				return ErrWorkspaceStorageInUse
			}

			return err
		}
	}

	if hasPendingCleanup {
		return ErrCleanupPending
	}

	return nil
}
