package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/databases"
)

// StreamedDatabases is the part of the databases module that tells whether the streamed database
// still exists.
type StreamedDatabases interface {
	GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]databases.Database, error)
}

// createSlotUnlessDatabaseDeleted creates the slot, then drops it again when the database is gone. The
// check comes after the creation because the deletion drops the slot after it commits: a slot created
// before that drop goes with it, and a slot created after it is dropped here. A streamer that has not
// received its cancellation yet therefore never leaves a slot of a deleted database behind.
func (s *WalStreamSupervisor) createSlotUnlessDatabaseDeleted(ctx context.Context, logger *slog.Logger) error {
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
