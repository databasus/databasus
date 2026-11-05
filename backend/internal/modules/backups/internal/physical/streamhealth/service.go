package streamhealth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// GetWalStreamStatistics reads the samples, the streamer row and the newest
// archived segment of one database and decides its verdict, keeping-up summary
// and chart series at query.Now. The verdict reads the last hour of samples,
// the series reads the requested period.
func (s *Service) GetWalStreamStatistics(
	ctx context.Context,
	query WalStreamStatisticsQuery,
) (*WalStreamStatistics, error) {
	newestSample, _, err := s.walStreamSampleRepository.FindNewestByDatabase(ctx, query.DatabaseID)
	if err != nil {
		return nil, err
	}

	hourAgo := query.Now.Add(-time.Hour)

	recentSamples, err := s.walStreamSampleRepository.FindSinceByDatabase(ctx, query.DatabaseID, hourAgo)
	if err != nil {
		return nil, err
	}

	sessionStartsLastHour, err := s.walStreamSampleRepository.CountSessionStartsSince(ctx, query.DatabaseID, hourAgo)
	if err != nil {
		return nil, err
	}

	streamer, _, err := s.walStreamerRepository.FindByDatabaseID(ctx, query.DatabaseID)
	if err != nil {
		return nil, err
	}

	newestCommittedSegment, _, err := s.walSegmentRepository.FindLatestCommittedSegment(ctx, query.DatabaseID)
	if err != nil {
		return nil, err
	}

	seriesFrom := catalog.GetFirstSeriesBucketAt(query.Now, query.Period, query.BucketDuration)
	bucketCount := int(query.Period / query.BucketDuration)

	seriesPoints, err := s.walStreamSampleRepository.GetSeries(ctx, catalog.WalStreamSeriesQuery{
		DatabaseID:     query.DatabaseID,
		FirstBucketAt:  seriesFrom,
		BucketDuration: query.BucketDuration,
		BucketCount:    bucketCount,
		SampleInterval: SampleInterval,
	})
	if err != nil {
		return nil, err
	}

	statistics := &WalStreamStatistics{
		Verdict: GetVerdict(VerdictInput{
			NewestSample:           newestSample,
			RecentSamples:          recentSamples,
			Streamer:               streamer,
			SessionStartsLastHour:  sessionStartsLastHour,
			SlotRebuildLagBytes:    query.SlotRebuildLagBytes,
			IsStreamingExpected:    query.IsStreamingExpected,
			HasCompletedFullBackup: query.HasCompletedFullBackup,
			RestoreBlockingGaps:    query.RestoreBlockingGaps,
			Now:                    query.Now,
		}),
		KeepingUp:        GetKeepingUp(recentSamples, query.SlotRebuildLagBytes, query.Now),
		StreamerState:    GetStreamerState(streamer, query.Now),
		RestartsLastHour: GetRestartsLastHour(recentSamples, sessionStartsLastHour, query.Now),
		Streamer:         streamer,
		NewestSample:     newestSample,
		SeriesFrom:       seriesFrom,
		SeriesTo:         seriesFrom.Add(time.Duration(bucketCount) * query.BucketDuration),
		SeriesPoints:     seriesPoints,
	}

	if newestCommittedSegment != nil {
		statistics.LastArchivedWalAt = new(newestCommittedSegment.ReceivedAt)
	}

	return statistics, nil
}

type WalStreamStatisticsQuery struct {
	DatabaseID     uuid.UUID
	Period         time.Duration
	BucketDuration time.Duration

	SlotRebuildLagBytes    int64
	IsStreamingExpected    bool
	HasCompletedFullBackup bool
	RestoreBlockingGaps    []catalog.LSNRange

	Now time.Time
}

// Everything the WAL stream statistics need from the catalog, decided at one
// moment. NewestSample, Streamer and LastArchivedWalAt are nil when they do not
// exist yet.
type WalStreamStatistics struct {
	Verdict          Verdict
	KeepingUp        KeepingUp
	StreamerState    StreamerState
	RestartsLastHour int

	Streamer          *catalog.WalStreamer
	NewestSample      *catalog.WalStreamSample
	LastArchivedWalAt *time.Time

	SeriesFrom   time.Time
	SeriesTo     time.Time
	SeriesPoints []*catalog.WalStreamSeriesPoint
}
