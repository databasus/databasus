package module

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/users"
)

func (m *Module) ListPhysicalBackups(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	query *physical.GetPhysicalBackupsRequest,
) (*physical.GetPhysicalBackupsResponse, error) {
	return m.physical.GetBackups(ctx, actor, databaseID, query)
}

func (m *Module) RequestPhysicalBackup(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	kind physical.TriggerBackupType,
) error {
	switch kind {
	case physical.TriggerBackupTypeFull:
		return m.physical.RequestFullBackup(ctx, actor, databaseID)
	case physical.TriggerBackupTypeIncremental:
		return m.physical.RequestIncrementalBackup(ctx, actor, databaseID)
	case physical.TriggerBackupTypeAuto:
		return m.physical.RequestBackup(ctx, actor, databaseID)
	}
	return m.physical.RequestBackup(ctx, actor, databaseID)
}

func (m *Module) CancelPhysicalBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) error {
	return m.physical.CancelBackup(ctx, actor, backupID)
}

func (m *Module) DeletePhysicalBackup(ctx context.Context, actor users.Actor, backupID uuid.UUID) error {
	return m.physical.DeleteBackup(ctx, actor, backupID)
}

func (m *Module) IssuePhysicalRestoreToken(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	targetTime *time.Time,
) (string, error) {
	return m.physical.GenerateRestoreToken(
		ctx,
		actor,
		databaseID,
		&physical.GenerateRestoreTokenRequest{TargetTime: targetTime},
	)
}

func (m *Module) IssuePhysicalBackupRestoreToken(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
) (string, error) {
	return m.physical.GenerateBackupRestoreToken(ctx, actor, backupID)
}

func (m *Module) OpenPhysicalRestoreStream(ctx context.Context, token string) (physical.RestoreStream, error) {
	return m.physical.OpenRestoreStream(ctx, token)
}

func (m *Module) GetWalStreamStatistics(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	period physical.WalStreamStatisticsPeriod,
) (*physical.GetWalStreamStatisticsResponse, error) {
	return m.physical.GetWalStreamStatistics(
		ctx,
		actor,
		databaseID,
		&physical.GetWalStreamStatisticsRequest{Period: period},
	)
}

func (m *Module) GetWalStreamLiveState(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*physical.GetWalStreamLiveResponse, error) {
	return m.physical.GetWalStreamLive(ctx, actor, databaseID)
}

func (m *Module) GetLatestCompletedPhysicalFullBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) (*catalog.FullBackup, error) {
	return m.physical.GetLatestCompletedFullBackup(ctx, databaseID)
}
