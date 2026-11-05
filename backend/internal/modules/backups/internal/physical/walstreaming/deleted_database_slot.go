package walstreaming

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/databases"
)

type StreamedDatabases interface {
	GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]databases.Database, error)
}

func (s *Streamer) createSlotUnlessDatabaseDeleted(ctx context.Context, logger *slog.Logger) error {
	source := pgphysical.NewSource(s.spec.SourceDB.Engine)

	if err := source.VerifySlot(ctx, logger, s.spec.SourceDB.WalStreamSlotName); err != nil {
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

	if err := source.DropSlot(ctx, logger, s.spec.SourceDB.WalStreamSlotName); err != nil {
		return fmt.Errorf("drop the slot of a deleted database: %w", err)
	}

	return errStreamedDatabaseDeleted
}
