package apptest

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
)

func (a *App) RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, completedAt time.Time) error {
	return a.app.RecordBackupSuccess(ctx, databaseID, completedAt)
}

func (a *App) RecordBackupFailure(ctx context.Context, databaseID uuid.UUID, message string) error {
	return a.app.RecordBackupFailure(ctx, databaseID, message)
}

func (a *App) SetHealthStatus(ctx context.Context, databaseID uuid.UUID, status *databases.HealthStatus) error {
	return a.app.SetHealthStatus(ctx, databaseID, status)
}
