package verification_runs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	databases_bridge "databasus-backend/internal/features/databases"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
	"databasus-backend/internal/storage"
)

type VerificationService struct {
	verificationRepository *VerificationRepository
	databaseAccess         *databases_bridge.Access
	// getDatabases and getBackups are read per call: each test application installs its own modules.
	getDatabases     func() databases.Module
	getBackups       func() backups.Module
	configService    *verification_config.VerificationConfigService
	notifierService  NotificationSender
	workspaceService *workspaces_bridge.Access
	auditLogService  *audit_logs.AuditLogService
	logger           *slog.Logger
}

func (s *VerificationService) EnqueueManualVerification(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
) (*RestoreVerification, error) {
	backup, isFound, err := s.getLogicalBackup(ctx, backupID)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, errors.New("backup not found")
	}

	database, err := s.databaseAccess.GetDatabaseByID(ctx, backup.DatabaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to trigger verification for this database")
	}

	if err := s.guardBackupIsVerifiable(backup); err != nil {
		return nil, err
	}

	var created *RestoreVerification

	err = storage.GetDb().Transaction(func(tx *gorm.DB) error {
		existing, txErr := s.verificationRepository.FindNonTerminalForDatabase(tx, backup.DatabaseID)
		if txErr != nil {
			return txErr
		}

		for _, row := range existing {
			if row.Trigger == VerificationTriggerManual {
				return errors.New(
					"a manual verification is already scheduled for this database, please cancel it first",
				)
			}
		}

		verification := &RestoreVerification{
			ID:           uuid.New(),
			DatabaseID:   backup.DatabaseID,
			BackupID:     backup.ID,
			Trigger:      VerificationTriggerManual,
			Status:       VerificationStatusPending,
			AttemptCount: 1,
			CreatedAt:    time.Now().UTC(),
		}

		if createErr := tx.Create(verification).Error; createErr != nil {
			return createErr
		}

		created = verification
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("manual backup verification enqueued for database %q", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &database.WorkspaceID,
	})

	return created, nil
}

func (s *VerificationService) OnLogicalBackupCompleted(
	ctx context.Context,
	event backups.LogicalBackupCompleted,
) error {
	return s.scheduleVerificationForCompletedBackup(ctx, event.DatabaseID)
}

func (s *VerificationService) CancelVerification(
	ctx context.Context,
	actor users.Actor,
	verificationID uuid.UUID,
) error {
	verification, err := s.verificationRepository.FindByID(ctx, verificationID)
	if err != nil {
		return err
	}

	if verification == nil {
		return errors.New("verification not found")
	}

	database, err := s.databaseAccess.GetDatabaseByID(ctx, verification.DatabaseID)
	if err != nil {
		return err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionEditContent)
	if err != nil {
		return err
	}
	if !isAllowed {
		return errors.New("insufficient permissions to cancel verification for this database")
	}

	if verification.Status != VerificationStatusPending && verification.Status != VerificationStatusRunning {
		return fmt.Errorf("verification is not cancellable (status: %s)", verification.Status)
	}

	if err := s.verificationRepository.MarkTerminal(nil, verificationID, VerificationStatusCanceled, map[string]any{
		"fail_message": cancelMessageByUser,
	}); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("backup verification canceled for database %q", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: &database.WorkspaceID,
	})

	return nil
}

func (s *VerificationService) GetVerificationsByDatabaseID(
	ctx context.Context,
	actor users.Actor,
	databaseID uuid.UUID,
	limit, offset int,
) (*GetVerificationsResponse, error) {
	database, err := s.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to view verifications for this database")
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	verifications, err := s.verificationRepository.FindByDatabaseIDWithPagination(databaseID, limit, offset)
	if err != nil {
		return nil, err
	}

	total, err := s.verificationRepository.CountByDatabaseID(databaseID)
	if err != nil {
		return nil, err
	}

	return &GetVerificationsResponse{
		Verifications: verifications,
		Total:         total,
		Limit:         limit,
		Offset:        offset,
	}, nil
}

func (s *VerificationService) GetVerificationByID(
	ctx context.Context,
	actor users.Actor,
	verificationID uuid.UUID,
) (*RestoreVerification, error) {
	verification, err := s.verificationRepository.FindByID(ctx, verificationID)
	if err != nil {
		return nil, err
	}

	if verification == nil {
		return nil, errors.New("verification not found")
	}

	database, err := s.databaseAccess.GetDatabaseByID(ctx, verification.DatabaseID)
	if err != nil {
		return nil, err
	}

	isAllowed, err := s.workspaceService.IsActorAllowed(ctx, actor, database.WorkspaceID, workspaces.ActionView)
	if err != nil {
		return nil, err
	}
	if !isAllowed {
		return nil, errors.New("insufficient permissions to view this verification")
	}

	return verification, nil
}

