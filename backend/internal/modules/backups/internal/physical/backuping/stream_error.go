package backuping

import (
	"fmt"
	"strings"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func classifyFullStreamError(streamErr error, stderr []byte) streamOutcome {
	reason := catalog.BackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr)),
	}
}

// Verbatim messages from PostgreSQL's basebackup_incremental.c and
// walsummarizer.c, matched by substring because the timeline and LSN arguments
// are interpolated per call.
const (
	summariesMissingForRangeStderr = "WAL summaries are required on timeline"
	summarizerStalledStderr        = "WAL summarization is not progressing"
)

func classifyIncrStreamError(streamErr error, stderr []byte) streamOutcome {
	message := fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr))
	if strings.Contains(string(stderr), "could not parse backup manifest") {
		return outcomeManifestCorrupted(fmt.Errorf("%s", message), stderr)
	}

	if isSummariesExpiredError(stderr) {
		reason := catalog.BackupErrorSummariesExpired

		return streamOutcome{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: message,
		}
	}

	if isSummarizerStalledError(stderr) {
		reason := catalog.BackupErrorSummarizerFallingBehind

		return streamOutcome{
			Status:       catalog.BackupStatusError,
			ErrorReason:  &reason,
			ErrorMessage: message,
		}
	}

	reason := catalog.BackupErrorPgBasebackupFailed

	return streamOutcome{
		Status:       catalog.BackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: message,
	}
}

// isSummariesExpiredError detects the post-readiness-check race where the source cluster
// pruned WAL summaries between CheckSummarizerReadiness returning OK and pg_basebackup
// --incremental opening the actual range. The same message covers a range that was never
// summarized at all and one summarized with a hole in the middle.
func isSummariesExpiredError(stderr []byte) bool {
	return strings.Contains(string(stderr), summariesMissingForRangeStderr)
}

// isSummarizerStalledError detects a summarizer that stopped absorbing WAL.
// PostgreSQL waits for summarization through the backup start LSN and gives up
// only after a minute without a single absorbed record, so this is a real stall
// rather than a slow cluster.
func isSummarizerStalledError(stderr []byte) bool {
	return strings.Contains(string(stderr), summarizerStalledStderr)
}
