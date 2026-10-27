package physical_repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_dto "databasus-backend/internal/features/backups/backups/core/physical/dto"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

// Buckets align to date_bin with this origin, the same one Go uses to compute
// the first bucket, so a sample always lands in the bucket the series lists.
var seriesBucketOrigin = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

type PhysicalWalStreamSampleRepository struct{}

type WalStreamSeriesQuery struct {
	DatabaseID     uuid.UUID
	FirstBucketAt  time.Time
	BucketDuration time.Duration
	BucketCount    int

	// Every sample stands for this much time, so a rate divides by the time the
	// samples of a bucket cover, not by the bucket's length, and a partly filled
	// bucket is not understated.
	SampleInterval time.Duration
}

func (r *PhysicalWalStreamSampleRepository) Insert(sample *physical_models.PhysicalWalStreamSample) error {
	if sample.DatabaseID == uuid.Nil {
		return errors.New("database ID is required")
	}

	if sample.ID == uuid.Nil {
		sample.ID = uuid.New()
	}

	return storage.GetDb().Create(sample).Error
}

func (r *PhysicalWalStreamSampleRepository) DeleteOlderThan(cutoff time.Time) (int64, error) {
	result := storage.
		GetDb().
		Where("sampled_at < ?", cutoff).
		Delete(&physical_models.PhysicalWalStreamSample{})

	return result.RowsAffected, result.Error
}

func (r *PhysicalWalStreamSampleRepository) FindNewestByDatabase(
	databaseID uuid.UUID,
) (*physical_models.PhysicalWalStreamSample, error) {
	var sample physical_models.PhysicalWalStreamSample

	err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("sampled_at DESC").
		First(&sample).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &sample, nil
}

// FindSinceByDatabase returns the samples taken at or after since, oldest first.
func (r *PhysicalWalStreamSampleRepository) FindSinceByDatabase(
	databaseID uuid.UUID,
	since time.Time,
) ([]*physical_models.PhysicalWalStreamSample, error) {
	var samples []*physical_models.PhysicalWalStreamSample

	err := storage.
		GetDb().
		Where("database_id = ? AND sampled_at >= ?", databaseID, since).
		Order("sampled_at ASC").
		Find(&samples).Error

	return samples, err
}

// A session that crash-looped stays counted by when it started, even when its
// samples have already left the window.
func (r *PhysicalWalStreamSampleRepository) CountSessionStartsSince(
	databaseID uuid.UUID,
	since time.Time,
) (int, error) {
	var count int64

	err := storage.
		GetDb().
		Model(&physical_models.PhysicalWalStreamSample{}).
		Where("database_id = ? AND sampled_at >= ? AND session_started_at >= ?", databaseID, since, since).
		Distinct("session_started_at").
		Count(&count).Error

	return int(count), err
}

