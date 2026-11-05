package module

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func (m *Module) GetLatestCompletedPhysicalFullBackups(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]catalog.Backup, error) {
	return m.physical.GetLatestCompletedFullBackups(ctx, databaseIDs)
}
