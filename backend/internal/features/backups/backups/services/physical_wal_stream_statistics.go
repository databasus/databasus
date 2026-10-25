package backups_services

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/users"
)

var ErrWalStreamStatisticsNotPhysical = errors.New(
	"WAL stream statistics are only available for physical PostgreSQL databases",
)

var ErrUnknownWalStreamStatisticsPeriod = errors.New("period must be one of HOUR, DAY or WEEK")

// GetWalStreamStatistics returns the WAL stream statistics of a physical
// PostgreSQL database to a user who can view it. A physical database outside
// WAL streaming mode still gets an answer: the not streaming verdict and the
// samples it recorded before.
func (s *PhysicalBackupService) GetWalStreamStatistics(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	request *backups_dto_physical.GetWalStreamStatisticsRequest,
) (*backups_dto_physical.GetWalStreamStatisticsResponse, error) {
	database, err := s.databaseService.GetDatabase(ctx, actor, databaseID)
	if err != nil {
		return nil, err
	}

	if database.Type != databases.DatabaseTypePostgresPhysical {
		return nil, ErrWalStreamStatisticsNotPhysical
	}

	period, bucketDuration, isKnownPeriod := request.Period.GetDurations()
	if !isKnownPeriod {
		return nil, ErrUnknownWalStreamStatisticsPeriod
	}

	backupConfig, err := s.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return nil, err
	}

	restoreCoverage, err := s.chainViewService.GetRestoreCoverage(databaseID)
	if err != nil {
		return nil, err
	}

	isFullBackupPending, err := s.isFullBackupPending(databaseID, backupConfig)
	if err != nil {
		return nil, err
	}

	query := physical_core_service.WalStreamStatisticsQuery{
		DatabaseID:          databaseID,
		Period:              period,
		BucketDuration:      bucketDuration,
		SlotRebuildLagBytes: backupConfig.WalLagThresholdBytes,
		IsStreamingExpected: backupConfig.IsWalStreamingExpected(),
		Now:                 time.Now().UTC(),
	}

	if restoreCoverage.LatestRestorableAt != nil {
		query.HasCompletedFullBackup = true
		query.RestoreBlockingGaps = restoreCoverage.BlockingGaps
	}

	statistics, err := s.physicalCoreService.GetWalStreamStatistics(query)
	if err != nil {
		return nil, err
	}

	response := toWalStreamStatisticsResponse(request.Period, bucketDuration, isFullBackupPending, statistics)
	response.LatestRestorableAt = getTimeInUTC(restoreCoverage.LatestRestorableAt)
	response.RestorableRanges = toWalStreamTimeRanges(restoreCoverage.RestorableRanges)

	return response, nil
}

// Databasus requests a full backup by itself only after a slot rebuild, so the
// screen tells the user to start one unless one is already requested or running.
func (s *PhysicalBackupService) isFullBackupPending(
	databaseID uuid.UUID,
	backupConfig *backups_config_physical.PhysicalBackupConfig,
) (bool, error) {
	if backupConfig.ForceFullRequestedAt != nil {
		return true, nil
	}

	inFlightBackup, err := s.inFlightBackupRepository.FindByDatabaseID(databaseID)
	if err != nil {
		return false, err
	}

	return inFlightBackup != nil && inFlightBackup.BackupType == physical_enums.PhysicalBackupTypeFull, nil
}

