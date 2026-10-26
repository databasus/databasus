//go:build testhelpers

package restoring

import (
	"context"
	"errors"
	"io"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	"databasus-backend/internal/modules/storages"
)

type MockSuccessRestoreUsecase struct{}

func (uc *MockSuccessRestoreUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	return nil
}

type MockFailedRestoreUsecase struct {
	RestoreErr error
}

func (uc *MockFailedRestoreUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	return uc.RestoreErr
}

type MockCaptureCredentialsRestoreUsecase struct {
	CalledChan    chan restores_core.RestoreBackupRequest
	ShouldSucceed bool
}

func (uc *MockCaptureCredentialsRestoreUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	uc.CalledChan <- target

	if uc.ShouldSucceed {
		return nil
	}
	return errors.New("mock restore failed")
}

type MockBlockingRestoreUsecase struct {
	StartedChan chan bool
}

func (uc *MockBlockingRestoreUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	if uc.StartedChan != nil {
		uc.StartedChan <- true
	}

	<-ctx.Done()

	return ctx.Err()
}

type MockCaptureStorageRestoreUsecase struct {
	ReadBackupContent []byte
}

func (uc *MockCaptureStorageRestoreUsecase) Execute(
	ctx context.Context,
	backupConfig *backups_config_logical.LogicalBackupConfig,
	restore restores_core.Restore,
	source restores_core.SourceDatabase,
	target restores_core.RestoreBackupRequest,
	backup *backups_core_logical.LogicalBackup,
	storageReader storages.StorageReader,
	options restores_core.RestoreOptions,
) error {
	backupFile, err := storageReader.OpenFile(ctx, backup.FileName)
	if err != nil {
		return err
	}
	defer func() { _ = backupFile.Close() }()

	uc.ReadBackupContent, err = io.ReadAll(backupFile)

	return err
}