// GetSeries returns exactly BucketCount points, oldest first. Buckets without
// samples come back with every value nil.
func (r *PhysicalWalStreamSampleRepository) GetSeries(
	query WalStreamSeriesQuery,
) ([]*physical_dto.PhysicalWalStreamSeriesPoint, error) {
	if query.BucketCount <= 0 || query.BucketDuration <= 0 || query.SampleInterval <= 0 {
		return nil, errors.New("bucket count, bucket duration and sample interval must be positive")
	}

	var points []*physical_dto.PhysicalWalStreamSeriesPoint

	seriesEndAt := query.FirstBucketAt.Add(query.BucketDuration * time.Duration(query.BucketCount))

	err := storage.
		GetDb().
		Raw(`
			WITH buckets AS (
				SELECT generate_series(
					CAST(@first_bucket_at AS TIMESTAMPTZ),
					CAST(@last_bucket_at AS TIMESTAMPTZ),
					make_interval(secs => @bucket_seconds)
				) AS started_at
			),
			binned_samples AS (
				SELECT
					date_bin(make_interval(secs => @bucket_seconds), sampled_at, CAST(@origin AS TIMESTAMPTZ)) AS started_at,
					SUM(source_written_bytes)::float8
						/ NULLIF(COUNT(source_written_bytes) * @sample_seconds, 0) AS source_write_bytes_per_second,
					SUM(archived_wal_bytes)::float8 / (COUNT(*) * @sample_seconds) AS archive_bytes_per_second,
					SUM(archived_stored_bytes)::float8 / (COUNT(*) * @sample_seconds) AS archived_stored_bytes_per_second,
					MAX(slot_lag_bytes) AS max_slot_lag_bytes,
					MAX(unarchived_wal_bytes) AS max_unarchived_wal_bytes,
					MAX(local_queue_bytes) AS max_local_queue_bytes,
					MAX(local_queue_pause_bytes) AS local_queue_pause_bytes,
					SUM(receiver_restarts)::bigint AS restarts,
					SUM(receiver_pauses)::bigint AS pauses,
					MAX(highest_concurrent_uploads)::bigint AS highest_concurrent_uploads,
					(
						SUM(segment_write_avg_millis::bigint * archived_segments)
							FILTER (WHERE segment_write_avg_millis IS NOT NULL)
						/ NULLIF(SUM(archived_segments) FILTER (WHERE segment_write_avg_millis IS NOT NULL), 0)
					)::bigint AS segment_write_avg_millis,
					MAX(segment_write_max_millis)::bigint AS segment_write_max_millis
				FROM backups_physical_wal_stream_samples
				WHERE database_id = @database_id
				  AND sampled_at >= @first_bucket_at
				  AND sampled_at < @series_end_at
				GROUP BY 1
			),
			binned_session_starts AS (
				SELECT
					date_bin(make_interval(secs => @bucket_seconds), session_started_at, CAST(@origin AS TIMESTAMPTZ)) AS started_at,
					COUNT(DISTINCT session_started_at) AS session_starts
				FROM backups_physical_wal_stream_samples
				WHERE database_id = @database_id
				  AND sampled_at >= @first_bucket_at
				  AND session_started_at >= @first_bucket_at
				  AND session_started_at < @series_end_at
				GROUP BY 1
			)
			SELECT
				buckets.started_at,
				binned_samples.source_write_bytes_per_second,
				binned_samples.archive_bytes_per_second,
				binned_samples.archived_stored_bytes_per_second,
				binned_samples.max_slot_lag_bytes,
				binned_samples.max_unarchived_wal_bytes,
				binned_samples.max_local_queue_bytes,
				binned_samples.local_queue_pause_bytes,
				binned_samples.restarts,
				binned_samples.pauses,
				CASE
					WHEN binned_samples.started_at IS NULL AND binned_session_starts.started_at IS NULL THEN NULL
					ELSE COALESCE(binned_session_starts.session_starts, 0)
				END AS session_starts,
				binned_samples.highest_concurrent_uploads,
				binned_samples.segment_write_avg_millis,
				binned_samples.segment_write_max_millis
			FROM buckets
			LEFT JOIN binned_samples ON binned_samples.started_at = buckets.started_at
			LEFT JOIN binned_session_starts ON binned_session_starts.started_at = buckets.started_at
			ORDER BY buckets.started_at
		`, map[string]any{
			"database_id":     query.DatabaseID,
			"first_bucket_at": query.FirstBucketAt,
			"last_bucket_at":  seriesEndAt.Add(-query.BucketDuration),
			"series_end_at":   seriesEndAt,
			"bucket_seconds":  query.BucketDuration.Seconds(),
			"sample_seconds":  query.SampleInterval.Seconds(),
			"origin":          seriesBucketOrigin,
		}).
		Scan(&points).Error

	return points, err
}

// GetFirstSeriesBucketAt returns the start of the oldest bucket of a series that
// covers period and ends with the bucket holding now: one bucket after the
// bucket that holds now minus period, so the series has period ÷ bucket buckets.
func GetFirstSeriesBucketAt(now time.Time, period, bucketDuration time.Duration) time.Time {
	periodStart := now.Add(-period)
	bucketsSinceOrigin := periodStart.Sub(seriesBucketOrigin) / bucketDuration

	return seriesBucketOrigin.Add(bucketsSinceOrigin*bucketDuration + bucketDuration)
}