func toWalStreamStatisticsResponse(
	period backups_dto_physical.WalStreamStatisticsPeriod,
	bucketDuration time.Duration,
	isFullBackupPending bool,
	statistics *physical_core_service.WalStreamStatistics,
) *backups_dto_physical.GetWalStreamStatisticsResponse {
	response := &backups_dto_physical.GetWalStreamStatisticsResponse{
		Health: backups_dto_physical.WalStreamHealth{
			Level:               statistics.Verdict.Level,
			Reasons:             getReasonsInUTC(statistics.Verdict.Reasons),
			Notices:             getNoticesInUTC(statistics.Verdict.Notices),
			IsFullBackupPending: isFullBackupPending,
		},
		Streamer: backups_dto_physical.WalStreamStreamer{
			State:            statistics.StreamerState,
			RestartsLastHour: statistics.RestartsLastHour,
		},
		KeepingUp: statistics.KeepingUp,
		Series: backups_dto_physical.WalStreamSeries{
			Period:        period,
			BucketSeconds: int64(bucketDuration.Seconds()),
			From:          statistics.SeriesFrom,
			To:            statistics.SeriesTo,
			Points:        getSeriesPointsInUTC(statistics.SeriesPoints),
		},
	}

	if statistics.Streamer != nil {
		response.Streamer.SessionStartedAt = new(statistics.Streamer.StartedAt.UTC())
		response.Streamer.FailingSince = getTimeInUTC(statistics.Streamer.FailingSince)
	}

	if sample := statistics.NewestSample; sample != nil {
		response.Current = &backups_dto_physical.WalStreamCurrent{
			SampledAt:                sample.SampledAt.UTC(),
			LastArchivedWalAt:        getTimeInUTC(statistics.LastArchivedWalAt),
			SessionStartedAt:         sample.SessionStartedAt.UTC(),
			SourceState:              sample.SourceState,
			SourceUnreachableSince:   getTimeInUTC(sample.SourceUnreachableSince),
			SourceWrittenBytes:       sample.SourceWrittenBytes,
			ArchivedSegments:         sample.ArchivedSegments,
			ArchivedWalBytes:         sample.ArchivedWalBytes,
			ArchivedStoredBytes:      sample.ArchivedStoredBytes,
			WalSegmentSizeBytes:      sample.WalSegmentSizeBytes,
			HighestConcurrentUploads: sample.HighestConcurrentUploads,
			SegmentWriteAvgMillis:    sample.SegmentWriteAvgMillis,
			SegmentWriteMaxMillis:    sample.SegmentWriteMaxMillis,
			SlotWalStatus:            sample.SlotWalStatus,
			SlotExtraWalSince:        getTimeInUTC(sample.SlotExtraWalSince),
			SlotLagBytes:             sample.SlotLagBytes,
			SlotSafeWalSizeBytes:     sample.SlotSafeWalSizeBytes,
			UnarchivedWalBytes:       sample.UnarchivedWalBytes,
			ArchiveStalledSince:      getTimeInUTC(sample.ArchiveStalledSince),
			LocalQueueBytes:          sample.LocalQueueBytes,
			LocalQueuePauseBytes:     sample.LocalQueuePauseBytes,
			ReceiverRestarts:         sample.ReceiverRestarts,
			ReceiverPauses:           sample.ReceiverPauses,
			IsReceiverRunning:        sample.IsReceiverRunning,
			IsForcedRotationDenied:   sample.IsForcedRotationDenied,
		}

		if sample.SourceCurrentLSN != nil {
			response.Current.SourceCurrentLsn = new(sample.SourceCurrentLSN.String())
		}

		if sample.ArchiveProgressLSN != nil {
			response.Current.ArchiveProgressLsn = new(sample.ArchiveProgressLSN.String())
		}
	}

	return response
}

// The database driver hands times back in the server's local zone, and the
// statistics promise UTC.
func getTimeInUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	return new(value.UTC())
}

func getReasonsInUTC(reasons []wal_health.Reason) []wal_health.Reason {
	utcReasons := make([]wal_health.Reason, 0, len(reasons))

	for _, reason := range reasons {
		reason.Since = getTimeInUTC(reason.Since)
		utcReasons = append(utcReasons, reason)
	}

	return utcReasons
}

func getNoticesInUTC(notices []wal_health.Notice) []wal_health.Notice {
	utcNotices := make([]wal_health.Notice, 0, len(notices))

	for _, notice := range notices {
		notice.Since = getTimeInUTC(notice.Since)
		utcNotices = append(utcNotices, notice)
	}

	return utcNotices
}

func getSeriesPointsInUTC(
	points []*physical_dto.PhysicalWalStreamSeriesPoint,
) []*physical_dto.PhysicalWalStreamSeriesPoint {
	for _, point := range points {
		point.StartedAt = point.StartedAt.UTC()
	}

	return points
}

func toWalStreamTimeRanges(timeRanges []chain_view.TimeRange) []backups_dto_physical.WalStreamTimeRange {
	responseRanges := make([]backups_dto_physical.WalStreamTimeRange, 0, len(timeRanges))

	for _, timeRange := range timeRanges {
		responseRanges = append(responseRanges, backups_dto_physical.WalStreamTimeRange{
			From: timeRange.From.UTC(),
			To:   timeRange.To.UTC(),
		})
	}

	return responseRanges
}
