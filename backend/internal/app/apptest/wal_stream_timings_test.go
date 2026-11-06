package apptest

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

func TestWalStreamTimings(t *testing.T) {
	t.Parallel()

	t.Run("archives WAL from a quiet source when the application shortens forced rotation", func(t *testing.T) {
		t.Parallel()

		timings := backups.ProductionWalStreamTimings()
		timings.ForcedRotationInterval = 2 * time.Second
		observer := walRotationObserver{Handler: slog.DiscardHandler, rotations: make(chan time.Time, 1)}
		startedAt := time.Now()
		testApp := New(t, WithPhysicalBackupLoops(), WithWalStreamTimings(timings), WithLogSink(observer))
		source := containers.StartPhysicalPostgres(t, "postgres:17")
		owner, database := enablePhysicalBackupLoops(t, testApp, source)

		testwait.Until(t, "the WAL receiver owns the source slot", func(context.Context) error {
			if getActivePhysicalReplicationSlotCount(t, source) != 1 {
				return fmt.Errorf("no active WAL receiver")
			}
			return nil
		})
		testwait.Until(
			t,
			"the initial full backup completes before checking forced rotation",
			func(context.Context) error {
				for _, backup := range listPhysicalBackups(t, testApp, owner, database.Id) {
					if backup.Type == "FULL" && backup.Status == "COMPLETED" {
						return nil
					}
				}
				return fmt.Errorf("the initial full backup has not completed")
			},
		)

		rotatedAt := observer.waitForRotation(t)
		require.Less(t, rotatedAt.Sub(startedAt), backups.ProductionWalStreamTimings().ForcedRotationInterval)

		conn := openPhysicalSource(t, source)
		var quietWalFilename string
		require.NoError(
			t,
			conn.QueryRow(t.Context(), "SELECT pg_walfile_name(pg_current_wal_lsn())").Scan(&quietWalFilename),
		)
		_, err := conn.Exec(t.Context(), "CREATE TABLE quiet_source (id int)")
		require.NoError(t, err)

		testwait.Until(t, "forced rotation archives a segment from the quiet source", func(context.Context) error {
			for _, backup := range listPhysicalBackups(t, testApp, owner, database.Id) {
				if backup.Type == "WAL" && backup.ToWalFilename != nil && *backup.ToWalFilename >= quietWalFilename {
					return nil
				}
			}
			return fmt.Errorf("no archived WAL segment")
		})
		assert.Equal(t, 5*time.Minute, backups.ProductionWalStreamTimings().ForcedRotationInterval)
	})
}

type walRotationObserver struct {
	slog.Handler
	rotations chan time.Time
}

func (observer walRotationObserver) Handle(_ context.Context, record slog.Record) error {
	if record.Message == "forced a wal segment switch so the recovery point keeps advancing" {
		select {
		case observer.rotations <- record.Time:
		default:
		}
	}

	return nil
}

func (observer walRotationObserver) WithAttrs(attrs []slog.Attr) slog.Handler {
	observer.Handler = observer.Handler.WithAttrs(attrs)
	return observer
}

func (observer walRotationObserver) WithGroup(name string) slog.Handler {
	observer.Handler = observer.Handler.WithGroup(name)
	return observer
}

func (observer walRotationObserver) waitForRotation(t *testing.T) time.Time {
	t.Helper()
	var rotatedAt time.Time
	testwait.Until(
		t,
		"the streamer successfully forces a WAL switch with the configured interval",
		func(context.Context) error {
			select {
			case rotatedAt = <-observer.rotations:
				return nil
			default:
				return fmt.Errorf("no successful forced WAL switch")
			}
		},
	)
	return rotatedAt
}
