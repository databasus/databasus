package backups_services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	"databasus-backend/internal/features/backups/backups/download/restore_stream"
	"databasus-backend/internal/features/backups/backups/download/restore_token"
	backups_dto_physical "databasus-backend/internal/features/backups/backups/dto/physical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

// ErrNoExtendableChain is returned when an incremental backup is requested but
// no chain can take one — the caller must run a full first.
var ErrNoExtendableChain = errors.New("no extendable chain; run a full backup first")

// ErrInvalidBackupsCursor is returned when a backup list request carries a
// cursor the server did not issue or cannot decode.
var ErrInvalidBackupsCursor = errors.New("invalid backups cursor")

// ErrBackupNotFound is returned when a backup id matches no FULL, incremental,
// or WAL row owned by a database the user can access.
var ErrBackupNotFound = errors.New("backup not found")

// ErrBackupNotInProgress is returned when a cancel is requested for a backup
// that is not currently running.
var ErrBackupNotInProgress = errors.New("backup is not in progress")

// PhysicalBackupService is the workspace-authorized front door to the physical
// backup catalog: listing every backup (FULL / incremental / WAL) as a flat
// list, triggering and cancelling backups, deleting a backup with its dependent
// cascade, and producing the restore stream. It composes the catalog
// (chain_view), the request-flag config service, the cascade-delete core
// service, the in-flight canceller, and the shared restore stream writer.
// OpenRestoreStream is the auth-free seam reused by restore verification.
type PhysicalBackupService struct {
	databaseService             *databases.DatabaseService
	workspaceService            *workspaces_services.WorkspaceService
	backupConfigService         *backups_config_physical.BackupConfigService
	chainViewService            *chain_view.ChainViewService
	physicalCoreService         *physical_core_service.PhysicalBackupService
	fullBackupRepository        *physical_repositories.PhysicalFullBackupRepository
	incrementalBackupRepository *physical_repositories.PhysicalIncrementalBackupRepository
	walSegmentRepository        *physical_repositories.PhysicalWalSegmentRepository
	inFlightBackupRepository    *physical_repositories.PhysicalInFlightBackupRepository
	canceller                   *backuping_physical.PhysicalBackupCanceller
	walStreamSupervisor         *backuping_physical.PhysicalWalStreamSupervisor
	restoreStreamWriter         *restore_stream.Writer
	restoreTokenService         *restore_token.Service
	secretKeyService            *encryption_secrets.SecretKeyService
	auditLogService             *audit_logs.AuditLogService
	logger                      *slog.Logger
}

