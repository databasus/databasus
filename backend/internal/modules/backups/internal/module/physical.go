package module

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func (m *Module) GetLatestCompletedPhysicalFullBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) (*catalog.Backup, error) {
	return m.physical.GetLatestCompletedFullBackup(ctx, databaseID)
}
