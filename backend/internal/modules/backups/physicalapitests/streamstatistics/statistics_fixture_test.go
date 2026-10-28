package streamstatistics_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func getWalStreamStatistics(
	t *testing.T,
	catalog *catalogFixture,
	period string,
) *statisticsResponse {
	t.Helper()

	var response statisticsResponse

	catalog.call(
		t,
		catalogCall{
			Request:       statisticsRequest(catalog.DatabaseID, period),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusOK,
			Destination:   &response,
		},
	)

	return &response
}

func statisticsRequest(databaseID uuid.UUID, period string) apptest.PhysicalRequest {
	return apptest.PhysicalStatisticsRequest(databaseID, period)
}

func getReasonCodes(reasons []healthReason) []string {
	codes := make([]string, 0, len(reasons))

	for _, reason := range reasons {
		codes = append(codes, reason.Code)
	}

	return codes
}

func getNoticeCodes(notices []healthNotice) []string {
	codes := make([]string, 0, len(notices))

	for _, notice := range notices {
		codes = append(codes, notice.Code)
	}

	return codes
}

func assertOnlyCodesTimesAndPositionsAreStrings(t *testing.T, fieldName string, value any) {
	t.Helper()

	documentedCodes := map[string][]string{
		"level": {
			string("NOT_STREAMING"), string("COLLECTING_DATA"),
			string("BROKEN"), string("NEEDS_ATTENTION"),
			string("HEALTHY"), string("INFO"),
		},
		"code": {
			string("STREAMING_STOPPED"), string("SOURCE_UNREACHABLE"),
			string("WAL_CHAIN_BROKEN"), string("SLOT_MISSING"),
			string("SLOT_LOST"), string("ARCHIVE_NOT_ADVANCING"),
			string("ARCHIVE_SLOWER_THAN_SOURCE"), string("SLOT_KEEPS_EXTRA_WAL"),
			string("SLOT_LIMIT_APPROACHING"), string("NEAR_SLOT_REBUILD"),
			string("LOCAL_QUEUE_FILLING"), string("FREQUENT_RESTARTS"),
			string("FORCED_ROTATION_REFUSED"), string("NO_SLOT_WAL_LIMIT"),
			"SLOT_KEEPS_EXTRA_WAL_STEADILY", string("NO_FULL_BACKUP_YET"),
		},
		"state": {
			string("RUNNING"), string("NOT_RUNNING"),
			string("NOT_STARTED"), string("UNKNOWN"),
			string("FALLING_BEHIND"), string("CATCHING_UP"),
			string("IDLE"), string("KEEPING_UP"),
		},
		"sourceState": {
			string("REACHABLE"),
			string("UNREACHABLE"),
			string("SLOT_MISSING"),
		},
		"slotWalStatus": {"reserved", "extended", "unreserved", "lost"},
		"period":        {"HOUR", "DAY", "WEEK"},
	}

	switch typedValue := value.(type) {
	case map[string]any:
		for nestedFieldName, nestedValue := range typedValue {
			assertOnlyCodesTimesAndPositionsAreStrings(t, nestedFieldName, nestedValue)
		}

	case []any:
		for _, nestedValue := range typedValue {
			assertOnlyCodesTimesAndPositionsAreStrings(t, fieldName, nestedValue)
		}

	case string:
		if allowedCodes, isCodeField := documentedCodes[fieldName]; isCodeField {
			assert.Contains(t, allowedCodes, typedValue, "field %s holds an undocumented code", fieldName)

			return
		}

		if strings.HasSuffix(fieldName, "Lsn") {
			assert.Regexp(t, `^[0-9A-F]+/[0-9A-F]+$`, typedValue, "field %s holds a WAL position", fieldName)

			return
		}

		_, err := time.Parse(time.RFC3339Nano, typedValue)
		assert.NoError(t, err, "field %s holds text that is neither a code, a time nor a position", fieldName)
	}
}

func seedCompletedFullWithWalGap(t *testing.T, catalog *catalogFixture) *apptest.PhysicalFullBackupSeed {
	t.Helper()

	fullStartLSN := fixtureLSN(0x10*segmentBytes + 0x28)
	full := catalog.insertFull(
		t,
		newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fullStartLSN,
				StopLSN:    fullStartLSN + 0x100,
			},
		),
	)

	for _, segmentNo := range []uint64{0x10, 0x11, 0x13, 0x14} {
		catalog.insertWal(
			t,
			newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: fixtureSegmentNo(segmentNo).GetFilename(1, segmentBytes),
					StartLSN:    fixtureLSN(segmentNo * segmentBytes),
					EndLSN:      fixtureLSN((segmentNo + 1) * segmentBytes),
				},
			),
		)
	}

	return full
}

func seedTestCompletedFullAtSegment(
	t *testing.T,
	catalog *catalogFixture,
	segmentNo uint64,
	completedAt time.Time,
) *apptest.PhysicalFullBackupSeed {
	t.Helper()

	startLSN := fixtureLSN(segmentNo*segmentBytes + 0x28)
	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StartLSN:   startLSN,
			StopLSN:    startLSN + 0x100,
		},
	)
	full.CompletedAt = &completedAt

	return catalog.insertFull(t, full)
}

func seedTestCompletedIncrementalAtSegment(
	t *testing.T,
	catalog *catalogFixture,
	full *apptest.PhysicalFullBackupSeed,
	segmentNo uint64,
	completedAt time.Time,
) *apptest.PhysicalIncrementalBackupSeed {
	t.Helper()

	startLSN := fixtureLSN(segmentNo*segmentBytes + 0x28)
	incremental := newIncrementalSeed(
		incrementalSeedSpec{
			DatabaseID:                catalog.DatabaseID,
			StorageID:                 catalog.StorageID,
			RootFullBackupID:          full.ID,
			ParentIncrementalBackupID: nil,
			TimelineID:                1,
			StartLSN:                  startLSN,
			StopLSN:                   startLSN + 0x100,
		},
	)
	incremental.CompletedAt = &completedAt

	return catalog.insertIncremental(t, incremental)
}

func seedTestWalSegmentReceivedAt(
	t *testing.T,
	catalog *catalogFixture,
	segmentNo uint64,
	receivedAt time.Time,
) *apptest.PhysicalWalSegmentSeed {
	t.Helper()

	segment := newWalSeed(
		walSeedSpec{
			DatabaseID:  catalog.DatabaseID,
			StorageID:   catalog.StorageID,
			TimelineID:  1,
			WalFilename: fixtureSegmentNo(segmentNo).GetFilename(1, segmentBytes),
			StartLSN:    fixtureLSN(segmentNo * segmentBytes),
			EndLSN:      fixtureLSN((segmentNo + 1) * segmentBytes),
		},
	)
	segment.ReceivedAt = receivedAt

	return catalog.insertWal(t, segment)
}
