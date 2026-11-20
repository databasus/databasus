package physical

import "strings"

// Verbatim messages from PostgreSQL's basebackup_incremental.c and walsummarizer.c, matched by
// substring because the timeline and LSN arguments are interpolated per call.
const (
	walSummariesMissingStderr       = "WAL summaries are required on timeline"
	walSummarizationStalledStderr   = "WAL summarization is not progressing"
	walSummarizationDisabledStderr  = "unless WAL summarization is enabled"
	manifestTimelineNotInHistory    = "found in manifest, but not in this server's history"
	manifestTimelineRangeMismatched = "manifest requires WAL from"
)

// IsWalSummariesMissingError reports an incremental backup refused because the server has no WAL
// summaries for part of the range since the parent backup. The same message covers a range that
// was never summarized, one summarized with a hole and one whose summaries were removed.
func IsWalSummariesMissingError(stderr []byte) bool {
	return strings.Contains(string(stderr), walSummariesMissingStderr)
}

// IsWalSummarizationStalledError reports a summarizer that stopped absorbing WAL. PostgreSQL waits
// for summarization through the backup start LSN and gives up only after a minute without a single
// absorbed record, so this is a real stall rather than a slow cluster.
func IsWalSummarizationStalledError(stderr []byte) bool {
	return strings.Contains(string(stderr), walSummarizationStalledStderr)
}

// IsWalSummarizationDisabledError reports an incremental backup refused because summarize_wal is
// off on the server, as on a promoted node that never summarized WAL.
func IsWalSummarizationDisabledError(stderr []byte) bool {
	return strings.Contains(string(stderr), walSummarizationDisabledStderr)
}

// IsManifestTimelineRefusal reports an incremental backup refused because the parent manifest's
// timelines do not fit the server's timeline history.
func IsManifestTimelineRefusal(stderr []byte) bool {
	text := string(stderr)

	return strings.Contains(text, manifestTimelineNotInHistory) ||
		strings.Contains(text, manifestTimelineRangeMismatched)
}
