package usecases_physical_postgresql

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	storages_bridge "databasus-backend/internal/features/storages"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
)

func newFullBackupProcessSpec(t *testing.T, fixture *PhysicalDBFixture) FullBackupSpec {
	t.Helper()
	fullBackup, err := physical_repositories.GetFullBackupRepository().FindByID(fixture.BackupID)
	require.NoError(t, err)
	return FullBackupSpec{
		CommonBackupSpec: CommonBackupSpec{
			ClientTools: pgshared.ClientTools{
				Root: t.TempDir(),
			},
			SourceDB:     fixture.SourceDB,
			DatabaseName: fixture.DB.Name,
			StorageID:    fixture.Storage.ID,
			FileStore:    storages_bridge.GetModule(),
			Encryption:   fullBackup.Encryption,
			FullRepo:     physical_repositories.GetFullBackupRepository(),
			HistoryRepo:  physical_repositories.GetWalHistoryRepository(),
			Logger:       slog.New(slog.DiscardHandler),
		}, Backup: fullBackup,
	}
}

func TestFullBackupClientProcess(t *testing.T) {
	t.Run("ends the tool and its child when the attempt is cancelled", func(t *testing.T) {
		fixture := SetupPhysicalDBForBackup(t)
		spec := newFullBackupProcessSpec(t, fixture)
		pidPath := filepath.Join(t.TempDir(), "pid")
		childPath := filepath.Join(t.TempDir(), "child")
		toolstub.Write(
			t,
			spec.ClientTools.GetExecutable(spec.SourceDB.Version, pgshared.ExecutablePgBasebackup),
			toolstub.Spec{
				Script: fmt.Sprintf(
					"trap 'wait; exit' INT\npython3 -c 'import os, signal, sys, time; signal.signal(signal.SIGINT, lambda *_: sys.exit(0)); open(\"%s\", \"w\").write(str(os.getpid())); time.sleep(600)' &\necho $$ > %q\nwait",
					childPath,
					pidPath,
				),
			},
		)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		completed := make(chan PhysicalBackupResult, 1)
		failures := make(chan error, 1)
		go func() {
			backup, err := NewCreateFullBackupUsecase().Execute(ctx, spec)
			completed <- backup
			failures <- err
		}()
		toolPID := waitForClientPID(t, pidPath)
		childPID := waitForClientPID(t, childPath)
		cancelledAt := time.Now()
		cancel()
		backup := <-completed
		require.NoError(t, <-failures)
		assert.NotEqual(t, physical_enums.PhysicalBackupStatusCompleted, backup.Status)
		assert.Empty(t, backup.Receipts)
		testwait.Until(t, "the tool and its child exit", func(context.Context) error {
			if syscall.Kill(toolPID, 0) == nil || syscall.Kill(childPID, 0) == nil {
				return fmt.Errorf("backup processes are still running")
			}
			return nil
		})
		assert.Less(t, time.Since(cancelledAt), 10*time.Second)
	})
	t.Run("returns the server rejection without the tool directory when the tool fails", func(t *testing.T) {
		fixture := SetupPhysicalDBForBackup(t)
		spec := newFullBackupProcessSpec(t, fixture)
		toolstub.Write(
			t,
			spec.ClientTools.GetExecutable(spec.SourceDB.Version, pgshared.ExecutablePgBasebackup),
			toolstub.Spec{Script: "echo 'FATAL: no pg_hba.conf entry for replication connection' >&2; exit 1"},
		)
		backup, err := NewCreateFullBackupUsecase().Execute(t.Context(), spec)
		require.NoError(t, err)
		assert.Equal(t, physical_enums.PhysicalBackupStatusError, backup.Status)
		assert.Contains(t, backup.ErrorMessage, "exit status 1")
		assert.Contains(t, backup.ErrorMessage, "FATAL: no pg_hba.conf entry for replication connection")
		assert.NotContains(t, backup.ErrorMessage, spec.ClientTools.Root)
	})
}

func waitForClientPID(t *testing.T, path string) int {
	t.Helper()
	var clientPID int
	testwait.Until(t, "client writes its pid", func(context.Context) error {
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		clientPID, err = strconv.Atoi(strings.TrimSpace(string(contents)))
		return err
	})
	t.Cleanup(func() { _ = syscall.Kill(clientPID, syscall.SIGKILL) })
	return clientPID
}
