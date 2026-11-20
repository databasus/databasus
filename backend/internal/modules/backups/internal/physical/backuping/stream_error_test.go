package backuping

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// The stderr fixtures below carry PostgreSQL's own wording. Paraphrasing one would
// let the test pass while the real message went unmatched in production.

func Test_ClassifyIncrStreamError_WhenSummariesMissingForRange_ReturnsChainBrokenSummariesExpired(t *testing.T) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  WAL summaries are " +
		"required on timeline 1 from 0/1000000 to 0/2000000, but no summaries for that timeline and " +
		"LSN range exist")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr, false)

	require.Equal(t, catalog.BackupStatusChainBroken, outcome.Status)
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, catalog.BackupErrorSummariesExpired, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenSummariesIncompleteForRange_ReturnsChainBrokenSummariesExpired(t *testing.T) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  WAL summaries are " +
		"required on timeline 1 from 0/1000000 to 0/2000000, but the summaries for that timeline and " +
		"LSN range are incomplete")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr, false)

	require.Equal(t, catalog.BackupStatusChainBroken, outcome.Status)
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, catalog.BackupErrorSummariesExpired, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenSummarizationNotProgressing_ReturnsErrorSummarizerFallingBehind(t *testing.T) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  WAL summarization " +
		"is not progressing")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr, false)

	require.Equal(t, catalog.BackupStatusError, outcome.Status,
		"a stalled summarizer is transient, so the chain must stay extendable")
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, catalog.BackupErrorSummarizerFallingBehind, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenUnrelatedStderr_ReturnsErrorPgBasebackupFailed(t *testing.T) {
	outcome := classifyIncrStreamError(errors.New("exit status 1"),
		[]byte("pg_basebackup: error: connection to server failed"), true)

	require.Equal(t, catalog.BackupStatusError, outcome.Status)
	require.NotNil(t, outcome.ErrorReason)
	require.Equal(t, catalog.BackupErrorPgBasebackupFailed, *outcome.ErrorReason)
}

func Test_ClassifyIncrStreamError_WhenRefusedAcrossPromotion_ReturnsChainBrokenTimelineSwitch(t *testing.T) {
	refusals := map[string]string{
		"summaries missing": "pg_basebackup: error: could not initiate base backup: ERROR:  WAL summaries are " +
			"required on timeline 2 from 0/7000000 to 0/9000028, but no summaries for that timeline and " +
			"LSN range exist",
		"summarization disabled": "pg_basebackup: error: could not initiate base backup: ERROR:  incremental " +
			"backups cannot be taken unless WAL summarization is enabled",
		"manifest timeline": "pg_basebackup: error: could not initiate base backup: ERROR:  timeline 1 found " +
			"in manifest, but not in this server's history",
	}

	for name, stderr := range refusals {
		t.Run(name, func(t *testing.T) {
			outcome := classifyIncrStreamError(errors.New("exit status 1"), []byte(stderr), true)

			require.Equal(t, catalog.BackupStatusChainBroken, outcome.Status)
			require.Equal(t, catalog.BackupErrorTimelineSwitchDetected, *outcome.ErrorReason)
		})
	}
}

func Test_ClassifyIncrStreamError_WhenManifestTimelineRefusedOnSameTimeline_ReturnsErrorPgBasebackupFailed(
	t *testing.T,
) {
	stderr := []byte("pg_basebackup: error: could not initiate base backup: ERROR:  timeline 1 found " +
		"in manifest, but not in this server's history")

	outcome := classifyIncrStreamError(errors.New("exit status 1"), stderr, false)

	require.Equal(t, catalog.BackupStatusError, outcome.Status)
	require.Equal(t, catalog.BackupErrorPgBasebackupFailed, *outcome.ErrorReason)
}