func (s *VerificationService) ClaimVerification(
	ctx context.Context,
	agent *verification_agents.Agent,
	req *ClaimRequest,
) (*JobAssignment, error) {
	logger := s.logger.With("agent_id", agent.ID)

	runningBackupIDs, err := s.verificationRepository.ListRunningBackupIDsByAgentID(agent.ID)
	if err != nil {
		return nil, err
	}

	runningBackups, err := s.getBackups().GetLogicalBackups(ctx, runningBackupIDs)
	if err != nil {
		return nil, err
	}

	var assignment *JobAssignment

	err = storage.GetDb().Transaction(func(tx *gorm.DB) error {
		pendingVerifications, txErr := s.verificationRepository.LockOldestPendingTop100(tx)
		if txErr != nil {
			return txErr
		}

		candidates, txErr := s.pairWithBackups(ctx, pendingVerifications)
		if txErr != nil {
			return txErr
		}

		for _, candidate := range candidates {
			candidateLogger := logger.With("verification_id", candidate.Verification.ID)

			admission := EvaluateDiskAdmission(req.Capacity, runningBackups, candidate.Backup)
			if !admission.IsFit {
				candidateLogger.DebugContext(
					ctx,
					fmt.Sprintf(
						"skipping claim: needs %d MB, %d MB already used of %d MB budget",
						admission.CandidateRequiredMb, admission.RunningUsedMb, admission.TotalBudgetMb,
					),
				)

				continue
			}

			database, dbErr := s.databaseAccess.GetDatabaseByID(ctx, candidate.Verification.DatabaseID)
			if dbErr != nil {
				return dbErr
			}

			assignedDatabase, validateErr := readAssignedDatabase(ctx, s.getDatabases(), database)
			if validateErr != nil {
				if !errors.Is(validateErr, errDatabaseNotVerifiable) {
					return validateErr
				}

				candidateLogger.WarnContext(
					ctx,
					"skipping claim: database not verifiable",
					"database_id", candidate.Verification.DatabaseID,
					"error", validateErr,
				)

				continue
			}

			now := time.Now().UTC()
			if claimErr := s.verificationRepository.UpdateClaim(
				tx,
				candidate.Verification.ID,
				agent.ID,
				now,
			); claimErr != nil {
				return claimErr
			}

			assignment = &JobAssignment{
				VerificationID:     candidate.Verification.ID,
				BackupID:           candidate.Verification.BackupID,
				BackupSizeMb:       candidate.Backup.BackupSizeMb,
				MaxContainerDiskMb: float64(EstimateRequiredForRestoreDiskMb(candidate.Backup)),
				Database:           assignedDatabase,
				TimescaledbVersion: candidate.Backup.TimescaledbVersion,
			}

			return nil
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return assignment, nil
}

func (s *VerificationService) OnAgentHeartbeated(
	ctx context.Context,
	agent *verification_agents.Agent,
	currentVerificationIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	ownedRunning, err := s.verificationRepository.FindRunningByAgent(agent.ID)
	if err != nil {
		return nil, err
	}

	reportedVerifications := make(map[uuid.UUID]struct{}, len(currentVerificationIDs))
	for _, id := range currentVerificationIDs {
		reportedVerifications[id] = struct{}{}
	}

	ownedVerifications := make(map[uuid.UUID]struct{}, len(ownedRunning))
	for _, row := range ownedRunning {
		ownedVerifications[row.ID] = struct{}{}
	}

	// An ID the agent reports that the backend no longer owns as RUNNING was
	// hard-deleted (config cascade) or flipped terminal/CANCELED by the reaper:
	// tell the agent to abort it.
	verificationsToAbort := make([]uuid.UUID, 0)
	for _, id := range currentVerificationIDs {
		if _, stillOwned := ownedVerifications[id]; !stillOwned {
			verificationsToAbort = append(verificationsToAbort, id)
		}
	}

	s.reclaimDroppedRunningJobs(ctx, agent, ownedRunning, reportedVerifications)

	return verificationsToAbort, nil
}

func (s *VerificationService) GetBackupFile(
	ctx context.Context,
	agent *verification_agents.Agent,
	verificationID uuid.UUID,
) (io.ReadCloser, error) {
	verification, err := s.verificationRepository.FindByID(ctx, verificationID)
	if err != nil {
		return nil, err
	}

	if verification == nil {
		return nil, errors.New("verification not found")
	}

	if verification.Status != VerificationStatusRunning {
		return nil, errors.New("verification is not in RUNNING state")
	}

	if verification.AgentID == nil || *verification.AgentID != agent.ID {
		return nil, errors.New("verification is not owned by this agent")
	}

	return s.getBackups().OpenLogicalBackupFile(ctx, verification.BackupID)
}

func (s *VerificationService) SubmitReport(
	ctx context.Context,
	agent *verification_agents.Agent,
	verificationID uuid.UUID,
	req *ReportRequest,
) error {
	verification, err := s.verificationRepository.FindByID(ctx, verificationID)
	if err != nil {
		return err
	}

	if verification == nil {
		return errors.New("verification gone")
	}

	if req.Status == VerificationStatusFailed {
		return s.RequeueOrFail(ctx, verification, classifyAgentReport(req), req.FailMessage)
	}

	if restoredDbTooSmall, sizeMessage := s.isRestoredTooSmall(ctx, verification, req); restoredDbTooSmall {
		return s.RequeueOrFail(ctx, verification, FailureReasonRestoredTooSmall, &sizeMessage)
	}

	if err := s.verificationRepository.WriteSuccessReport(verificationID, agent.ID, req); err != nil {
		return err
	}

	s.syncBackupVerificationStatus(ctx, verification, VerificationStatusCompleted)
	s.notifyTerminal(ctx, verification, VerificationStatusCompleted, nil)

	return nil
}

func (s *VerificationService) RequeueOrFail(
	ctx context.Context,
	row *RestoreVerification,
	reason FailureReason,
	agentMessage *string,
) error {
	if shouldRequeueAgentSide(reason, row.AttemptCount) {
		return s.verificationRepository.Requeue(nil, row.ID)
	}

	failMessage := failureMessageFor(reason, agentMessage)

	if err := s.verificationRepository.MarkTerminal(nil, row.ID, VerificationStatusFailed, map[string]any{
		"fail_message": failMessage,
	}); err != nil {
		return err
	}

	// Re-read so the notification body reflects the row's freshly stamped finished_at.
	updated, err := s.verificationRepository.FindByID(ctx, row.ID)
	if err == nil && updated != nil {
		row = updated
	}

	s.syncBackupVerificationStatus(ctx, row, VerificationStatusFailed)
	s.notifyTerminal(ctx, row, VerificationStatusFailed, &failMessage)
	return nil
}

// reclaimDroppedRunningJobs requeues (or, once attempts are exhausted, fails)
// every RUNNING job this still-online agent owns that its heartbeat no longer
// lists — the agent dropped it (report given up after the retry budget, or the
// agent restarted mid-run). Rows younger than agentJobReclaimGrace are skipped:
// the claim stamps started_at before the owner has had a heartbeat cycle to
// report the job, so a just-claimed row is legitimately unreported for one
// cycle. Best-effort: it logs its own errors and never fails the heartbeat.
func (s *VerificationService) reclaimDroppedRunningJobs(
	ctx context.Context,
	agent *verification_agents.Agent,
	ownedRunning []*RestoreVerification,
	reported map[uuid.UUID]struct{},
) {
	reclaimEligibleBefore := time.Now().UTC().Add(-agentJobReclaimGrace)

	for _, row := range ownedRunning {
		if _, stillReported := reported[row.ID]; stillReported {
			continue
		}

		if row.StartedAt == nil || !row.StartedAt.Before(reclaimEligibleBefore) {
			continue
		}

		logger := s.logger.With("verification_id", row.ID, "agent_id", agent.ID)
		logger.InfoContext(ctx, "reclaiming dropped verification")

		if err := s.RequeueOrFail(ctx, row, FailureReasonAgentDroppedJob, nil); err != nil {
			logger.ErrorContext(ctx, "failed to reclaim dropped verification", "error", err)
		}
	}
}

func (s *VerificationService) scheduleVerificationForCompletedBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) error {
	logger := s.logger.With("database_id", databaseID)

	config, err := s.configService.GetByDatabaseIDNoAuth(databaseID)
	if err != nil {
		return fmt.Errorf("load verification config on backup completion: %w", err)
	}

	if config == nil ||
		!config.IsScheduledVerificationEnabled ||
		config.ScheduleType != verification_config.VerificationScheduleAfterBackup {
		return nil
	}

	verifiableBackup, isFound, err := s.getBackups().GetLatestCompletedLogicalBackup(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("load latest completed backup: %w", err)
	}

	if !isFound {
		logger.InfoContext(ctx, "skipping after-backup verification: no verifiable backup yet")
		return nil
	}

	database, err := s.databaseAccess.GetDatabaseByID(ctx, databaseID)
	if err != nil {
		return fmt.Errorf("load database for after-backup verification: %w", err)
	}

	if _, err := readAssignedDatabase(ctx, s.getDatabases(), database); err != nil {
		if !errors.Is(err, errDatabaseNotVerifiable) {
			return err
		}

		logger.WarnContext(ctx, "skipping after-backup verification: database not verifiable", "error", err)

		return nil
	}

	err = storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if cancelErr := s.cancelPendingAutoScheduledVerificationsByDatabaseID(tx, databaseID); cancelErr != nil {
			return cancelErr
		}

		return tx.Create(&RestoreVerification{
			ID:           uuid.New(),
			DatabaseID:   databaseID,
			BackupID:     verifiableBackup.ID,
			Trigger:      VerificationTriggerAfterBackup,
			Status:       VerificationStatusPending,
			AttemptCount: 1,
			CreatedAt:    time.Now().UTC(),
		}).Error
	})
	if err != nil {
		return fmt.Errorf("enqueue after-backup verification: %w", err)
	}

	return nil
}

