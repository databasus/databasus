package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/lib/toolstub"
	"databasus-backend/internal/modules/databases"
)

func TestReceiverRestart(t *testing.T) {
	t.Run("ends the previous receiver before restarting after back pressure", func(t *testing.T) {
		fixture := SetupPhysicalDBForBackup(t)
		tools := pgshared.ClientTools{Root: t.TempDir()}
		supervisor := NewWalStreamSupervisor(WalStreamSpec{
			ClientTools: tools, DatabaseID: fixture.DB.ID, SourceDB: fixture.SourceDB,
			WatchDirRoot: t.TempDir(), Logger: slog.New(slog.DiscardHandler),
		})
		require.NoError(t, os.MkdirAll(supervisor.watchDir, 0o700))
		require.NoError(t, supervisor.createSlotUnlessDatabaseDeleted(t.Context(), supervisor.logger))
		t.Cleanup(func() {
			_ = pgphysical.NewSource(fixture.SourceDB.Engine).
				DropSlot(context.Background(), supervisor.logger, supervisor.slotName)
		})
		supervisor.setLocalQueuePauseBytes(1024)
		recordDir := t.TempDir()
		pidPath := filepath.Join(recordDir, "pid")
		childPath := filepath.Join(recordDir, "child")
		interruptedPath := filepath.Join(recordDir, "interrupted")
		overlapPath := filepath.Join(recordDir, "overlap")
		stub := toolstub.Write(
			t,
			tools.GetExecutable(fixture.SourceDB.Version, pgshared.ExecutablePgReceivewal),
			toolstub.Spec{Script: fmt.Sprintf(`
if test -f %[1]q; then
 previous=$(cat %[1]q)
 previous_child=$(cat %[3]q)
 if kill -0 "$previous" 2>/dev/null || kill -0 "$previous_child" 2>/dev/null; then echo overlap > %[2]q; fi
fi
trap 'echo interrupted > %[4]q; sleep 3; wait; exit 0' INT
python3 -c 'import os, signal, sys, time; signal.signal(signal.SIGINT, lambda *_: (time.sleep(3), sys.exit(0))); open("%[3]s", "w").write(str(os.getpid())); time.sleep(600)' &
echo $$ > %[1]q
wait`, pidPath, overlapPath, childPath, interruptedPath)},
		)
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		supervisionDone := make(chan error, 1)
		monitorDone := make(chan struct{})
		go func() { supervisionDone <- supervisor.runReceivewalSupervision(ctx, supervisor.logger) }()
		go func() { defer close(monitorDone); supervisor.runBackpressureMonitor(ctx, supervisor.logger) }()
		firstPID := waitForClientPID(t, pidPath)
		firstChildPID := waitForClientPID(t, childPath)
		segmentPath := writeSegmentOfSize(t, supervisor.watchDir, walName(1, 50), 2048)
		testwait.Until(t, "back pressure interrupts the previous receiver", func(context.Context) error {
			_, err := os.Stat(interruptedPath)
			return err
		})
		require.NoError(t, os.Remove(segmentPath))
		conn := OpenAdminConn(t, fixture)
		require.True(t, SlotExists(t, conn, supervisor.slotName))
		testwait.Until(t, "the next receiver starts after the backlog drains", func(context.Context) error {
			contents, err := os.ReadFile(pidPath)
			if err != nil {
				return err
			}
			receiverPID, err := strconv.Atoi(strings.TrimSpace(string(contents)))
			if err != nil {
				return err
			}
			if receiverPID == firstPID {
				return errors.New("the next receiver has not started")
			}
			return nil
		})
		require.Len(t, stub.GetInvocations(t), 2)
		require.ErrorIs(t, syscall.Kill(firstPID, 0), syscall.ESRCH)
		require.ErrorIs(t, syscall.Kill(firstChildPID, 0), syscall.ESRCH)
		_, overlapErr := os.Stat(overlapPath)
		require.ErrorIs(t, overlapErr, os.ErrNotExist)
		cancel()
		require.NoError(t, <-supervisionDone)
		<-monitorDone
		assert.True(t, SlotExists(t, conn, supervisor.slotName))
	})
	t.Run("honors a restart requested before the run registers", func(t *testing.T) {
		t.Parallel()
		tools := pgshared.ClientTools{Root: t.TempDir()}
		stub := toolstub.Write(
			t,
			tools.GetExecutable(pgshared.Version17, pgshared.ExecutablePgReceivewal),
			toolstub.Spec{},
		)
		supervisor := NewWalStreamSupervisor(WalStreamSpec{
			ClientTools: tools, DatabaseID: uuid.New(), SourceDB: databases.PostgresqlPhysicalConnectionSettings{
				Engine: pgphysical.Settings{
					Host:     "localhost",
					Port:     5432,
					Username: "repl",
				},
				Version: pgshared.Version17,
			}, WatchDirRoot: t.TempDir(), Logger: slog.New(slog.DiscardHandler),
		})
		supervisor.clearRestartRequest()
		supervisor.signalRestart()
		receiver := supervisor.spawnAndSupervise(t.Context(), supervisor.logger)
		assert.Equal(t, receiverInternalRestart, receiver.Exit)
		assert.Empty(t, stub.GetInvocations(t))
		supervisor.clearRestartRequest()
		receiver = supervisor.spawnAndSupervise(t.Context(), supervisor.logger)
		assert.Equal(t, receiverRetryable, receiver.Exit)
		require.Len(t, stub.GetInvocations(t), 1)
	})
}
