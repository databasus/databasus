package verifications

func isAgentSideFailure(reason FailureReason) bool {
	return reason == FailureReasonAgentLostContact ||
		reason == FailureReasonAgentRemoved ||
		reason == FailureReasonAgentSetupFailed ||
		reason == FailureReasonAgentDroppedJob
}

func shouldRequeueAgentSide(reason FailureReason, attemptCount int) bool {
	return isAgentSideFailure(reason) && attemptCount < MaxAgentSideAttempts
}

func classifyAgentReport(req *ReportRequest) FailureReason {
	if req.FailureKind != nil && *req.FailureKind == string(FailureReasonDiskLimitExceeded) {
		return FailureReasonDiskLimitExceeded
	}

	if req.PgRestoreExitCode != nil {
		return FailureReasonBackupRejected
	}

	return FailureReasonAgentSetupFailed
}

func failureMessageFor(reason FailureReason, agentMessage *string) string {
	hasAgentMessage := agentMessage != nil && *agentMessage != ""

	switch reason {
	case FailureReasonAgentSetupFailed:
		if hasAgentMessage {
			return *agentMessage
		}

		return "agent failed before pg_restore ran (download, container, or OOM) without diagnostic detail"
	case FailureReasonBackupRejected:
		if hasAgentMessage {
			return *agentMessage
		}

		return "pg_restore rejected the backup without diagnostic detail"
	case FailureReasonDiskLimitExceeded:
		if hasAgentMessage {
			return *agentMessage
		}

		return "restore exceeded the per-job disk budget"
	case FailureReasonRestoredTooSmall:
		if hasAgentMessage {
			return *agentMessage
		}

		return "restored database is smaller than expected vs the original raw size"
	case FailureReasonAgentLostContact:
		return "verification stalled: agent went silent"
	case FailureReasonAgentRemoved:
		return "verification failed: the owning agent was removed before the run finished"
	case FailureReasonAgentDroppedJob:
		return "verification was dropped by its still-online agent before it reported a result (likely a report-delivery failure or an agent restart mid-run)"
	case FailureReasonUnclaimedTooLong:
		return "verification was not picked up by any agent in time"
	default:
		return "verification failed"
	}
}

type FailureReason string

// Agent-side reasons are retried up to MaxAgentSideAttempts because the same
// input may succeed on the next agent. Backup-side reasons go terminal on the
// first failure — retrying the same dump produces the same rejection and only
// delays the user notification.
const (
	FailureReasonAgentLostContact  FailureReason = "AGENT_LOST_CONTACT"
	FailureReasonAgentRemoved      FailureReason = "AGENT_REMOVED"
	FailureReasonAgentSetupFailed  FailureReason = "AGENT_SETUP_FAILED"
	FailureReasonAgentDroppedJob   FailureReason = "AGENT_DROPPED_JOB"
	FailureReasonBackupRejected    FailureReason = "BACKUP_REJECTED"
	FailureReasonDiskLimitExceeded FailureReason = "DISK_LIMIT_EXCEEDED"
	FailureReasonRestoredTooSmall  FailureReason = "RESTORED_TOO_SMALL"
	FailureReasonUnclaimedTooLong  FailureReason = "UNCLAIMED_TOO_LONG"
)
