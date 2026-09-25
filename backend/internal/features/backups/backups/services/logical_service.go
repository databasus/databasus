package backups_services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"

	audit_logs "databasus-backend/internal/features/audit_logs"
	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	"databasus-backend/internal/features/backups/backups/download/download_token"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
	"databasus-backend/internal/features/backups/backups/encryption"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	users_models "databasus-backend/internal/features/users/models"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_encryption "databasus-backend/internal/util/encryption"
	files_utils "databasus-backend/internal/util/files"
)

type LogicalBackupService struct {
	databaseService     *databases.DatabaseService
	storageService      *storages.StorageService
	backupRepository    *backups_core_logical.BackupRepository
	notifierService     *notifiers.NotifierService
	notificationSender  backups_core_logical.NotificationSender
	backupConfigService *backups_config_logical.BackupConfigService
	secretKeyService    *encryption_secrets.SecretKeyService
	fieldEncryptor      util_encryption.FieldEncryptor

	createBackupUseCase backups_core_logical.CreateBackupUsecase

	logger *slog.Logger

	backupRemoveListeners []backups_core_logical.BackupRemoveListener

	workspaceService          *workspaces_services.WorkspaceService
	auditLogService           *audit_logs.AuditLogService
	taskCancellationRequester *task_cancellation.Requester
	downloadTokenService      *download_token.Service
	backupSchedulerService    *backuping_logical.BackupsScheduler
	backupCleaner             *backuping_logical.BackupCleaner
}

func (s *LogicalBackupService) AddBackupRemoveListener(listener backups_core_logical.BackupRemoveListener) {
	s.backupRemoveListeners = append(s.backupRemoveListeners, listener)
}

func (s *LogicalBackupService) HasSuccessfulBackupSince(
	databaseID uuid.UUID,
	since time.Time,
) (bool, error) {
	return s.backupRepository.ExistsCompletedSince(databaseID, since)
}

func (s *LogicalBackupService) GetLatestCompletedBackup(
	databaseID uuid.UUID,
) (*backups_core_logical.LogicalBackup, error) {
	return s.backupRepository.FindLatestCompleted(databaseID)
}

func (s *LogicalBackupService) GetBackupTotalsByDatabaseIDs(
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]backups_core_logical.DatabaseBackupTotals, error) {
	totals, err := s.backupRepository.GetTotalsByDatabaseIDs(databaseIDs)
	if err != nil {
		return nil, err
	}

	totalsByDatabaseID := make(map[uuid.UUID]backups_core_logical.DatabaseBackupTotals, len(totals))
	for _, databaseTotals := range totals {
		totalsByDatabaseID[databaseTotals.DatabaseID] = databaseTotals
	}

	return totalsByDatabaseID, nil
}

func (s *LogicalBackupService) GetInstallationBackupTotals() (backups_core_logical.BackupTotals, error) {
	return s.backupRepository.GetInstallationTotals()
}

func (s *LogicalBackupService) OnBeforeBackupsStorageChange(ctx context.Context, databaseID uuid.UUID) error {
	err := s.deleteDbBackups(ctx, databaseID)
	if err != nil {
		return err
	}

	return nil
}

// The logical_backups rows are the only record of the file names in a storage, so
// the storage cannot go while they exist.
func (s *LogicalBackupService) GetStorageBackupReferenceCount(storageID uuid.UUID) (int64, error) {
	return s.backupRepository.CountByStorageID(storageID)
}

func (s *LogicalBackupService) OnBeforeDatabaseRemove(ctx context.Context, databaseID uuid.UUID) error {
	err := s.deleteDbBackups(ctx, databaseID)
	if err != nil {
		return err
	}

	return nil
}

func (s *LogicalBackupService) MakeBackupWithAuth(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
) error {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot create backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canAccess {
		return errors.New("insufficient permissions to create backup for this database")
	}

	s.backupSchedulerService.StartBackup(ctx, database, true)

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Backup manually initiated for database: %s", database.Name),
		UserID:      &user.ID,
		WorkspaceID: database.WorkspaceID,
	})

	return nil
}

