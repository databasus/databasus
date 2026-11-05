package upgrade

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

const physicalWALSegmentsPerLogID = (1 << 32) / (16 * 1024 * 1024)

type physicalBackupSnapshot struct {
	RootFullBackupID          string
	ParentIncrementalBackupID string
	ID                        string
	Type                      string
	Status                    string
	TimelineID                int
	StartLSN                  string
	StopLSN                   string
	FromWalFilename           string
	ToWalFilename             string
}

func physicalFullBackupItem(backup physicalBackupSnapshot) upgradecheck.Item {
	return upgradecheck.Item{Kind: "physical full backup", Name: backup.ID, Values: map[string]string{
		"status":     backup.Status,
		"timelineId": strconv.Itoa(backup.TimelineID),
		"startLsn":   backup.StartLSN,
		"stopLsn":    backup.StopLSN,
	}}
}

func physicalIncrementalBackupItem(backup physicalBackupSnapshot) upgradecheck.Item {
	return upgradecheck.Item{Kind: "physical incremental backup", Name: backup.ID, Values: map[string]string{
		"status": backup.Status, "timelineId": strconv.Itoa(backup.TimelineID),
		"startLsn": backup.StartLSN, "stopLsn": backup.StopLSN,
		"rootFullBackupId": backup.RootFullBackupID, "parentIncrementalBackupId": backup.ParentIncrementalBackupID,
	}}
}

func physicalWALSegmentItems(backup physicalBackupSnapshot) ([]upgradecheck.Item, error) {
	timelineID, first, err := parsePhysicalWALFilename(backup.FromWalFilename)
	if err != nil {
		return nil, err
	}

	lastTimelineID, last, err := parsePhysicalWALFilename(backup.ToWalFilename)
	if err != nil {
		return nil, err
	}

	if timelineID != lastTimelineID || first > last || last-first > 1024 {
		return nil, fmt.Errorf("invalid WAL range %q to %q", backup.FromWalFilename, backup.ToWalFilename)
	}

	items := []upgradecheck.Item{}
	for segment := first; segment <= last; segment++ {
		items = append(
			items,
			upgradecheck.Item{
				Kind: "physical WAL segment",
				Name: fmt.Sprintf(
					"%08X%08X%08X",
					timelineID,
					segment/physicalWALSegmentsPerLogID,
					segment%physicalWALSegmentsPerLogID,
				),
				Values: map[string]string{"status": backup.Status, "timelineId": strconv.Itoa(backup.TimelineID)},
			},
		)
	}
	return items, nil
}

func parsePhysicalWALFilename(filename string) (uint64, uint64, error) {
	if len(filename) != 24 {
		return 0, 0, fmt.Errorf("invalid WAL filename %q", filename)
	}

	timelineID, err := strconv.ParseUint(filename[:8], 16, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("parse WAL timeline: %w", err)
	}

	logID, err := strconv.ParseUint(filename[8:16], 16, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("parse WAL log: %w", err)
	}

	segmentID, err := strconv.ParseUint(filename[16:], 16, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("parse WAL segment: %w", err)
	}

	if segmentID >= physicalWALSegmentsPerLogID {
		return 0, 0, fmt.Errorf("invalid WAL segment in filename %q", filename)
	}

	return timelineID, logID*physicalWALSegmentsPerLogID + segmentID, nil
}

func TestPhysicalWALSegmentItems(t *testing.T) {
	t.Parallel()

	t.Run("lists consecutive segments across a log boundary with 16 MiB WAL segments", func(t *testing.T) {
		t.Parallel()
		segments, err := physicalWALSegmentItems(physicalBackupSnapshot{
			Status: "COMPLETED", TimelineID: 1,
			FromWalFilename: "0000000100000000000000FF",
			ToWalFilename:   "000000010000000100000001",
		})

		require.NoError(t, err)
		segmentNames := make([]string, 0, len(segments))
		for _, segment := range segments {
			segmentNames = append(segmentNames, segment.Name)
		}
		assert.Equal(t, []string{
			"0000000100000000000000FF",
			"000000010000000100000000",
			"000000010000000100000001",
		}, segmentNames)
	})

	t.Run("rejects an invalid segment number with 16 MiB WAL segments", func(t *testing.T) {
		t.Parallel()
		_, err := physicalWALSegmentItems(physicalBackupSnapshot{
			FromWalFilename: "000000010000000000000100",
			ToWalFilename:   "000000010000000000000100",
		})

		require.Error(t, err)
	})
}