func (s *VerificationService) cancelPendingAutoScheduledVerificationsByDatabaseID(
	tx *gorm.DB,
	databaseID uuid.UUID,
) error {
	existing, err := s.verificationRepository.FindNonTerminalForDatabase(tx, databaseID)
	if err != nil {
		return err
	}

	for _, row := range existing {
		if row.Trigger == VerificationTriggerManual {
			continue
		}

		if row.Status == VerificationStatusPending {
			if cancelErr := s.verificationRepository.MarkTerminal(
				tx, row.ID, VerificationStatusCanceled, map[string]any{
					"fail_message": cancelMessageSupersededByBackup,
				},
			); cancelErr != nil {
				return cancelErr
			}
		}
	}

	return nil
}

func isAgentSideFailure(reason FailureReason) bool {
	return reason == FailureReasonAgentLostContact ||
		reason == FailureReasonAgentRemoved ||
		reason == FailureReasonAgentSetupFailed ||
		reason == FailureReasonAgentDroppedJob
}

// shouldRequeueAgentSide is the single retry-vs-terminal policy: an agent-side
// failure with attempts left is requeued; everything else goes terminal
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

func (s *VerificationService) isRestoredTooSmall(
	ctx context.Context,
	verification *RestoreVerification,
	req *ReportRequest,
) (bool, string) {
	if req.DBSizeBytesAfterRestore == nil {
		return false, ""
	}

	backup, isFound, err := s.getLogicalBackup(ctx, verification.BackupID)
	if err != nil || !isFound || backup.BackupRawDbSizeMb <= 0 {
		return false, ""
	}

	rawDbSizeBytes := int64(backup.BackupRawDbSizeMb * 1024 * 1024)
	minAcceptableBytes := int64(float64(rawDbSizeBytes) * minRestoredSizeRatio)

	if *req.DBSizeBytesAfterRestore >= minAcceptableBytes {
		return false, ""
	}

	message := fmt.Sprintf(
		"restored database is %d bytes, which is less than %.0f%% of the original size (%d bytes)",
		*req.DBSizeBytesAfterRestore, minRestoredSizeRatio*100, rawDbSizeBytes,
	)

	return true, message
}