func (s *LogicalBackupService) GetBackups(
	ctx context.Context,
	user *users_models.User,
	databaseID uuid.UUID,
	limit, offset int,
	filters *backups_core_logical.BackupFilters,
) (*backups_dto_logical.GetBackupsResponse, error) {
	database, err := s.databaseService.GetDatabaseByID(databaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot get backups for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to access backups for this database")
	}

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	backups, err := s.backupRepository.FindByDatabaseIDWithFiltersAndPagination(
		databaseID, filters, limit, offset,
	)
	if err != nil {
		return nil, err
	}

	total, err := s.backupRepository.CountByDatabaseIDWithFilters(databaseID, filters)
	if err != nil {
		return nil, err
	}

	return &backups_dto_logical.GetBackupsResponse{
		Backups: backups,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}

func (s *LogicalBackupService) DeleteBackup(
	ctx context.Context,
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot delete backup for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete backup for this database")
	}

	if backup.Status == backups_core_logical.BackupStatusInProgress {
		return errors.New("backup is in progress")
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Backup deleted for database: %s", database.Name),
		UserID:      &user.ID,
		WorkspaceID: database.WorkspaceID,
	})

	return s.backupCleaner.DeleteBackup(ctx, backup)
}

func (s *LogicalBackupService) GetBackup(backupID uuid.UUID) (*backups_core_logical.LogicalBackup, error) {
	return s.backupRepository.FindByID(backupID)
}

func (s *LogicalBackupService) SetRestoreVerificationStatus(
	backupID uuid.UUID,
	status backups_core_logical.RestoreVerificationStatus,
) error {
	return s.backupRepository.UpdateRestoreVerificationStatus(backupID, status)
}

func (s *LogicalBackupService) CancelBackup(
	ctx context.Context,
	user *users_models.User,
	backupID uuid.UUID,
) error {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return err
	}

	if database.WorkspaceID == nil {
		return errors.New("cannot cancel backup for database without workspace")
	}

	canManage, err := s.workspaceService.CanUserManageDBs(ctx, *database.WorkspaceID, user)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to cancel backup for this database")
	}

	if backup.Status != backups_core_logical.BackupStatusInProgress {
		return errors.New("backup is not in progress")
	}

	if err := s.taskCancellationRequester.RequestCancellation(ctx, backupID); err != nil {
		return err
	}

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Backup cancelled for database: %s", database.Name),
		UserID:      &user.ID,
		WorkspaceID: database.WorkspaceID,
	})

	return nil
}

func (s *LogicalBackupService) GetBackupReader(
	ctx context.Context,
	backupID uuid.UUID,
) (io.ReadCloser, error) {
	logger := s.logger.With("backup_id", backupID)

	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find backup: %w", err)
	}

	storage, err := s.storageService.GetStorageByID(ctx, backup.StorageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage: %w", err)
	}

	fileReader, err := storage.GetFile(ctx, s.fieldEncryptor, logger, backup.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backup file: %w", err)
	}

	if backup.Encryption == backups_core_enums.BackupEncryptionNone {
		logger.InfoContext(ctx, "returning non-encrypted backup")
		return fileReader, nil
	}

	if backup.Encryption != backups_core_enums.BackupEncryptionEncrypted {
		if err := fileReader.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close file reader", "error", err)
		}
		return nil, fmt.Errorf("unsupported encryption type: %s", backup.Encryption)
	}

	if backup.EncryptionSalt == nil || backup.EncryptionIV == nil {
		if err := fileReader.Close(); err != nil {
			logger.ErrorContext(ctx, "failed to close file reader", "error", err)
		}
		return nil, fmt.Errorf("backup marked as encrypted but missing encryption metadata")
	}

	masterKey, err := s.secretKeyService.GetSecretKey()
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to get master key: %w", err)
	}

	salt, err := base64.StdEncoding.DecodeString(*backup.EncryptionSalt)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	iv, err := base64.StdEncoding.DecodeString(*backup.EncryptionIV)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to decode IV: %w", err)
	}

	decryptionReader, err := encryption.NewDecryptionReader(
		fileReader,
		masterKey,
		backup.ID,
		salt,
		iv,
	)
	if err != nil {
		if closeErr := fileReader.Close(); closeErr != nil {
			logger.ErrorContext(ctx, "failed to close file reader", "error", closeErr)
		}
		return nil, fmt.Errorf("failed to create decrypting reader: %w", err)
	}

	logger.InfoContext(ctx, "returning encrypted backup with decryption")

	return &backups_dto_logical.DecryptionReaderCloser{
		DecryptionReader: decryptionReader,
		BaseReader:       fileReader,
	}, nil
}

