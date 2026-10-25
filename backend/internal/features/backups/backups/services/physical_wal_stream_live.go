package backups_services

import (
	"context"
	"time"

	"github.com/google/uuid"

	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	usecases_physical_postgresql "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/users"
)

// GetWalStreamLive returns the current values of a physical PostgreSQL
// database's running WAL stream to a user who can view it. It reads the
// streamer's memory and opens no connection to the source.
func (s *PhysicalBackupService) GetWalStreamLive(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
) (*backups_dto_physical.GetWalStreamLiveResponse, error) {
	database, err := s.databaseService.GetDatabase(ctx, actor, databaseID)
	if err != nil {
		return nil, err
	}

	if database.Type != databases.DatabaseTypePostgresPhysical {
		return nil, ErrWalStreamStatisticsNotPhysical
	}

	liveState, isRunning := s.walStreamSupervisor.GetLiveState(databaseID, time.Now().UTC())
	if !isRunning {
		return &backups_dto_physical.GetWalStreamLiveResponse{}, nil
	}

	return &backups_dto_physical.GetWalStreamLiveResponse{
		IsStreamerRunning: true,
		Current:           toWalStreamLive(liveState),
	}, nil
}

func toWalStreamLive(
	liveState *usecases_physical_postgresql.WalStreamLiveState,
) *backups_dto_physical.WalStreamLive {
	live := &backups_dto_physical.WalStreamLive{
		MeasuredAt:           liveState.MeasuredAt.UTC(),
		SourceProbedAt:       getTimeInUTC(liveState.SourceProbedAt),
		SourceState:          liveState.SourceState,
		SlotWalStatus:        liveState.SlotWalStatus,
		SlotLagBytes:         liveState.SlotLagBytes,
		SlotSafeWalSizeBytes: liveState.SlotSafeWalSizeBytes,
		LastArchivedWalAt:    getTimeInUTC(liveState.LastArchivedWalAt),
		UnarchivedWalBytes:   liveState.UnarchivedWalBytes,
		LocalQueueBytes:      liveState.LocalQueueBytes,
		LocalQueuePauseBytes: liveState.LocalQueuePauseBytes,
		IsReceiverRunning:    liveState.IsReceiverRunning,
	}

	if liveState.SourceCurrentLSN != nil {
		live.SourceCurrentLsn = new(liveState.SourceCurrentLSN.String())
	}

	if liveState.ArchiveProgressLSN != nil {
		live.ArchiveProgressLsn = new(liveState.ArchiveProgressLSN.String())
	}

	return live
}