// GetBackups returns one page of a database's backups — FULLs, incrementals and
// WAL ranges of consecutive committed segments — as one flat list sorted
// newest-first. It pages by keyset cursor rather than offset because the newest
// WAL range grows and new backups appear at the top between requests, which
// would shift offsets. One extra row is fetched to know whether a next page
// exists. The aggregate on-disk usage rides along so the UI need not sum it.
func (s *PhysicalBackupService) GetBackups(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
	request *backups_dto_physical.GetPhysicalBackupsRequest,
) (*backups_dto_physical.GetPhysicalBackupsResponse, error) {
	if _, err := s.databaseService.GetDatabase(ctx, user, databaseID); err != nil {
		return nil, err
	}

	limit := normalizeBackupsPageLimit(request.Limit)
	filter := toBackupListFilter(request)

	after, err := decodePhysicalListCursor(request.Cursor)
	if err != nil {
		return nil, err
	}

	rows, err := s.physicalCoreService.ListBackups(databaseID, filter, after, limit+1)
	if err != nil {
		return nil, err
	}

	totalUsageMb, err := s.physicalCoreService.GetTotalUsageMBByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	hasNextPage := len(rows) > limit
	if hasNextPage {
		rows = rows[:limit]
	}

	items := make([]backups_dto_physical.PhysicalBackupListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, rowToListItem(row))
	}

	response := &backups_dto_physical.GetPhysicalBackupsResponse{
		Backups:      items,
		TotalUsageMb: totalUsageMb,
	}

	if hasNextPage {
		lastRow := rows[len(rows)-1]

		response.NextCursor, err = encodePhysicalListCursor(physical_core_service.BackupListPosition{
			SortTime: lastRow.SortTime,
			ID:       lastRow.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	return response, nil
}

// RequestBackup triggers the backup the scheduler would naturally pick next:
// an incremental when incrementals are enabled and an extendable chain exists,
// otherwise a full.
func (s *PhysicalBackupService) RequestBackup(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.authorizeManage(ctx, user, databaseID)
	if err != nil {
		return err
	}

	config, err := s.backupConfigService.GetBackupConfigByDbId(databaseID)
	if err != nil {
		return err
	}

	if err := s.requestSchedulerChoice(databaseID, config); err != nil {
		return err
	}

	s.writeBackupAuditLog(ctx, user, database,
		fmt.Sprintf("Physical backup requested for database: %s", database.Name))

	return nil
}

func (s *PhysicalBackupService) RequestFullBackup(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.authorizeManage(ctx, user, databaseID)
	if err != nil {
		return err
	}

	if err := s.backupConfigService.RequestFullBackupNow(databaseID); err != nil {
		return err
	}

	s.writeBackupAuditLog(ctx, user, database,
		fmt.Sprintf("Physical full backup requested for database: %s", database.Name))

	return nil
}

func (s *PhysicalBackupService) RequestIncrementalBackup(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.authorizeManage(ctx, user, databaseID)
	if err != nil {
		return err
	}

	extendable, err := s.chainViewService.FindLastExtendableChainByDatabase(databaseID)
	if err != nil {
		return err
	}

	if extendable == nil {
		return ErrNoExtendableChain
	}

	if err := s.backupConfigService.RequestIncrementalBackupNow(databaseID); err != nil {
		return err
	}

	s.writeBackupAuditLog(ctx, user, database,
		fmt.Sprintf("Physical incremental backup requested for database: %s", database.Name))

	return nil
}

// CancelBackup stops an in-progress FULL or incremental backup: it authorizes
// the caller against the owning database, refuses anything not currently
// running, then cancels the in-flight task and releases its claim.
func (s *PhysicalBackupService) CancelBackup(ctx context.Context, user *users_models.User, backupID uuid.UUID) error {
	databaseID, status, found, err := s.resolveBackupDatabaseAndStatus(backupID)
	if err != nil {
		return err
	}
	if !found {
		return ErrBackupNotFound
	}

	database, err := s.authorizeManage(ctx, user, databaseID)
	if err != nil {
		return err
	}

	if status != physical_enums.PhysicalBackupStatusInProgress {
		return ErrBackupNotInProgress
	}

	cancelled, err := s.canceller.CancelInFlightBackup(ctx, backuping_physical.CancelInFlightBackupSpec{
		DatabaseID: databaseID,
		BackupID:   backupID,
	})
	if err != nil {
		return err
	}

	// The row said IN_PROGRESS but no claim owns it: the backup just finished or
	// was already cancelled. Report it as not-in-progress rather than a success.
	if !cancelled {
		return ErrBackupNotInProgress
	}

	s.writeBackupAuditLog(ctx, user, database,
		fmt.Sprintf("Physical backup cancelled for database: %s", database.Name))

	return nil
}

// DeleteBackup removes a backup and its dependent cascade, authorizing the
// caller first. A running backup in the removed set is stopped before its row
// goes. The cascade differs by type:
//   - FULL: the whole chain — incrementals and WAL included.
//   - incremental: that incremental and every incremental descending from it;
//     WAL and the FULL are left intact.
//   - WAL: that segment and every later segment up to the next FULL/INCR anchor.
func (s *PhysicalBackupService) DeleteBackup(ctx context.Context, user *users_models.User, backupID uuid.UUID) error {
	full, err := s.fullBackupRepository.FindByID(backupID)
	if err != nil {
		return err
	}
	if full != nil {
		database, err := s.authorizeManage(ctx, user, full.DatabaseID)
		if err != nil {
			return err
		}

		if err := s.deleteFull(ctx, full); err != nil {
			return err
		}

		s.writeBackupAuditLog(ctx, user, database,
			fmt.Sprintf("Physical full backup deleted for database: %s", database.Name))

		return nil
	}

	incremental, err := s.incrementalBackupRepository.FindByID(backupID)
	if err != nil {
		return err
	}
	if incremental != nil {
		database, err := s.authorizeManage(ctx, user, incremental.DatabaseID)
		if err != nil {
			return err
		}

		if err := s.deleteIncremental(ctx, incremental); err != nil {
			return err
		}

		s.writeBackupAuditLog(ctx, user, database,
			fmt.Sprintf("Physical incremental backup deleted for database: %s", database.Name))

		return nil
	}

	walSegment, err := s.walSegmentRepository.FindByID(backupID)
	if err != nil {
		return err
	}
	if walSegment != nil {
		database, err := s.authorizeManage(ctx, user, walSegment.DatabaseID)
		if err != nil {
			return err
		}

		if _, _, err := s.physicalCoreService.DeleteWalSegment(ctx, walSegment.ID); err != nil {
			return err
		}

		s.writeBackupAuditLog(ctx, user, database,
			fmt.Sprintf("Physical WAL segment deleted for database: %s", database.Name))

		return nil
	}

	return ErrBackupNotFound
}

// GenerateRestoreToken authorizes the caller against the database and issues a
// single-use restore-stream token for the given PITR target (nil ⇒ latest). It
// resolves the restore set up front (without streaming) so an unreachable target
// — a WAL gap, no chain, or a target before the earliest full — is reported here
// rather than burning a token that only fails once the user runs the curl.
func (s *PhysicalBackupService) GenerateRestoreToken(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
	request *backups_dto_physical.GenerateRestoreTokenRequest,
) (string, error) {
	if _, err := s.databaseService.GetDatabase(ctx, user, databaseID); err != nil {
		return "", err
	}

	if _, err := s.chainViewService.ResolveRestoreSet(databaseID, request.TargetTime); err != nil {
		return "", err
	}

	return s.restoreTokenService.GenerateRestoreToken(ctx, databaseID, user.ID, request.TargetTime)
}

// GenerateBackupRestoreToken authorizes the caller and issues a single-use token
// for a per-backup restore (the FULL plus its incremental ancestors, no WAL) of
// a specific FULL or incremental. It resolves the restore set up front so a
// missing or not-yet-restorable backup is reported here, not mid-stream.
func (s *PhysicalBackupService) GenerateBackupRestoreToken(
	ctx context.Context,
	user *users_models.User,
	backupID uuid.UUID,
) (string, error) {
	databaseID, _, found, err := s.resolveBackupDatabaseAndStatus(backupID)
	if err != nil {
		return "", err
	}
	if !found {
		return "", ErrBackupNotFound
	}

	if _, err := s.databaseService.GetDatabase(ctx, user, databaseID); err != nil {
		return "", err
	}

	if _, err := s.chainViewService.ResolveRestoreSetForBackup(databaseID, backupID); err != nil {
		return "", err
	}

	return s.restoreTokenService.GenerateBackupRestoreToken(ctx, databaseID, user.ID, backupID)
}

// Performs NO authorization — callers gate access (the user path via a restore token, restore
// verification via agent ownership). This is the shared seam between the two.
func (s *PhysicalBackupService) OpenRestoreStream(
	ctx context.Context,
	databaseID uuid.UUID,
	targetTime *time.Time,
	w io.Writer,
) error {
	set, err := s.chainViewService.ResolveRestoreSet(databaseID, targetTime)
	if err != nil {
		return err
	}

	return s.writeRestoreStream(ctx, set, w)
}

// The per-backup restore set is the FULL plus its incremental ancestors, no WAL. Like
// OpenRestoreStream it performs no authorization — the caller's token already did.
func (s *PhysicalBackupService) OpenRestoreStreamForBackup(
	ctx context.Context,
	databaseID, backupID uuid.UUID,
	w io.Writer,
) error {
	set, err := s.chainViewService.ResolveRestoreSetForBackup(databaseID, backupID)
	if err != nil {
		return err
	}

	return s.writeRestoreStream(ctx, set, w)
}

// OnBackupsStorageChanged removes every full backup of the database kept outside storageID, each
// with the incrementals and WAL of its chain, through the same deletion a user triggers, so an
// in-flight backup of that chain stops and WAL another full backup needs is kept.
func (s *PhysicalBackupService) OnBackupsStorageChanged(
	ctx context.Context,
	databaseID uuid.UUID,
	storageID uuid.UUID,
) error {
	fullsInOtherStorages, err := s.fullBackupRepository.FindAllByDatabaseOutsideStorage(databaseID, storageID)
	if err != nil {
		return err
	}

	if len(fullsInOtherStorages) == 0 {
		return nil
	}

	logger := s.logger.With("database_id", databaseID, "storage_id", storageID)

	var removalErrors []error

	for _, full := range fullsInOtherStorages {
		if err := s.deleteFull(ctx, full); err != nil {
			logger.ErrorContext(ctx, "failed to remove full backup kept in a previous storage",
				"backup_id", full.ID, "error", err)

			removalErrors = append(removalErrors, fmt.Errorf("remove full backup %s: %w", full.ID, err))
		}
	}

	logger.InfoContext(ctx, "removed full backups kept in previous storages",
		"removed_count", len(fullsInOtherStorages)-len(removalErrors),
		"found_count", len(fullsInOtherStorages))

	return errors.Join(removalErrors...)
}

func (s *PhysicalBackupService) writeRestoreStream(
	ctx context.Context,
	set *chain_view.RestoreSet,
	w io.Writer,
) error {
	masterKey, err := s.resolveMasterKey(set)
	if err != nil {
		return err
	}

	return s.restoreStreamWriter.Write(ctx, w, set, masterKey)
}

func (s *PhysicalBackupService) deleteFull(ctx context.Context, full *physical_models.PhysicalFullBackup) error {
	s.stopInFlightForDelete(ctx, full.DatabaseID, func(inFlightBackupID uuid.UUID) bool {
		if inFlightBackupID == full.ID {
			return true
		}

		incremental, err := s.incrementalBackupRepository.FindByID(inFlightBackupID)

		return err == nil && incremental != nil && incremental.RootFullBackupID == full.ID
	})

	// An in-progress FULL has no start_lsn yet, so there is no chain span to
	// cascade — drop the bare row (its claim, if any, was cancelled above).
	if full.StartLSN == nil {
		return s.fullBackupRepository.DeleteByID(full.ID)
	}

	_, err := s.physicalCoreService.DeleteFullCompletely(ctx, full.ID)

	return err
}

func (s *PhysicalBackupService) deleteIncremental(
	ctx context.Context,
	target *physical_models.PhysicalIncrementalBackup,
) error {
	s.stopInFlightForDelete(ctx, target.DatabaseID, func(inFlightBackupID uuid.UUID) bool {
		if inFlightBackupID == target.ID {
			return true
		}

		// A descendant of the target shares its root and starts at or after it.
		claimed, err := s.incrementalBackupRepository.FindByID(inFlightBackupID)
		if err != nil || claimed == nil {
			return false
		}

		return claimed.RootFullBackupID == target.RootFullBackupID &&
			claimed.StartLSN != nil && target.StartLSN != nil &&
			*claimed.StartLSN >= *target.StartLSN
	})

	_, err := s.physicalCoreService.DeleteIncrementalCascade(ctx, target.ID)

	return err
}

// stopInFlightForDelete cancels the database's in-flight backup when willDelete
// reports its backup is one this delete will remove, so the running executor
// stops before its row disappears. Best-effort: lookup failures are logged, not
// fatal — a stale claim expires on its own and the delete proceeds.
func (s *PhysicalBackupService) stopInFlightForDelete(
	ctx context.Context,
	databaseID uuid.UUID,
	willDelete func(inFlightBackupID uuid.UUID) bool,
) {
	claim, err := s.inFlightBackupRepository.FindByDatabaseID(databaseID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to look up in-flight backup for delete",
			"database_id", databaseID, "error", err)

		return
	}

	if claim == nil {
		return
	}

	if willDelete(claim.BackupID) {
		s.canceller.CancelInFlightForDatabase(ctx, databaseID)
	}
}

// requestSchedulerChoice requests the backup the scheduler would naturally pick:
// an incremental when incrementals are enabled and an extendable chain exists,
// otherwise a full.
func (s *PhysicalBackupService) requestSchedulerChoice(
	databaseID uuid.UUID,
	config *backups_config_physical.PhysicalBackupConfig,
) error {
	if config.PostgresqlPhysical == nil || !config.PostgresqlPhysical.BackupType.IsRequireWalSummary() {
		return s.backupConfigService.RequestFullBackupNow(databaseID)
	}

	extendable, err := s.chainViewService.FindLastExtendableChainByDatabase(databaseID)
	if err != nil {
		return err
	}

	if extendable == nil {
		return s.backupConfigService.RequestFullBackupNow(databaseID)
	}

	return s.backupConfigService.RequestIncrementalBackupNow(databaseID)
}

// authorizeManage loads the database and requires the caller to hold DB-management
// rights in its workspace (owner/admin/member; a viewer is rejected). It returns
// the database so callers can name it in audit logs. This is the gate for every
// state-changing physical-backup operation — trigger, cancel, delete — matching
// the logical backup service.
func (s *PhysicalBackupService) authorizeManage(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) (*databases.Database, error) {
	database, err := s.databaseService.GetDatabase(ctx, user, databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot manage backups for a database without a workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}

	if !canManage {
		return nil, errors.New("insufficient permissions to manage backups for this database")
	}

	return database, nil
}

// writeBackupAuditLog records a workspace audit entry for a state-changing
// physical-backup operation, attributing it to the acting user.
func (s *PhysicalBackupService) writeBackupAuditLog(
	ctx context.Context,
	user *users_models.User,
	database *databases.Database,
	message string,
) {
	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     message,
		UserID:      &user.ID,
		WorkspaceID: database.WorkspaceID,
	})
}

