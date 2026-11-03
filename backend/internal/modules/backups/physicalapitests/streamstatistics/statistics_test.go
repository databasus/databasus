package streamstatistics_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestGetWalStreamStatistics(t *testing.T) {
	t.Parallel()

	t.Run("returns current values and series when a workspace member requests statistics", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)

		newestSample := catalog.insertSamples(t, samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 10, NewestAt: time.Now().UTC(), Modify: nil})[9]

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Equal(t, "HEALTHY", response.Health.Level)
		assert.Empty(t, response.Health.Reasons)
		assert.Equal(t, "RUNNING", response.Streamer.State)

		require.NotNil(t, response.Current)
		assert.WithinDuration(t, newestSample.SampledAt, response.Current.SampledAt, time.Millisecond)
		assert.Equal(t, "REACHABLE", response.Current.SourceState)
		assert.Equal(t, *newestSample.SlotLagBytes, *response.Current.SlotLagBytes)
		assert.Equal(t, *newestSample.UnarchivedWalBytes, *response.Current.UnarchivedWalBytes)
		assert.Equal(t, newestSample.LocalQueuePauseBytes, response.Current.LocalQueuePauseBytes)

		assert.Equal(t, "KEEPING_UP", response.KeepingUp.State)
		require.NotNil(t, response.KeepingUp.SourceWriteBytesPerSecond)

		assert.Equal(t, "HOUR", response.Series.Period)
		assert.Len(t, response.Series.Points, 60)
	})
	t.Run("returns no statistics when the request is not allowed", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 2, NewestAt: time.Now().UTC(), Modify: nil},
		)

		outsider := catalog.App.CreateNonAdminUser(t, catalog.Admin)

		logicalDatabaseID := catalog.App.InsertDatabase(t, catalog.WorkspaceID)

		testCases := []struct {
			name           string
			url            apptest.PhysicalRequest
			authToken      string
			expectedStatus int
			code           apiclient.ProblemCode
		}{
			{
				name:           "returns 404 without statistics when the user belongs to another workspace",
				url:            statisticsRequest(catalog.DatabaseID, "HOUR"),
				authToken:      "Bearer " + outsider.Token,
				expectedStatus: http.StatusNotFound,
				code:           apiclient.ProblemCodeNotFound,
			},
			{
				name:           "returns 401 without statistics when the request has no session",
				url:            statisticsRequest(catalog.DatabaseID, "HOUR"),
				authToken:      "",
				expectedStatus: http.StatusUnauthorized,
				code:           apiclient.ProblemCodeUnauthenticated,
			},
			{
				name:           "returns 404 without statistics when the database does not exist",
				url:            statisticsRequest(uuid.New(), "HOUR"),
				authToken:      "Bearer " + catalog.Owner.Token,
				expectedStatus: http.StatusNotFound,
				code:           apiclient.ProblemCodeNotFound,
			},
			{
				name:           "returns 400 without statistics when the database uses logical backups",
				url:            statisticsRequest(logicalDatabaseID, "HOUR"),
				authToken:      "Bearer " + catalog.Owner.Token,
				expectedStatus: http.StatusBadRequest,
				code:           apiclient.ProblemCodeInvalidRequest,
			},
			{
				name:           "returns 400 without statistics when the period is unknown",
				url:            statisticsRequest(catalog.DatabaseID, "MONTH"),
				authToken:      "Bearer " + catalog.Owner.Token,
				expectedStatus: http.StatusBadRequest,
				code:           apiclient.ProblemCodeInvalidRequest,
			},
			{
				name:           "returns 400 without statistics when the period is missing",
				url:            apptest.PhysicalStatisticsRequest(catalog.DatabaseID, ""),
				authToken:      "Bearer " + catalog.Owner.Token,
				expectedStatus: http.StatusBadRequest,
				code:           apiclient.ProblemCodeInvalidRequest,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				response := catalog.call(
					t,
					catalogCall{
						Request:       testCase.url,
						Authorization: testCase.authToken,
						StatusCode:    testCase.expectedStatus,
					},
				)

				var problem apiclient.Problem
				require.NoError(t, json.Unmarshal(response.Body, &problem))
				assert.Equal(t, testCase.code, problem.Code)
				assert.NotContains(t, string(response.Body), `"health"`)
				assert.NotContains(t, string(response.Body), `"series"`)
			})
		}
	})
	t.Run("averages over sampled time when the caller requests a week", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)

		firstResponse := getWalStreamStatistics(t, catalog, "WEEK")
		bucketDuration := time.Duration(firstResponse.Series.BucketSeconds) * time.Second
		require.Equal(t, 28*time.Minute, bucketDuration)

		busyBucketStartedAt := firstResponse.Series.From.Add(100 * bucketDuration)
		quietBucketStartedAt := firstResponse.Series.From.Add(200 * bucketDuration)

		busyArchivedBytes := []int64{30 * segmentBytes, 60 * segmentBytes, 90 * segmentBytes}
		for sampleIndex, archivedBytes := range busyArchivedBytes {
			sample := newStreamSample(
				catalog.DatabaseID, busyBucketStartedAt.Add(time.Duration(sampleIndex+1)*time.Minute))
			sample.ArchivedWalBytes = archivedBytes
			catalog.insertSample(t, sample)
		}

		quietSample := newStreamSample(catalog.DatabaseID, quietBucketStartedAt.Add(time.Minute))
		quietSample.ArchivedWalBytes = 3 * segmentBytes
		catalog.insertSample(t, quietSample)

		response := getWalStreamStatistics(t, catalog, "WEEK")
		require.LessOrEqual(t, len(response.Series.Points), 360)

		pointsByStart := make(map[time.Time]*seriesPoint)
		for _, point := range response.Series.Points {
			pointsByStart[point.StartedAt.UTC()] = point
		}

		busyPoint := pointsByStart[busyBucketStartedAt]
		require.NotNil(t, busyPoint)
		require.NotNil(t, busyPoint.ArchiveBytesPerSecond)
		assert.InDelta(t, float64(180*segmentBytes)/(3*sampleInterval.Seconds()),
			*busyPoint.ArchiveBytesPerSecond, 1)

		quietPoint := pointsByStart[quietBucketStartedAt]
		require.NotNil(t, quietPoint)
		require.NotNil(t, quietPoint.ArchiveBytesPerSecond)
		assert.InDelta(t, float64(3*segmentBytes)/sampleInterval.Seconds(),
			*quietPoint.ArchiveBytesPerSecond, 1)
	})
	t.Run("returns sixty consecutive minute buckets when the caller requests an hour", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		requestedAt := catalog.App.Clock.Now().Add(-2 * time.Minute)
		catalog.App.Clock.Set(requestedAt)

		response := getWalStreamStatistics(t, catalog, "HOUR")

		require.Len(t, response.Series.Points, 60)
		assert.Equal(t, int64(60), response.Series.BucketSeconds)

		lastPoint := response.Series.Points[59]

		assert.Equal(t, requestedAt.Truncate(time.Minute), lastPoint.StartedAt)
		assert.Equal(t, response.Series.To, lastPoint.StartedAt.Add(time.Minute))

		for pointIndex := 1; pointIndex < len(response.Series.Points); pointIndex++ {
			assert.Equal(t, time.Minute,
				response.Series.Points[pointIndex].StartedAt.Sub(response.Series.Points[pointIndex-1].StartedAt))
		}
	})
	t.Run("keeps empty points when the streamer has an hour without samples", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)

		now := time.Now().UTC()
		holeStartedAt := now.Add(-2 * time.Hour)
		holeEndedAt := now.Add(-time.Hour)

		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 120, NewestAt: holeStartedAt, Modify: nil},
		)
		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 120, NewestAt: now, Modify: nil},
		)

		response := getWalStreamStatistics(t, catalog, "DAY")
		bucketDuration := time.Duration(response.Series.BucketSeconds) * time.Second

		holePointCount := 0

		for _, point := range response.Series.Points {
			isInsideHole := point.StartedAt.After(holeStartedAt) &&
				point.StartedAt.Add(bucketDuration).Before(holeEndedAt)
			if !isInsideHole {
				continue
			}

			holePointCount++

			assert.Nil(t, point.ArchiveBytesPerSecond)
			assert.Nil(t, point.SourceWriteBytesPerSecond)
			assert.Nil(t, point.MaxUnarchivedWalBytes)
			assert.Nil(t, point.Restarts)
		}

		assert.Greater(t, holePointCount, 10)
	})
	t.Run("removes samples when the database is deleted", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 3, NewestAt: time.Now().UTC(), Modify: nil},
		)

		catalog.call(
			t,
			catalogCall{
				Request: func(ctx context.Context, client *apiclient.ClientWithResponses) (*http.Response, error) {
					return client.DeleteDatabase(ctx, catalog.DatabaseID)
				},
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNoContent,
			},
		)

		var remainingSamples int64
		catalog.App.QuerySQL(
			t,
			&remainingSamples,
			"SELECT count(*) FROM backups_physical_wal_stream_samples WHERE database_id = ?",
			catalog.DatabaseID,
		)
		assert.Zero(t, remainingSamples)
	})
	t.Run("returns the newest archive receive time when WAL segments are committed", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)
		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 2, NewestAt: time.Now().UTC(), Modify: nil},
		)

		olderSegment := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000010",
				StartLSN:    fixtureLSN(0x10 * segmentBytes),
				EndLSN:      fixtureLSN(0x11 * segmentBytes),
			},
		)
		olderSegment.ReceivedAt = time.Now().UTC().Add(-10 * time.Minute)
		catalog.insertWal(t, olderSegment)

		newerSegment := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000011",
				StartLSN:    fixtureLSN(0x11 * segmentBytes),
				EndLSN:      fixtureLSN(0x12 * segmentBytes),
			},
		)
		newerSegment.ReceivedAt = time.Now().UTC().Add(-2 * time.Minute)
		catalog.insertWal(t, newerSegment)

		response := getWalStreamStatistics(t, catalog, "HOUR")

		require.NotNil(t, response.Current)
		require.NotNil(t, response.Current.LastArchivedWalAt)
		assert.WithinDuration(t, newerSegment.ReceivedAt, *response.Current.LastArchivedWalAt, time.Millisecond)
	})
	t.Run("returns documented codes when streaming needs attention", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)

		catalog.insertSamples(
			t,
			samplesSeedSpec{
				DatabaseID: catalog.DatabaseID,
				Count:      10,
				NewestAt:   time.Now().UTC(),
				Modify: func(sample *apptest.PhysicalWalStreamSampleSeed) {
					sample.LocalQueueBytes = sample.LocalQueuePauseBytes
					sample.SlotSafeWalSizeBytes = nil
				},
			},
		)
		seedTestCompletedFullAtSegment(t, catalog, 0x10, time.Now().UTC().Add(-time.Hour))
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, time.Now().UTC().Add(-30*time.Minute))

		response := catalog.call(
			t,
			catalogCall{
				Request:       statisticsRequest(catalog.DatabaseID, "DAY"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
			},
		)

		var statistics statisticsResponse
		require.NoError(t, json.Unmarshal(response.Body, &statistics))

		assert.Equal(t, "NEEDS_ATTENTION", statistics.Health.Level)
		assert.Equal(t, []string{"LOCAL_QUEUE_FILLING"},
			getReasonCodes(statistics.Health.Reasons))
		assert.Contains(t, getNoticeCodes(statistics.Health.Notices), "NO_SLOT_WAL_LIMIT")
		require.NotEmpty(t, statistics.RestorableRanges, "the format check also covers the restorable ranges")

		var body any
		require.NoError(t, json.Unmarshal(response.Body, &body))
		assertOnlyCodesTimesAndPositionsAreStrings(t, "", body)
	})
	t.Run("reports when streaming stopped after the newest failed sample", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)

		catalog.insertStreamer(t, &apptest.PhysicalWalStreamerSeed{
			DatabaseID:      catalog.DatabaseID,
			StartedAt:       time.Now().UTC().Add(-time.Hour),
			LastHeartbeatAt: time.Now().UTC().Add(-10 * time.Minute),
			Status:          "FAILED",
			FailingSince:    new(time.Now().UTC().Add(-10 * time.Minute)),
		})

		newestSample := catalog.insertSamples(t, samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 5, NewestAt: time.Now().UTC().Add(-10 * time.Minute), Modify: func(sample *apptest.PhysicalWalStreamSampleSeed) {
			sample.LocalQueueBytes = sample.LocalQueuePauseBytes
		}})[4]

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Equal(t, "BROKEN", response.Health.Level)
		require.Equal(t, []string{"STREAMING_STOPPED"},
			getReasonCodes(response.Health.Reasons), "no other reason is read from a sample that old")
		require.NotNil(t, response.Health.Reasons[0].Since)
		assert.WithinDuration(t, newestSample.SampledAt, *response.Health.Reasons[0].Since, time.Millisecond)
		assert.Equal(t, "NOT_RUNNING", response.Streamer.State)
	})
	t.Run("reports a broken WAL chain until a newer full replaces it", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)
		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 4, NewestAt: time.Now().UTC(), Modify: nil},
		)

		seedCompletedFullWithWalGap(t, catalog)

		brokenResponse := getWalStreamStatistics(t, catalog, "HOUR")
		assert.Equal(t, "BROKEN", brokenResponse.Health.Level)
		assert.Equal(t, []string{"WAL_CHAIN_BROKEN"},
			getReasonCodes(brokenResponse.Health.Reasons), "a healthy slot does not hide a broken chain")

		newerFullStartLSN := fixtureLSN(0x20*segmentBytes + 0x28)
		newerFull := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   newerFullStartLSN,
				StopLSN:    newerFullStartLSN + 0x100,
			},
		)
		newerFull.CreatedAt = time.Now().UTC().Add(time.Second)
		catalog.insertFull(t, newerFull)

		repairedResponse := getWalStreamStatistics(t, catalog, "HOUR")
		assert.Equal(t, "HEALTHY", repairedResponse.Health.Level)
		assert.Empty(t, repairedResponse.Health.Reasons)
	})
	t.Run("keeps the broken verdict when an incremental fails on a WAL gap", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)
		catalog.insertSamples(
			t,
			samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 4, NewestAt: time.Now().UTC(), Modify: nil},
		)

		full := seedCompletedFullWithWalGap(t, catalog)

		incrementalStartLSN := fixtureLSN(0x14*segmentBytes + 0x28)
		brokenIncremental := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  incrementalStartLSN,
				StopLSN:                   incrementalStartLSN + 0x100,
			},
		)
		brokenIncremental.Status = "CHAIN_BROKEN"
		catalog.insertIncremental(t, brokenIncremental)

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Contains(t, getReasonCodes(response.Health.Reasons), "WAL_CHAIN_BROKEN")
		assert.NotContains(t, getNoticeCodes(response.Health.Notices), "NO_FULL_BACKUP_YET",
			"a chain broken by an incremental still has a completed full backup")
	})
	t.Run("restores to the newest WAL receive time when WAL follows the full without a gap", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		now := time.Now().UTC()

		seedTestCompletedFullAtSegment(t, catalog, 0x10, now.Add(-time.Hour))
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, now.Add(-30*time.Minute))
		newestSegment := seedTestWalSegmentReceivedAt(t, catalog, 0x11, now.Add(-time.Minute))

		response := getWalStreamStatistics(t, catalog, "HOUR")

		require.NotNil(t, response.LatestRestorableAt)
		assert.WithinDuration(t, newestSegment.ReceivedAt, *response.LatestRestorableAt, time.Millisecond)
		assert.Equal(t, time.UTC, response.LatestRestorableAt.Location())
	})
	t.Run("restores to the segment before a gap when the newest chain has a gap", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		now := time.Now().UTC()

		seedTestCompletedFullAtSegment(t, catalog, 0x10, now.Add(-time.Hour))
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, now.Add(-30*time.Minute))
		segmentBeforeGap := seedTestWalSegmentReceivedAt(t, catalog, 0x11, now.Add(-20*time.Minute))
		seedTestWalSegmentReceivedAt(t, catalog, 0x13, now.Add(-10*time.Minute))
		seedTestWalSegmentReceivedAt(t, catalog, 0x14, now.Add(-time.Minute))

		response := getWalStreamStatistics(t, catalog, "HOUR")

		require.NotNil(t, response.LatestRestorableAt)
		assert.WithinDuration(t, segmentBeforeGap.ReceivedAt, *response.LatestRestorableAt, time.Millisecond)
	})
	t.Run("restores to the newest WAL when an incremental completes after a gap", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		now := time.Now().UTC()

		full := seedTestCompletedFullAtSegment(t, catalog, 0x10, now.Add(-time.Hour))
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, now.Add(-50*time.Minute))
		seedTestWalSegmentReceivedAt(t, catalog, 0x11, now.Add(-40*time.Minute))

		incrementalStartLSN := fixtureLSN(0x13*segmentBytes + 0x28)
		incrementalAfterGap := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  incrementalStartLSN,
				StopLSN:                   incrementalStartLSN + 0x100,
			},
		)
		incrementalAfterGap.CompletedAt = new(now.Add(-20 * time.Minute))
		catalog.insertIncremental(t, incrementalAfterGap)

		seedTestWalSegmentReceivedAt(t, catalog, 0x13, now.Add(-15*time.Minute))
		newestSegment := seedTestWalSegmentReceivedAt(t, catalog, 0x14, now.Add(-time.Minute))

		response := getWalStreamStatistics(t, catalog, "HOUR")

		require.NotNil(t, response.LatestRestorableAt)
		assert.WithinDuration(t, newestSegment.ReceivedAt, *response.LatestRestorableAt, time.Millisecond,
			"a gap below the newest completed backup does not limit the latest restore")
	})
	t.Run("returns two healthy ranges when an incremental bridges a WAL gap", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)
		now := time.Now().UTC()
		catalog.insertSamples(t, samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 4, NewestAt: now, Modify: nil})

		full := seedTestCompletedFullAtSegment(t, catalog, 0x10, now.Add(-time.Hour))
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, now.Add(-50*time.Minute))
		segmentBeforeGap := seedTestWalSegmentReceivedAt(t, catalog, 0x11, now.Add(-40*time.Minute))
		incrementalAfterGap := seedTestCompletedIncrementalAtSegment(t, catalog, full, 0x13, now.Add(-20*time.Minute))
		seedTestWalSegmentReceivedAt(t, catalog, 0x13, now.Add(-15*time.Minute))
		newestSegment := seedTestWalSegmentReceivedAt(t, catalog, 0x14, now.Add(-time.Minute))

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Equal(t, "HEALTHY", response.Health.Level)
		assert.NotContains(t, getReasonCodes(response.Health.Reasons), "WAL_CHAIN_BROKEN",
			"a gap the latest restore crosses through a completed incremental does not break the chain")

		require.NotNil(t, response.LatestRestorableAt)
		assert.WithinDuration(t, newestSegment.ReceivedAt, *response.LatestRestorableAt, time.Millisecond)

		require.Len(t, response.RestorableRanges, 2, "the moments inside the gap cannot be restored")
		assert.WithinDuration(t, *full.CompletedAt, response.RestorableRanges[0].From, time.Millisecond)
		assert.WithinDuration(t, segmentBeforeGap.ReceivedAt, response.RestorableRanges[0].To, time.Millisecond)
		assert.WithinDuration(t, *incrementalAfterGap.CompletedAt, response.RestorableRanges[1].From, time.Millisecond)
		assert.WithinDuration(t, newestSegment.ReceivedAt, response.RestorableRanges[1].To, time.Millisecond)
		assert.Equal(t, time.UTC, response.RestorableRanges[0].From.Location())
	})
	t.Run("reports a broken chain when a gap follows the newest incremental", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)
		now := time.Now().UTC()
		catalog.insertSamples(t, samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 4, NewestAt: now, Modify: nil})

		full := seedTestCompletedFullAtSegment(t, catalog, 0x10, now.Add(-time.Hour))
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, now.Add(-50*time.Minute))
		seedTestCompletedIncrementalAtSegment(t, catalog, full, 0x11, now.Add(-45*time.Minute))
		segmentBeforeGap := seedTestWalSegmentReceivedAt(t, catalog, 0x11, now.Add(-40*time.Minute))
		seedTestWalSegmentReceivedAt(t, catalog, 0x13, now.Add(-time.Minute))

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Equal(t, "BROKEN", response.Health.Level)
		assert.Equal(t, []string{"WAL_CHAIN_BROKEN"},
			getReasonCodes(response.Health.Reasons))

		require.NotNil(t, response.LatestRestorableAt)
		assert.WithinDuration(t, segmentBeforeGap.ReceivedAt, *response.LatestRestorableAt, time.Millisecond)
		require.Len(t, response.RestorableRanges, 1)
		assert.WithinDuration(t, segmentBeforeGap.ReceivedAt, response.RestorableRanges[0].To, time.Millisecond)
	})
	t.Run("keeps a broken chain when the incremental after a gap fails", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		catalog.insertStreamer(
			t,
			&apptest.PhysicalWalStreamerSeed{
				DatabaseID:      catalog.DatabaseID,
				StartedAt:       time.Now().UTC().Add(-time.Hour),
				LastHeartbeatAt: time.Now().UTC(),
				Status:          "RUNNING",
			},
		)
		now := time.Now().UTC()
		catalog.insertSamples(t, samplesSeedSpec{DatabaseID: catalog.DatabaseID, Count: 4, NewestAt: now, Modify: nil})

		full := seedCompletedFullWithWalGap(t, catalog)

		incrementalStartLSN := fixtureLSN(0x13*segmentBytes + 0x28)
		failedIncremental := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  incrementalStartLSN,
				StopLSN:                   incrementalStartLSN + 0x100,
			},
		)
		failedIncremental.Status = "ERROR"
		catalog.insertIncremental(t, failedIncremental)

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Equal(t, "BROKEN", response.Health.Level)
		assert.Contains(t, getReasonCodes(response.Health.Reasons), "WAL_CHAIN_BROKEN",
			"a failed incremental does not bridge the gap")
		assert.Len(t, response.RestorableRanges, 1)
	})
	t.Run("restores to full completion when no WAL follows the full", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)

		full := seedTestCompletedFullAtSegment(t, catalog, 0x10, time.Now().UTC().Add(-time.Hour))

		response := getWalStreamStatistics(t, catalog, "HOUR")

		require.NotNil(t, response.LatestRestorableAt)
		assert.WithinDuration(t, *full.CompletedAt, *response.LatestRestorableAt, time.Millisecond)
	})
	t.Run("returns no restore point when no full backup completed", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)
		seedTestWalSegmentReceivedAt(t, catalog, 0x10, time.Now().UTC())

		response := catalog.call(
			t,
			catalogCall{
				Request:       statisticsRequest(catalog.DatabaseID, "HOUR"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusOK,
			},
		)

		var statistics statisticsResponse
		require.NoError(t, json.Unmarshal(response.Body, &statistics))

		assert.Nil(t, statistics.LatestRestorableAt)
		assert.Contains(t, string(response.Body), `"restorableRanges":[]`, "the list is empty, not null")
	})
	t.Run("reports a pending full when it is requested or running", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)

		assert.False(t, getWalStreamStatistics(t, catalog, "HOUR").Health.IsFullBackupPending)

		catalog.insertFull(
			t,
			newInProgressFull(
				inProgressFullSpec{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1},
			),
		)

		assert.True(t, getWalStreamStatistics(t, catalog, "HOUR").Health.IsFullBackupPending,
			"a running full backup is pending")

		catalog.App.ExecuteSQL(
			t,
			"DELETE FROM backups_physical WHERE status = 'IN_PROGRESS' AND database_id = ?",
			catalog.DatabaseID,
		)
		assert.False(t, getWalStreamStatistics(t, catalog, "HOUR").Health.IsFullBackupPending)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, "FULL"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusAccepted,
			},
		)

		assert.True(t, getWalStreamStatistics(t, catalog, "HOUR").Health.IsFullBackupPending,
			"a requested full backup is pending")
	})
	t.Run("reports a stopped streamer when its heartbeat is stale", func(t *testing.T) {
		t.Parallel()

		catalog := newStatisticsCatalog(t)

		catalog.insertStreamer(t, &apptest.PhysicalWalStreamerSeed{
			DatabaseID:      catalog.DatabaseID,
			StartedAt:       time.Now().UTC().Add(-time.Hour),
			LastHeartbeatAt: time.Now().UTC().Add(-2*time.Minute - time.Minute),
			Status:          "RUNNING",
		})

		response := getWalStreamStatistics(t, catalog, "HOUR")

		assert.Equal(t, "NOT_RUNNING", response.Streamer.State)
	})

	t.Run("keeps recorded series when streaming is no longer expected", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []struct {
			name          string
			stopStreaming func(*testing.T, *catalogFixture)
		}{
			{"reports not streaming when backups are disabled", func(t *testing.T, catalog *catalogFixture) {
				t.Helper()
				readConfig := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
				response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(t.Context(), catalog.DatabaseID, apiclient.BackupConfigInput{IsBackupsEnabled: false, StorageId: readConfig.StorageId, Encryption: readConfig.Encryption, Settings: readConfig.Settings})
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
			}},
			{"reports not streaming when the seeded database uses full and incremental backups", func(t *testing.T, catalog *catalogFixture) {
				t.Helper()
				catalog.App.ExecuteSQL(t, "UPDATE databases_postgresql_physical SET backup_type = 'FULL_INCREMENTAL' WHERE database_id = ?", catalog.DatabaseID)
			}},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				catalog := newStatisticsCatalog(t)
				catalog.insertStreamer(
					t,
					&apptest.PhysicalWalStreamerSeed{
						DatabaseID:      catalog.DatabaseID,
						StartedAt:       time.Now().UTC().Add(-time.Hour),
						LastHeartbeatAt: time.Now().UTC(),
						Status:          "RUNNING",
					},
				)
				catalog.insertSamples(
					t,
					samplesSeedSpec{
						DatabaseID: catalog.DatabaseID,
						Count:      20,
						NewestAt:   time.Now().UTC().Add(-3 * time.Hour),
					},
				)

				testCase.stopStreaming(t, catalog)
				response := getWalStreamStatistics(t, catalog, "DAY")
				assert.Equal(t, "NOT_STREAMING", response.Health.Level)

				pointsWithValues := 0
				for _, point := range response.Series.Points {
					if point.ArchiveBytesPerSecond != nil {
						pointsWithValues++
					}
				}
				assert.Positive(t, pointsWithValues, "samples recorded before streaming stopped stay in the series")
			})
		}
	})
}
