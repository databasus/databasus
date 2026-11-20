package chains

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func TestBuildTimelineLine(t *testing.T) {
	t.Parallel()

	storageID := uuid.New()
	otherStorageID := uuid.New()
	const firstFork walmath.LSN = 0x3000148
	const secondFork walmath.LSN = 0x7000060
	const segmentBoundaryFork walmath.LSN = 0x5000000

	scenarios := []struct {
		name           string
		histories      []*catalog.WalHistoryFile
		full           *catalog.Backup
		expectedRanges []TimelineLineRange
		isOffLine      bool
	}{
		{
			name:      "crosses one fork point from a timeline 1 full",
			histories: []*catalog.WalHistoryFile{newForkHistory(storageID, 2, 1, firstFork)},
			full:      newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{
				{TimelineID: 1, StartLSN: 0x2000028, EndLSN: firstFork},
				{TimelineID: 2, StartLSN: firstFork, EndLSN: catalog.LSNMax},
			},
		},
		{
			name: "crosses two fork points from a timeline 1 full",
			histories: []*catalog.WalHistoryFile{
				newForkHistory(storageID, 2, 1, firstFork),
				newForkHistory(storageID, 3, 2, secondFork),
			},
			full: newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{
				{TimelineID: 1, StartLSN: 0x2000028, EndLSN: firstFork},
				{TimelineID: 2, StartLSN: firstFork, EndLSN: secondFork},
				{TimelineID: 3, StartLSN: secondFork, EndLSN: catalog.LSNMax},
			},
		},
		{
			name: "starts on the full's timeline between two fork points",
			histories: []*catalog.WalHistoryFile{
				newForkHistory(storageID, 2, 1, firstFork),
				newForkHistory(storageID, 3, 2, secondFork),
			},
			full: newLineFull(storageID, 2, 0x4000028),
			expectedRanges: []TimelineLineRange{
				{TimelineID: 2, StartLSN: 0x4000028, EndLSN: secondFork},
				{TimelineID: 3, StartLSN: secondFork, EndLSN: catalog.LSNMax},
			},
		},
		{
			name:      "splits the line exactly at a fork point on a segment boundary",
			histories: []*catalog.WalHistoryFile{newForkHistory(storageID, 2, 1, segmentBoundaryFork)},
			full:      newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{
				{TimelineID: 1, StartLSN: 0x2000028, EndLSN: segmentBoundaryFork},
				{TimelineID: 2, StartLSN: segmentBoundaryFork, EndLSN: catalog.LSNMax},
			},
		},
		{
			name: "leaves a full off the line when a higher sibling does not descend from it",
			histories: []*catalog.WalHistoryFile{
				newForkHistory(storageID, 2, 1, firstFork),
				newForkHistory(storageID, 3, 1, secondFork),
			},
			full:           newLineFull(storageID, 2, 0x4000028),
			expectedRanges: []TimelineLineRange{{TimelineID: 2, StartLSN: 0x4000028, EndLSN: catalog.LSNMax}},
			isOffLine:      true,
		},
		{
			name: "follows the higher sibling from the shared parent timeline",
			histories: []*catalog.WalHistoryFile{
				newForkHistory(storageID, 2, 1, firstFork),
				newForkHistory(storageID, 3, 1, secondFork),
			},
			full: newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{
				{TimelineID: 1, StartLSN: 0x2000028, EndLSN: secondFork},
				{TimelineID: 3, StartLSN: secondFork, EndLSN: catalog.LSNMax},
			},
		},
		{
			name: "stops at a conflicting link",
			histories: []*catalog.WalHistoryFile{
				withConflict(newForkHistory(storageID, 2, 1, firstFork)),
			},
			full:           newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{{TimelineID: 1, StartLSN: 0x2000028, EndLSN: catalog.LSNMax}},
		},
		{
			name:           "stops at a link published without a fork point",
			histories:      []*catalog.WalHistoryFile{newLegacyHistory(storageID, 2)},
			full:           newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{{TimelineID: 1, StartLSN: 0x2000028, EndLSN: catalog.LSNMax}},
		},
		{
			name:           "stops at a link in another storage",
			histories:      []*catalog.WalHistoryFile{newForkHistory(otherStorageID, 2, 1, firstFork)},
			full:           newLineFull(storageID, 1, 0x2000028),
			expectedRanges: []TimelineLineRange{{TimelineID: 1, StartLSN: 0x2000028, EndLSN: catalog.LSNMax}},
		},
		{
			name:           "leaves a full started after the fork point off the line",
			histories:      []*catalog.WalHistoryFile{newForkHistory(storageID, 2, 1, firstFork)},
			full:           newLineFull(storageID, 1, 0x4000028),
			expectedRanges: []TimelineLineRange{{TimelineID: 1, StartLSN: 0x4000028, EndLSN: catalog.LSNMax}},
			isOffLine:      true,
		},
		{
			name:           "keeps a full ahead of the line on its own timeline while its history is pending",
			histories:      []*catalog.WalHistoryFile{},
			full:           newLineFull(storageID, 2, 0x4000028),
			expectedRanges: []TimelineLineRange{{TimelineID: 2, StartLSN: 0x4000028, EndLSN: catalog.LSNMax}},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			line := BuildTimelineLine(scenario.histories, scenario.full)

			require.Equal(t, scenario.expectedRanges, line.Ranges)
			require.Equal(t, scenario.isOffLine, line.IsOffLine)
		})
	}
}

func newLineFull(storageID uuid.UUID, timelineID int, startLSN walmath.LSN) *catalog.Backup {
	return &catalog.Backup{
		ID:         uuid.New(),
		Kind:       catalog.BackupKindFull,
		StorageID:  storageID,
		TimelineID: timelineID,
		StartLSN:   &startLSN,
	}
}

func newForkHistory(
	storageID uuid.UUID,
	timelineID int,
	parentTimelineID int,
	forkLSN walmath.LSN,
) *catalog.WalHistoryFile {
	history := newLegacyHistory(storageID, timelineID)
	history.ParentTimelineID = &parentTimelineID
	history.ForkLSN = &forkLSN

	return history
}

func newLegacyHistory(storageID uuid.UUID, timelineID int) *catalog.WalHistoryFile {
	return &catalog.WalHistoryFile{
		ID:              uuid.New(),
		StorageID:       storageID,
		TimelineID:      timelineID,
		HistoryFilename: walmath.FormatHistoryFilename(uint32(timelineID)),
	}
}

func withConflict(history *catalog.WalHistoryFile) *catalog.WalHistoryFile {
	history.IsConflicting = true

	return history
}