// resolveBackupDatabaseAndStatus probes the FULL then incremental tables for a
// backup id, returning its owning database and status. WAL segments have no
// trigger/cancel identity and are intentionally not probed here.
func (s *PhysicalBackupService) resolveBackupDatabaseAndStatus(
	backupID uuid.UUID,
) (uuid.UUID, physical_enums.PhysicalBackupStatus, bool, error) {
	full, err := s.fullBackupRepository.FindByID(backupID)
	if err != nil {
		return uuid.Nil, "", false, err
	}
	if full != nil {
		return full.DatabaseID, full.Status, true, nil
	}

	incremental, err := s.incrementalBackupRepository.FindByID(backupID)
	if err != nil {
		return uuid.Nil, "", false, err
	}
	if incremental != nil {
		return incremental.DatabaseID, incremental.Status, true, nil
	}

	return uuid.Nil, "", false, nil
}

// resolveMasterKey fetches the master key only when the set actually contains an
// encrypted artifact, so non-encrypted restores don't depend on a configured key.
func (s *PhysicalBackupService) resolveMasterKey(set *chain_view.RestoreSet) (string, error) {
	if !restoreSetHasEncryption(set) {
		return "", nil
	}

	return s.secretKeyService.GetSecretKey()
}

func rowToListItem(row physical_core_service.PhysicalBackupListRow) backups_dto_physical.PhysicalBackupListItem {
	return backups_dto_physical.PhysicalBackupListItem{
		ID:                        row.ID,
		Type:                      physical_enums.PhysicalBackupType(row.Type),
		Status:                    row.Status,
		TimelineID:                row.TimelineID,
		StartLSN:                  row.StartLSN,
		StopLSN:                   row.StopLSN,
		RootFullBackupID:          row.RootFullBackupID,
		ParentIncrementalBackupID: row.ParentIncrementalBackupID,
		FromWalFilename:           row.FromWalFilename,
		ToWalFilename:             row.ToWalFilename,
		WalSegmentsCount:          row.WalSegmentsCount,
		HasGapBefore:              row.HasGapBefore,
		SizeMb:                    row.SizeMb,
		FailMessage:               row.FailMessage,
		CreatedAt:                 row.CreatedAt,
		CompletedAt:               row.CompletedAt,
	}
}

