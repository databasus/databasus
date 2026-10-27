package backuping_physical

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	storages_bridge "databasus-backend/internal/features/storages"
	"databasus-backend/internal/lib/toolstub"
)

func TestFullBackupToolFailure(t *testing.T) {
	t.Run("records the exit status and server rejection without the tool directory", func(t *testing.T) {
		fixture := postgresql_executor.SetupPhysicalDBForBackup(t)
		tools := pgshared.ClientTools{Root: t.TempDir()}
		toolstub.Write(
			t,
			tools.GetExecutable(fixture.SourceDB.Version, pgshared.ExecutablePgBasebackup),
			toolstub.Spec{Script: "echo 'FATAL: no pg_hba.conf entry for replication connection' >&2; exit 1"},
		)
		fullRepo := physical_repositories.GetFullBackupRepository()
		fullBackup, err := fullRepo.FindByID(fixture.BackupID)
		require.NoError(t, err)
		backup, err := postgresql_executor.NewCreateFullBackupUsecase().
			Execute(t.Context(), postgresql_executor.FullBackupSpec{
				CommonBackupSpec: postgresql_executor.CommonBackupSpec{
					ClientTools:  tools,
					SourceDB:     fixture.SourceDB,
					DatabaseName: fixture.DB.Name,
					StorageID:    fixture.Storage.ID,
					FileStore:    storages_bridge.GetModule(),
					Encryption:   fullBackup.Encryption,
					FullRepo:     fullRepo,
					HistoryRepo:  physical_repositories.GetWalHistoryRepository(),
					Logger:       slog.New(slog.DiscardHandler),
				}, Backup: fullBackup,
			})
		require.NoError(t, err)
		require.NoError(t, CreateTestPhysicalBackuper(nil).persistFullResult(t.Context(), fullBackup, backup, nil))
		persistedBackup, err := fullRepo.FindByID(fixture.BackupID)
		require.NoError(t, err)
		require.Equal(t, physical_enums.PhysicalBackupStatusError, persistedBackup.Status)
		require.NotNil(t, persistedBackup.FailMessage)
		assert.Contains(t, *persistedBackup.FailMessage, "exit status 1")
		assert.Contains(t, *persistedBackup.FailMessage, "FATAL: no pg_hba.conf entry for replication connection")
		assert.NotContains(t, *persistedBackup.FailMessage, tools.Root)
	})
}
