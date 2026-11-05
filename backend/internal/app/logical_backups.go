package app

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups"
)

func (a *App) GetLogicalBackups(ctx context.Context, backupIDs []uuid.UUID) ([]backups.LogicalBackup, error) {
	return a.backupsModule.GetLogicalBackups(ctx, backupIDs)
}
