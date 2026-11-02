package restoring_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPhysicalRestoreWalScope(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name   string
		offset fixtureLSN
	}{
		{name: "excludes the segment ending at the full start when the full is segment aligned", offset: 0},
		{name: "includes its boundary segment and ignores other timelines and databases when the full starts inside a segment", offset: 0x28},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			timeline := newRestoreTimeline()
			full := catalog.seedRestoreFull(
				t,
				restoreFullSpec{
					StartLSN:    lsnAt(4) + testCase.offset,
					StopLSN:     lsnAt(4) + testCase.offset + 0x100,
					CompletedAt: timeline.at(0),
				},
			)

			for _, segment := range []int{3, 4, 5} {
				catalog.seedRestoreWal(t, restoreWalSpec{Segment: segment, ReceivedAt: timeline.at(segment)})
			}

			otherDatabaseID := catalog.App.InsertPhysicalDatabase(t, catalog.WorkspaceID)
			otherTimeline := newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  2,
					WalFilename: "000000020000000000000004",
					StartLSN:    lsnAt(4) + testCase.offset/2,
					EndLSN:      lsnAt(5),
				},
			)
			catalog.insertWal(t, otherTimeline)
			foreign := newWalSeed(
				walSeedSpec{
					DatabaseID:  otherDatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "000000010000000000000004",
					StartLSN:    lsnAt(4) + testCase.offset/2,
					EndLSN:      lsnAt(5),
				},
			)
			catalog.insertWal(t, foreign)

			entries := catalog.downloadRestore(t, restoreRequest{})
			assertRestoreArtifacts(t, entries, full.ID, nil, 4, 5)
			for _, segment := range []int{4, 5} {
				assert.Equal(
					t,
					fmt.Sprintf("stored WAL 0000000100000000%08X", segment),
					entries[fmt.Sprintf("wal/0000000100000000%08X.zst", segment)],
				)
			}
		})
	}
}
