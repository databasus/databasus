package walstreaming

import (
	"cmp"
	"path/filepath"
	"slices"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

type queuedWalSegment struct {
	walFilename string
	localPath   string
	timelineID  int
	startLSN    walmath.LSN
	endLSN      walmath.LSN
	isComplete  bool
}

func listQueuedWalSegments(watchDir string, segmentSizeBytes int64) []queuedWalSegment {
	queuedSegments := listQueuedWalSegmentsIn(watchDir, segmentSizeBytes)

	return append(
		queuedSegments,
		listQueuedWalSegmentsIn(filepath.Join(watchDir, pendingUploadDirName), segmentSizeBytes)...)
}

func listQueuedWalSegmentsIn(directory string, segmentSizeBytes int64) []queuedWalSegment {
	segments := pgphysical.GetWalSegmentFiles(directory, segmentSizeBytes)
	queuedSegments := make([]queuedWalSegment, 0, len(segments))
	for _, segment := range segments {
		if !walmath.IsWalFilename(segment.Filename, walmath.DefaultSegmentSizeBytes) {
			continue
		}
		queuedSegments = append(queuedSegments, queuedWalSegment{
			walFilename: segment.Filename, localPath: segment.LocalPath, timelineID: segment.TimelineID,
			startLSN: segment.StartLSN, endLSN: segment.EndLSN, isComplete: segment.IsComplete,
		})
	}

	return queuedSegments
}

func compareWalOrder(firstTimelineID int, firstLSN walmath.LSN, secondTimelineID int, secondLSN walmath.LSN) int {
	return cmp.Or(cmp.Compare(firstTimelineID, secondTimelineID), cmp.Compare(firstLSN, secondLSN))
}

func sortQueuedWalSegments(queuedSegments []queuedWalSegment) {
	slices.SortStableFunc(queuedSegments, func(first, second queuedWalSegment) int {
		return compareWalOrder(first.timelineID, first.startLSN, second.timelineID, second.startLSN)
	})
}
