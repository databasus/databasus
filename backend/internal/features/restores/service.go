package restores

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/features/restores/restoring"
	"databasus-backend/internal/features/restores/usecases"
	"databasus-backend/internal/features/storages"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/tools"
)

type RestoreService struct {
	backupService             *backups_services.LogicalBackupService
	restoreRepository         *restores_core.RestoreRepository
	storageService            *storages.StorageService
	backupConfigService       *backups_config_logical.BackupConfigService
	restoreBackupUsecase      *usecases.RestoreBackupUsecase
	databaseService           *databases.DatabaseService
	logger                    *slog.Logger
	workspaceService          *workspaces_services.WorkspaceService
	auditLogService           *audit_logs.AuditLogService
	fieldEncryptor            encryption.FieldEncryptor
	diskService               *disk.DiskService
	taskCancellationRequester *tasks_cancellation.Requester
}

func (s *RestoreService) OnBeforeBackupRemove(backup *backups_core_logical.LogicalBackup) error {
	restores, err := s.restoreRepository.FindByBackupID(backup.ID)
	if err != nil {
		return err
	}

	for _, restore := range restores {
		if restore.Status == restores_core.RestoreStatusInProgress {
			return errors.New("restore is in progress, backup cannot be removed")
		}
	}

	for _, restore := range restores {
		if err := s.restoreRepository.DeleteByID(restore.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *RestoreService) GetRestores(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
) ([]*restores_core.Restore, error) {
	backup, err := s.backupService.GetBackup(backupID)
	if err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot get restores for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
		ctx,
		*database.WorkspaceID,
		actor,
	)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access restores for this backup")
	}

	return s.restoreRepository.FindByBackupID(backupID)
}

func (s *RestoreService) RestoreBackupWithAuth(
	ctx context.Context,
	actor users.Actor,
	backupID uuid.UUID,
	requestDTO restores_core.RestoreBackupRequest,
) error {
	backup, err := s.backupService.GetBackup(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot restore backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(
		ctx,
		*database.WorkspaceID,
		actor,
	)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to restore this backup")
	}

	backupDatabase, err := s.databaseService.GetDatabase(ctx, actor, backup.DatabaseID)
	if err != nil {
		return err
	}

	if err := s.populateRestoreTargetVersionThroughTunnel(ctx, backupDatabase, requestDTO); err != nil {
		return err
	}

	if err := s.validateVersionCompatibility(backupDatabase, requestDTO); err != nil {
		return err
	}

	// Validate disk space before starting restore
	if err := s.validateDiskSpace(ctx, backup, requestDTO); err != nil {
		return err
	}

	// Validate no parallel restores for the same database
	if err := s.validateNoParallelRestores(backup.DatabaseID); err != nil {
		return err
	}

	// Create restore record with the request configuration
	restore := restores_core.Restore{
		ID:                        uuid.New(),
		Status:                    restores_core.RestoreStatusInProgress,
		BackupID:                  backup.ID,
		Backup:                    backup,
		CreatedAt:                 time.Now().UTC(),
		RestoreDurationMs:         0,
		FailMessage:               nil,
		PostgresqlLogicalDatabase: requestDTO.PostgresqlLogicalDatabase,
		MysqlDatabase:             requestDTO.MysqlDatabase,
		MariadbDatabase:           requestDTO.MariadbDatabase,
		MongodbDatabase:           requestDTO.MongodbDatabase,
	}

	if err := s.restoreRepository.Save(&restore); err != nil {
		return err
	}

	// Prepare database cache with credentials from the request
	dbCache := &restoring.RestoreDatabaseCache{
		PostgresqlLogicalDatabase: requestDTO.PostgresqlLogicalDatabase,
		MysqlDatabase:             requestDTO.MysqlDatabase,
		MariadbDatabase:           requestDTO.MariadbDatabase,
		MongodbDatabase:           requestDTO.MongodbDatabase,
	}

	// Trigger restore via scheduler
	scheduler := restoring.GetRestoresScheduler()
	if err := scheduler.StartRestore(ctx, restore.ID, dbCache); err != nil {
		// Mark restore as failed if we can't schedule it
		failMsg := fmt.Sprintf("Failed to schedule restore: %v", err)
		restore.FailMessage = &failMsg
		restore.Status = restores_core.RestoreStatusFailed
		if saveErr := s.restoreRepository.Save(&restore); saveErr != nil {
			s.logger.ErrorContext(ctx, "failed to save restore after scheduling error", "error", saveErr)
		}
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Database restored for database: %s", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: database.WorkspaceID,
	})

	return nil
}

