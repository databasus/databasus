package physical

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func (s *Service) GetWalStreamStatistics(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	request *GetWalStreamStatisticsRequest,
) (*GetWalStreamStatisticsResponse, error) {
	_, err := s.authorizeDatabase(ctx, actor, databaseID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}

	period, bucketDuration, isKnownPeriod := request.Period.GetDurations()
	if !isKnownPeriod {
		return nil, ErrUnknownWalStreamStatisticsPeriod
	}

	backupConfig, err := s.configs.GetPhysicalBackupConfig(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	restoreCoverage, err := s.chains.GetRestoreCoverage(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	isFullBackupPending, err := s.isFullBackupPending(ctx, databaseID, backupConfig)
	if err != nil {
		return nil, err
	}

	query := streamhealth.WalStreamStatisticsQuery{
		DatabaseID:          databaseID,
		Period:              period,
		BucketDuration:      bucketDuration,
		SlotRebuildLagBytes: backupConfig.WalLagThresholdBytes,
		IsStreamingExpected: backupConfig.IsWalStreamingExpected(),
		Now:                 s.clock.Now(),
	}

	if restoreCoverage.LatestRestorableAt != nil {
		query.HasCompletedFullBackup = true
		query.RestoreBlockingGaps = restoreCoverage.BlockingGaps
	}

	statistics, err := s.health.GetWalStreamStatistics(ctx, query)
	if err != nil {
		return nil, err
	}

	response := toWalStreamStatisticsResponse(request.Period, bucketDuration, isFullBackupPending, statistics)
	response.LatestRestorableAt = getTimeInUTC(restoreCoverage.LatestRestorableAt)
	response.RestorableRanges = toWalStreamTimeRanges(restoreCoverage.RestorableRanges)

	return response, nil
}

func (s *Service) isFullBackupPending(ctx context.Context,
	databaseID uuid.UUID,
	backupConfig config.PhysicalBackupConfig,
) (bool, error) {
	if backupConfig.ForceFullRequestedAt != nil {
		return true, nil
	}

	runningBackup, _, err := s.backupRepository.FindRunningByDatabaseID(ctx, databaseID)
	if err != nil {
		return false, err
	}

	return runningBackup != nil && runningBackup.Kind == catalog.BackupKindFull, nil
}

func toWalStreamStatisticsResponse(
	period WalStreamStatisticsPeriod,
	bucketDuration time.Duration,
	isFullBackupPending bool,
	statistics *streamhealth.WalStreamStatistics,
) *GetWalStreamStatisticsResponse {
	response := &GetWalStreamStatisticsResponse{
		Health: WalStreamHealth{
			Level:               statistics.Verdict.Level,
			Reasons:             getReasonsInUTC(statistics.Verdict.Reasons),
			Notices:             getNoticesInUTC(statistics.Verdict.Notices),
			IsFullBackupPending: isFullBackupPending,
		},
		Streamer: WalStreamStreamer{
			State:            statistics.StreamerState,
			RestartsLastHour: statistics.RestartsLastHour,
		},
		KeepingUp: statistics.KeepingUp,
		Series: WalStreamSeries{
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
		response.Current = &WalStreamCurrent{
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

func getTimeInUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	return new(value.UTC())
}

func getReasonsInUTC(reasons []streamhealth.Reason) []streamhealth.Reason {
	utcReasons := make([]streamhealth.Reason, 0, len(reasons))

	for _, reason := range reasons {
		reason.Since = getTimeInUTC(reason.Since)
		utcReasons = append(utcReasons, reason)
	}

	return utcReasons
}

func getNoticesInUTC(notices []streamhealth.Notice) []streamhealth.Notice {
	utcNotices := make([]streamhealth.Notice, 0, len(notices))

	for _, notice := range notices {
		notice.Since = getTimeInUTC(notice.Since)
		utcNotices = append(utcNotices, notice)
	}

	return utcNotices
}

func getSeriesPointsInUTC(
	points []*catalog.WalStreamSeriesPoint,
) []*catalog.WalStreamSeriesPoint {
	for _, point := range points {
		point.StartedAt = point.StartedAt.UTC()
	}

	return points
}

func toWalStreamTimeRanges(timeRanges []chains.TimeRange) []WalStreamTimeRange {
	responseRanges := make([]WalStreamTimeRange, 0, len(timeRanges))

	for _, timeRange := range timeRanges {
		responseRanges = append(responseRanges, WalStreamTimeRange{
			From: timeRange.From.UTC(),
			To:   timeRange.To.UTC(),
		})
	}

	return responseRanges
}
