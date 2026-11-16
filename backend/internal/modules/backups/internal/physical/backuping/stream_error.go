package backuping

import (
	"fmt"
	"strings"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
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

// classifyIncrStreamError maps a failed INCREMENTAL. Across a promotion, a refusal tied to the
// parent manifest's timeline or to WAL summaries is recorded as a timeline switch, so the immediate
// replacement FULL anchors a new chain.
func classifyIncrStreamError(streamErr error, stderr []byte, isAcrossPromotion bool) streamOutcome {
	message := fmt.Sprintf("%v; stderr: %s", streamErr, truncateStderr(stderr))
	if strings.Contains(string(stderr), "could not parse backup manifest") {
		return outcomeManifestCorrupted(fmt.Errorf("%s", message), stderr)
	}

	if isAcrossPromotion && isPromotionRefusal(stderr) {
		reason := catalog.BackupErrorTimelineSwitchDetected

		return streamOutcome{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: message,
		}
	}

	if pgphysical.IsWalSummariesMissingError(stderr) {
		reason := catalog.BackupErrorSummariesExpired

		return streamOutcome{
			Status:       catalog.BackupStatusChainBroken,
			ErrorReason:  &reason,
			ErrorMessage: message,
		}
	}

	if pgphysical.IsWalSummarizationStalledError(stderr) {
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

func isPromotionRefusal(stderr []byte) bool {
	return pgphysical.IsWalSummariesMissingError(stderr) ||
		pgphysical.IsWalSummarizationDisabledError(stderr) ||
		pgphysical.IsManifestTimelineRefusal(stderr)
}