func (s *LogicalBackupService) GenerateDownloadToken(
	ctx context.Context,
	user *users_models.User,
	backupID uuid.UUID,
) (*download_token.GenerateTokenResponse, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, err
	}

	if database.WorkspaceID == nil {
		return nil, errors.New("cannot download backup for database without workspace")
	}

	canAccess, _, err := s.workspaceService.CanUserAccessWorkspace(ctx, *database.WorkspaceID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to download backup for this database")
	}

	token, err := s.downloadTokenService.Generate(ctx, backupID, user.ID)
	if err != nil {
		return nil, err
	}

	filename := s.generateBackupFilename(backup, database)

	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Download token generated for backup of database: %s", database.Name),
		UserID:      &user.ID,
		WorkspaceID: database.WorkspaceID,
	})

	return &download_token.GenerateTokenResponse{
		Token:    token,
		Filename: filename,
		BackupID: backupID,
	}, nil
}

func (s *LogicalBackupService) ValidateDownloadToken(
	ctx context.Context,
	token string,
) (*download_token.Token, error) {
	return s.downloadTokenService.ValidateAndConsume(ctx, token)
}

func (s *LogicalBackupService) GetLatestVerifiableBackup(
	databaseID uuid.UUID,
) (*backups_core_logical.LogicalBackup, error) {
	return s.backupRepository.FindLatestCompleted(databaseID)
}

func (s *LogicalBackupService) GetBackupFileWithoutAuth(
	ctx context.Context,
	backupID uuid.UUID,
) (io.ReadCloser, *backups_core_logical.LogicalBackup, *databases.Database, error) {
	backup, err := s.backupRepository.FindByID(backupID)
	if err != nil {
		return nil, nil, nil, err
	}

	database, err := s.databaseService.GetDatabaseByID(backup.DatabaseID)
	if err != nil {
		return nil, nil, nil, err
	}

	reader, err := s.GetBackupReader(ctx, backupID)
	if err != nil {
		return nil, nil, nil, err
	}

	return reader, backup, database, nil
}

func (s *LogicalBackupService) WriteAuditLogForDownload(
	ctx context.Context,
	userID uuid.UUID,
	backup *backups_core_logical.LogicalBackup,
	database *databases.Database,
) {
	s.auditLogService.WriteAuditLog(ctx, audit_logs_models.AuditEntry{
		Message:     fmt.Sprintf("Backup file downloaded for database: %s", database.Name),
		UserID:      &userID,
		WorkspaceID: database.WorkspaceID,
	})
}

func (s *LogicalBackupService) RefreshDownloadLock(ctx context.Context, userID uuid.UUID) {
	s.downloadTokenService.RefreshDownloadLock(ctx, userID)
}

func (s *LogicalBackupService) ReleaseDownloadLock(ctx context.Context, userID uuid.UUID) {
	s.downloadTokenService.ReleaseDownloadLock(ctx, userID)
}

func (s *LogicalBackupService) IsDownloadInProgress(userID uuid.UUID) bool {
	return s.downloadTokenService.IsDownloadInProgress(context.Background(), userID)
}

func (s *LogicalBackupService) deleteDbBackups(ctx context.Context, databaseID uuid.UUID) error {
	dbBackupsInProgress, err := s.backupRepository.FindByDatabaseIdAndStatus(
		databaseID,
		backups_core_logical.BackupStatusInProgress,
	)
	if err != nil {
		return err
	}

	if len(dbBackupsInProgress) > 0 {
		return errors.New("backup is in progress, storage cannot be removed")
	}

	dbBackups, err := s.backupRepository.FindByDatabaseID(
		databaseID,
	)
	if err != nil {
		return err
	}

	for _, dbBackup := range dbBackups {
		err := s.backupCleaner.DeleteBackup(ctx, dbBackup)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *LogicalBackupService) generateBackupFilename(
	backup *backups_core_logical.LogicalBackup,
	database *databases.Database,
) string {
	timestamp := backup.CreatedAt.Format("2006-01-02_15-04-05")
	safeName := files_utils.SanitizeFilename(database.Name)
	extension := s.getBackupExtension(database.Type)
	return fmt.Sprintf("%s_backup_%s%s", safeName, timestamp, extension)
}

func (s *LogicalBackupService) getBackupExtension(dbType databases.DatabaseType) string {
	switch dbType {
	case databases.DatabaseTypeMysql, databases.DatabaseTypeMariadb:
		return ".sql.zst"
	case databases.DatabaseTypePostgresLogical:
		return ".dump"
	case databases.DatabaseTypeMongodb:
		return ".archive"
	default:
		return ".backup"
	}
}