// toBackupListFilter maps the API request's optional filters onto the core
// filter, casting the typed Type enum values to the string column the UNION
// projects.
func toBackupListFilter(
	request *backups_dto_physical.GetPhysicalBackupsRequest,
) physical_core_service.BackupListFilter {
	filter := physical_core_service.BackupListFilter{
		Statuses:   request.Statuses,
		BeforeDate: request.BeforeDate,
	}

	for _, backupType := range request.Types {
		filter.Types = append(filter.Types, string(backupType))
	}

	return filter
}

// normalizeBackupsPageLimit applies the default page size for an unset/invalid
// limit and caps it so a single request can't ask for an unbounded page.
func normalizeBackupsPageLimit(limit int) int {
	const (
		defaultLimit = 50
		maxLimit     = 1000
	)

	if limit <= 0 {
		return defaultLimit
	}

	if limit > maxLimit {
		return maxLimit
	}

	return limit
}

func restoreSetHasEncryption(set *chain_view.RestoreSet) bool {
	if set.RootFull.Encryption == backups_core_enums.BackupEncryptionEncrypted {
		return true
	}

	for _, incremental := range set.Incrementals {
		if incremental.Encryption == backups_core_enums.BackupEncryptionEncrypted {
			return true
		}
	}

	for _, segment := range set.WalSegments {
		if segment.Encryption == backups_core_enums.BackupEncryptionEncrypted {
			return true
		}
	}

	// History files carry their encryption flag only in their sidecar, so a
	// non-encrypted catalog with encrypted history is theoretically possible; the
	// stream writer reads the sidecar and a master key is cheap to fetch when any
	// history exists.
	return len(set.HistoryFiles) > 0
}
