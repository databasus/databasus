package usecases_logical

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	databases_bridge "databasus-backend/internal/features/databases"
	storages_bridge "databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

// TestCreateBackup dumps a real database of every engine into a storage that fails. It pins issue
// #816: the storage failure ends the backup context, and the backup was reported as cancelled
// instead of with the storage error.
func TestCreateBackup(t *testing.T) {
	for _, engine := range []struct {
		name           string
		createDatabase func(t *testing.T, workspaceID uuid.UUID) databases.Database
	}{
		{
			name: "PostgreSQL",
			createDatabase: func(_ *testing.T, workspaceID uuid.UUID) databases.Database {
				return databases_bridge.CreateTestDatabase(workspaceID, nil)
			},
		},
		{
			name: "MySQL",
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) databases.Database {
				endpoint := containers.StartMysql(t, "mysql:8.0")

				return databases_bridge.CreateTestMysqlDatabase(endpoint.Host, endpoint.Port, workspaceID, nil)
			},
		},
		{
			name: "MariaDB",
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) databases.Database {
				endpoint := containers.StartMariadb(t, "mariadb:10.11")

				return databases_bridge.CreateTestMariadbDatabase(endpoint.Host, endpoint.Port, workspaceID, nil)
			},
		},
		{
			name: "MongoDB",
			createDatabase: func(t *testing.T, workspaceID uuid.UUID) databases.Database {
				endpoint := containers.StartMongodb(t, "mongo:7.0")

				return databases_bridge.CreateTestMongodbDatabase(endpoint.Host, endpoint.Port, workspaceID, nil)
			},
		},
	} {
		t.Run(engine.name, func(t *testing.T) {
			database := createTestDatabase(t, engine.createDatabase)
			storageErr := fmt.Errorf("failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))

			t.Run("reports the storage error when the storage times out after the dump finished", func(t *testing.T) {
				fileStore := &failingBackupFileStore{failureErr: storageErr, isFailingBeforeReading: false}

				_, err := GetCreateBackupUsecase().Execute(
					t.Context(), newTestBackup(), newUnencryptedBackupConfig(), database, fileStore, func(float64) {},
				)

				require.Positive(t, fileStore.readBytes.Load(), "the dump never reached the storage")
				assertStorageErrorReported(t, err, storageErr)
			})

			t.Run("reports the storage error when the storage fails before reading", func(t *testing.T) {
				fileStore := &failingBackupFileStore{failureErr: storageErr, isFailingBeforeReading: true}

				_, err := GetCreateBackupUsecase().Execute(
					t.Context(), newTestBackup(), newUnencryptedBackupConfig(), database, fileStore, func(float64) {},
				)

				assertStorageErrorReported(t, err, storageErr)
			})
		})
	}
}

func createTestDatabase(
	t *testing.T,
	createDatabase func(t *testing.T, workspaceID uuid.UUID) databases.Database,
) databases.Database {
	t.Helper()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner)
	database := createDatabase(t, workspace.ID)

	t.Cleanup(func() {
		databases_bridge.RemoveTestDatabase(context.WithoutCancel(t.Context()), database)
		workspaces_testing.RemoveTestWorkspace(context.WithoutCancel(t.Context()), workspace)
	})

	return database
}

func assertStorageErrorReported(t *testing.T, backupErr, storageErr error) {
	t.Helper()

	require.ErrorIs(t, backupErr, storageErr)
	assert.Contains(t, backupErr.Error(), "save to storage")
	assert.NotContains(t, backupErr.Error(), "backup cancelled")
}

func newTestBackup() *backups_core_logical.LogicalBackup {
	backupID := uuid.New()

	return &backups_core_logical.LogicalBackup{
		ID:         backupID,
		DatabaseID: uuid.New(),
		StorageID:  uuid.New(),
		FileName:   "test-backup-" + backupID.String(),
		Status:     backups_core_logical.BackupStatusInProgress,
	}
}

func newUnencryptedBackupConfig() backups.LogicalBackupConfig {
	return backups.LogicalBackupConfig{
		BackupConfig: backups.BackupConfig{
			Encryption: backups.EncryptionNone,
		},
	}
}

// failingBackupFileStore fails every write with failureErr. It either drains the whole stream
// first, so the dump runs to completion before the storage fails, or fails before reading a byte,
// so the storage fails while the dump is still producing data.
type failingBackupFileStore struct {
	failureErr             error
	isFailingBeforeReading bool
	readBytes              atomic.Int64
}

func (s *failingBackupFileStore) WriteFile(
	_ context.Context,
	_ storages.StoredFileReference,
	file io.Reader,
) (storages.WriteReceipt, error) {
	if !s.isFailingBeforeReading {
		readBytes, _ := io.Copy(io.Discard, file)
		s.readBytes.Store(readBytes)
	}

	return storages.WriteReceipt{}, s.failureErr
}

func (s *failingBackupFileStore) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	return storages_bridge.StartFakeBackgroundWrite(ctx, s.WriteFile, spec)
}