func (s *VerificationService) syncBackupVerificationStatus(
	ctx context.Context,
	verification *RestoreVerification,
	terminalStatus VerificationStatus,
) {
	var status backups.LogicalBackupVerificationStatus

	switch terminalStatus {
	case VerificationStatusCompleted:
		status = backups.LogicalBackupVerificationStatusVerifiedSuccessful
	case VerificationStatusFailed:
		status = backups.LogicalBackupVerificationStatusVerificationFailed
	default:
		return
	}

	if err := s.getBackups().SetLogicalBackupVerificationStatus(ctx, verification.BackupID, status); err != nil {
		s.logger.ErrorContext(
			ctx,
			"failed to sync backup restore-verification status",
			"error", err,
			"verification_id", verification.ID,
			"backup_id", verification.BackupID,
		)
	}
}

func (s *VerificationService) notifyTerminal(
	ctx context.Context,
	verification *RestoreVerification,
	terminalStatus VerificationStatus,
	failMessage *string,
) {
	if terminalStatus == VerificationStatusCanceled {
		return
	}

	config, err := s.configService.GetByDatabaseIDNoAuth(verification.DatabaseID)
	if err != nil {
		s.logger.ErrorContext(ctx,
			"failed to load verification config for notification",
			"error", err,
			"verification_id", verification.ID,
		)

		return
	}

	if config == nil {
		return
	}

	wantedType := verification_config.NotificationVerificationSuccess
	if terminalStatus == VerificationStatusFailed {
		wantedType = verification_config.NotificationVerificationFailed
	}

	if !slices.Contains(config.SendNotificationsOn, wantedType) {
		return
	}

	database, err := s.databaseAccess.GetDatabaseByID(ctx, verification.DatabaseID)
	if err != nil {
		s.logger.ErrorContext(ctx,
			"failed to load database for notification",
			"error", err,
			"verification_id", verification.ID,
		)

		return
	}

	title, body := buildNotificationCopy(verification, database, terminalStatus, failMessage)

	sentNotificationType := notifiers.CategoryVerificationSuccess
	if wantedType == verification_config.NotificationVerificationFailed {
		sentNotificationType = notifiers.CategoryVerificationFailed
	}

	s.notifierService.Notify(ctx, database.NotifierIDs, notifiers.Notification{
		Category: sentNotificationType,
		Heading:  title,
		Message:  body,
	})
}

