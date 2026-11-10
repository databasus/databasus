package walstreaming

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
)

type StreamedDatabases interface {
	GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]databases.Database, error)
}

func (s *Streamer) createSlotUnlessDatabaseDeleted(ctx context.Context, logger *slog.Logger) error {
	inspection, err := s.openSourceInspection(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = inspection.Close(context.WithoutCancel(ctx)) }()

	if err := inspection.EnsureSlot(ctx, logger, s.spec.SourceDB.WalStreamSlotName); err != nil {
		return err
	}

	if s.spec.Databases == nil {
		return nil
	}

	existingDatabases, err := s.spec.Databases.GetDatabases(ctx, []uuid.UUID{s.spec.DatabaseID})
	if err != nil {
		return fmt.Errorf("check that the streamed database exists: %w", err)
	}

	if len(existingDatabases) > 0 {
		return nil
	}

	if err := inspection.DropSlot(ctx, logger, s.spec.SourceDB.WalStreamSlotName); err != nil {
		return fmt.Errorf("drop the slot of a deleted database: %w", err)
	}

	return errStreamedDatabaseDeleted
}