func (s *RestoreService) CancelRestore(
	ctx context.Context,
	actor users.Actor,
	restoreID uuid.UUID,
) error {
	restore, err := s.restoreRepository.FindByID(restoreID)
	if err != nil {
		return err
	}

	backup, err := s.backupService.GetBackup(restore.BackupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot cancel restore for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, actor)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to cancel restore for this database")
	}

	if restore.Status != restores_core.RestoreStatusInProgress {
		return errors.New("restore is not in progress")
	}

	if err := s.taskCancellationRequester.RequestCancellation(ctx, restoreID); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Restore cancelled for database: %s", database.Name),
		UserID:      &actor.UserID,
		WorkspaceID: database.WorkspaceID,
	})

	return nil
}

func (s *RestoreService) validateVersionCompatibility(
	backupDatabase *databases.Database,
	requestDTO restores_core.RestoreBackupRequest,
) error {
	switch backupDatabase.Type {
	case databases.DatabaseTypePostgresLogical:
		if requestDTO.PostgresqlLogicalDatabase == nil {
			return errors.New("postgresql database configuration is required for restore")
		}
		if tools.IsBackupDbVersionHigherThanRestoreDbVersion(
			backupDatabase.PostgresqlLogical.Version,
			requestDTO.PostgresqlLogicalDatabase.Version,
		) {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore PG 15 backup to PG 15, 16 or higher. But cannot restore to 14 and lower`)
		}
	case databases.DatabaseTypeMysql:
		if requestDTO.MysqlDatabase == nil {
			return errors.New("mysql database configuration is required for restore")
		}
		isMysqlDowngrade, err := tools.IsMysqlBackupVersionHigherThanRestoreVersion(
			backupDatabase.Mysql.Version,
			requestDTO.MysqlDatabase.Version,
		)
		if err != nil {
			return err
		}
		if isMysqlDowngrade {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore MySQL 8.0 backup to MySQL 8.0, 8.4 or higher. But cannot restore to 5.7`)
		}
	case databases.DatabaseTypeMariadb:
		if requestDTO.MariadbDatabase == nil {
			return errors.New("mariadb database configuration is required for restore")
		}
		isMariadbDowngrade, err := tools.IsMariadbBackupVersionHigherThanRestoreVersion(
			backupDatabase.Mariadb.Version,
			requestDTO.MariadbDatabase.Version,
		)
		if err != nil {
			return err
		}
		if isMariadbDowngrade {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore MariaDB 10.11 backup to MariaDB 10.11, 11.4 or higher. But cannot restore to 10.6`)
		}
	case databases.DatabaseTypeMongodb:
		if requestDTO.MongodbDatabase == nil {
			return errors.New("mongodb database configuration is required for restore")
		}
		isMongodbDowngrade, err := tools.IsMongodbBackupVersionHigherThanRestoreVersion(
			backupDatabase.Mongodb.Version,
			requestDTO.MongodbDatabase.Version,
		)
		if err != nil {
			return err
		}
		if isMongodbDowngrade {
			return errors.New(`backup database version is higher than restore database version. ` +
				`Should be restored to the same version as the backup database or higher. ` +
				`For example, you can restore MongoDB 6.0 backup to MongoDB 6.0, 7.0 or higher. But cannot restore to 5.0`)
		}
	}

	return nil
}

// The restore target arrives as bare engine models in the request DTO, so it is assembled into the
// Database the generic tunnel dispatcher works from, the same way the restorer itself does.
func (s *RestoreService) populateRestoreTargetVersionThroughTunnel(
	ctx context.Context,
	backupDatabase *databases.Database,
	requestDTO restores_core.RestoreBackupRequest,
) error {
	restoreTarget := &databases.Database{
		Type:              backupDatabase.Type,
		PostgresqlLogical: requestDTO.PostgresqlLogicalDatabase,
		Mysql:             requestDTO.MysqlDatabase,
		Mariadb:           requestDTO.MariadbDatabase,
		Mongodb:           requestDTO.MongodbDatabase,
	}

	tunneledDatabase, err := databases.OpenTunnel(ctx, databases.OpenTunnelSpec{
		Database:  restoreTarget,
		Logger:    s.logger,
		Encryptor: s.fieldEncryptor,
	})
	if err != nil {
		return err
	}

	defer tunneledDatabase.Close()

	if err := tunneledDatabase.GetDatabaseThroughTunnel().PopulateVersion(
		s.logger,
		s.fieldEncryptor,
	); err != nil {
		return err
	}

	tunneledDatabase.CopyDiscoveredMetadataToOriginal()

	return nil
}

func (s *RestoreService) validateDiskSpace(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	requestDTO restores_core.RestoreBackupRequest,
) error {
	// Only validate disk space for PostgreSQL when file-based restore is needed:
	// - CPU > 1 (parallel jobs require file)
	// - IsExcludeExtensions (TOC filtering requires file)
	// Other databases and PostgreSQL with CPU=1 without extension exclusion stream directly
	if requestDTO.PostgresqlLogicalDatabase == nil {
		return nil
	}

	needsFileBased := requestDTO.PostgresqlLogicalDatabase.CpuCount > 1 ||
		requestDTO.PostgresqlLogicalDatabase.IsExcludeExtensions
	if !needsFileBased {
		return nil
	}

	diskUsage, err := s.diskService.GetDiskUsage(ctx)
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Convert backup size from MB to bytes
	backupSizeBytes := int64(backup.BackupSizeMb * 1024 * 1024)

	// Calculate required space: backup size + 10% buffer
	bufferBytes := int64(float64(backupSizeBytes) * 0.1)
	requiredBytes := backupSizeBytes + bufferBytes

	// Ensure minimum of 1 GB total (even if backup is small)
	minRequiredBytes := int64(1024 * 1024 * 1024) // 1 GB
	if requiredBytes < minRequiredBytes {
		requiredBytes = minRequiredBytes
	}

	// Check if there's enough free space
	if diskUsage.FreeSpaceBytes < requiredBytes {
		backupSizeGB := float64(backupSizeBytes) / (1024 * 1024 * 1024)
		bufferSizeGB := float64(bufferBytes) / (1024 * 1024 * 1024)
		requiredGB := float64(requiredBytes) / (1024 * 1024 * 1024)
		availableGB := float64(diskUsage.FreeSpaceBytes) / (1024 * 1024 * 1024)

		return fmt.Errorf(
			"to restore this backup, %.1f GB (%.1f GB backup + %.1f GB buffer) is required, but only %.1f GB is available. Please free up disk space before restoring",
			requiredGB,
			backupSizeGB,
			bufferSizeGB,
			availableGB,
		)
	}

	return nil
}

func (s *RestoreService) validateNoParallelRestores(databaseID uuid.UUID) error {
	inProgressRestores, err := s.restoreRepository.FindInProgressRestoresByDatabaseID(databaseID)
	if err != nil {
		return fmt.Errorf("failed to check for in-progress restores: %w", err)
	}

	isInProgress := len(inProgressRestores) > 0
	if isInProgress {
		return errors.New(
			"another restore is already in progress for this database. Please wait for it to complete or cancel it before starting a new restore",
		)
	}

	return nil
}
