package usecases_physical_postgresql

import (
	"cmp"
	"path/filepath"
	"slices"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

// A finalized segment waiting in the watch directory or in pending-upload/, including one whose
// upload is running: its file stays in place until the upload commits.
type queuedWalSegment struct {
	walFilename string
	localPath   string
	timelineID  int
	startLSN    walmath.LSN
	endLSN      walmath.LSN
	isComplete  bool
}

// The watch directory is listed before pending-upload/. realignResumePath only moves files from the
// first into the second, so a file renamed between the two reads is still found in the second.
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

// Timeline first, then position: the same order FindLatestCommittedSegment uses, so the oldest WAL
// drains first and the archive progress point moves forward.
func compareWalOrder(firstTimelineID int, firstLSN walmath.LSN, secondTimelineID int, secondLSN walmath.LSN) int {
	return cmp.Or(cmp.Compare(firstTimelineID, secondTimelineID), cmp.Compare(firstLSN, secondLSN))
}

func sortQueuedWalSegments(queuedSegments []queuedWalSegment) {
	slices.SortStableFunc(queuedSegments, func(first, second queuedWalSegment) int {
		return compareWalOrder(first.timelineID, first.startLSN, second.timelineID, second.startLSN)
	})
}