func (s *VerificationService) guardBackupIsVerifiable(backup backups.LogicalBackup) error {
	if backup.Status != backups.LogicalBackupStatusCompleted {
		return errors.New("only COMPLETED backups can be verified")
	}

	return nil
}

func (s *VerificationService) getLogicalBackup(
	ctx context.Context,
	backupID uuid.UUID,
) (backups.LogicalBackup, bool, error) {
	foundBackups, err := s.getBackups().GetLogicalBackups(ctx, []uuid.UUID{backupID})
	if err != nil || len(foundBackups) == 0 {
		return backups.LogicalBackup{}, false, err
	}

	return foundBackups[0], true, nil
}

func (s *VerificationService) pairWithBackups(
	ctx context.Context,
	verifications []*RestoreVerification,
) ([]ClaimCandidate, error) {
	backupIDs := make([]uuid.UUID, len(verifications))
	for i, verification := range verifications {
		backupIDs[i] = verification.BackupID
	}

	foundBackups, err := s.getBackups().GetLogicalBackups(ctx, backupIDs)
	if err != nil {
		return nil, err
	}

	backupByID := make(map[uuid.UUID]backups.LogicalBackup, len(foundBackups))
	for _, backup := range foundBackups {
		backupByID[backup.ID] = backup
	}

	candidates := make([]ClaimCandidate, 0, len(verifications))
	for _, verification := range verifications {
		backup, isFound := backupByID[verification.BackupID]
		if !isFound {
			continue
		}

		candidates = append(candidates, ClaimCandidate{Verification: verification, Backup: backup})
	}

	return candidates, nil
}

const (
	cancelMessageByUser             = "verification canceled by user"
	cancelMessageScheduleDisabled   = "verification canceled: its verification schedule was disabled"
	cancelMessageSupersededByBackup = "verification canceled: superseded by a newer backup's scheduled verification"
)

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

func buildNotificationCopy(
	verification *RestoreVerification,
	database databases.Database,
	terminalStatus VerificationStatus,
	failMessage *string,
) (title, body string) {
	workspaceLabel := fmt.Sprintf(" (workspace %q)", database.WorkspaceID.String())

	if terminalStatus == VerificationStatusCompleted {
		title = fmt.Sprintf("backup verification passed for %q%s", database.Name, workspaceLabel)
	} else {
		title = fmt.Sprintf(
			"backup verification failed after %d attempts for %q%s",
			verification.AttemptCount, database.Name, workspaceLabel,
		)
	}

	bodyParts := []string{
		fmt.Sprintf("verification id: %s", verification.ID),
		fmt.Sprintf("backup id: %s", verification.BackupID),
		fmt.Sprintf("attempt: %d", verification.AttemptCount),
	}

	if verification.RestoreDurationMs != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("restore duration: %d ms", *verification.RestoreDurationMs))
	}
	if verification.VerifyDurationMs != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("verify duration: %d ms", *verification.VerifyDurationMs))
	}
	if verification.DBSizeBytesAfterRestore != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("db size: %d bytes", *verification.DBSizeBytesAfterRestore))
	}
	if verification.TableCount != nil {
		bodyParts = append(bodyParts, fmt.Sprintf("table count: %d", *verification.TableCount))
	}

	if failMessage != nil && *failMessage != "" {
		messageRunes := []rune(*failMessage)
		if len(messageRunes) > 2000 {
			messageRunes = messageRunes[:2000]
		}

		bodyParts = append(bodyParts, fmt.Sprintf("error: %s", string(messageRunes)))
	}

	body = ""
	for i, part := range bodyParts {
		if i > 0 {
			body += "\n"
		}

		body += part
	}

	return title, body
}
