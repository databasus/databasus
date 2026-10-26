package usecases_physical_postgresql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/modules/databases"
)

const (
	// startupCleanupTimeout caps how long a single DB's cleanup may block
	// startup. Source PG might be unreachable; we log + skip rather than
	// hang the boot.
	startupCleanupTimeout = 5 * time.Second

	// The backup has already finished either way, so the second attempt at dropping
	// the slot must not hold its result hostage to an unreachable source.
	slotDropRetryTimeout = 10 * time.Second

	// startupCleanupConcurrency bounds parallel cleanup. With 100+ DBs
	// configured, unbounded fan-out would saturate Databasus's connection
	// pool. 10 keeps total boot delay near the slowest 10% of sources.
	startupCleanupConcurrency = 10
)

// WithBackupSlot creates a fresh per-backup replication slot on the source
// PG, runs fn, then drops the slot. Drop-if-exists at the start picks up
// any orphan from a previous crashed backup; defer drop covers the happy
// path and most failure paths.
func WithBackupSlot(
	ctx context.Context,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	logger *slog.Logger,
	fn func() error,
) error {
	conn, err := openInspectionConn(ctx, sourceDB)
	if err != nil {
		return fmt.Errorf("open conn for backup slot: %w", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	slotName := sourceDB.BackupSlotName

	if err := dropBackupSlotIfExists(ctx, conn, slotName); err != nil {
		return fmt.Errorf("pre-create drop of backup slot %q: %w", slotName, err)
	}

	if _, err := conn.Exec(
		ctx,
		"SELECT pg_create_physical_replication_slot($1, true)",
		slotName,
	); err != nil {
		if dropErr := dropBackupSlotAfterFailedCreate(sourceDB, logger); dropErr != nil {
			logger.WarnContext(
				ctx,
				"slot drop after a failed create failed; will be recovered by next backup or startup cleanup",
				"slot_name", slotName,
				"error", dropErr,
			)
		}

		return fmt.Errorf("create backup slot %q: %w", slotName, err)
	}

	logger.InfoContext(ctx, "per-backup replication slot created", "slot_name", slotName)

	defer func() {
		// Background context so defer runs even when ctx is cancelled.
		if dropErr := dropBackupSlotIfExists(context.Background(), conn, slotName); dropErr == nil {
			logger.InfoContext(ctx, "per-backup replication slot dropped", "slot_name", slotName)

			return
		}

		// The held connection is the first casualty of a network break, and a bastion in front of the
		// source makes those routine. Leaving the slot pins WAL on the source until the next backup
		// runs, which on a weekly full is a week of retained segments.
		if retryErr := dropBackupSlotOverFreshConn(sourceDB, slotName); retryErr != nil {
			logger.WarnContext(
				ctx,
				"post-backup slot drop failed; will be recovered by next backup or startup cleanup",
				"slot_name", slotName,
				"error", retryErr,
			)
		}
	}()

	return fn()
}

// dropBackupSlotAfterFailedCreate drops a slot the server may have created after the client gave
// up: a cancellation breaks the held connection while the server's session still holds the slot,
// so the slot is dropped over a fresh connection once that session lets go of it.
func dropBackupSlotAfterFailedCreate(
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	logger *slog.Logger,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), slotDropRetryTimeout)
	defer cancel()

	backupSlotSettings := sourceDB.Engine
	backupSlotSettings.ReplicationSlotName = sourceDB.BackupSlotName

	return pgphysical.NewSource(backupSlotSettings).DropSlotForRemoval(ctx, logger)
}

func dropBackupSlotOverFreshConn(
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	slotName string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), slotDropRetryTimeout)
	defer cancel()

	conn, err := openInspectionConn(ctx, sourceDB)
	if err != nil {
		return fmt.Errorf("reopen conn to drop backup slot %q: %w", slotName, err)
	}
	defer func() { _ = conn.Close(context.Background()) }()

	return dropBackupSlotIfExists(ctx, conn, slotName)
}

func dropBackupSlotIfExists(ctx context.Context, conn *pgx.Conn, slotName string) error {
	_, err := conn.Exec(
		ctx,
		`SELECT pg_drop_replication_slot(slot_name)
		   FROM pg_replication_slots WHERE slot_name = $1`,
		slotName,
	)

	return err
}

// RunStartupCleanup iterates every physical PG database and drops the per-backup
// replication slot left behind by a crash, EXCEPT where isSlotProtected reports
// the slot is still in use by an in-flight backup — a blind drop would unpin the
// WAL a running pg_basebackup still needs. Runs once at boot.
//
// Cleanup never blocks boot beyond startupCleanupTimeout per DB; unreachable
// sources and protected slots are logged and skipped.
func RunStartupCleanup(
	ctx context.Context,
	logger *slog.Logger,
	isSlotProtected func(databaseID uuid.UUID) (bool, error),
) error {
	allDatabases, err := databases_bridge.GetModule().ListAllDatabases(ctx)
	if err != nil {
		return fmt.Errorf("list databases for slot cleanup: %w", err)
	}

	sem := make(chan struct{}, startupCleanupConcurrency)

	var (
		wg           sync.WaitGroup
		droppedCount sync.Map
		skippedCount sync.Map
		failureCount sync.Map
	)

	for _, listedDatabase := range allDatabases {
		if listedDatabase.Type != databases.TypePostgresqlPhysical {
			continue
		}

		connectionSettings, err := databases_bridge.GetModule().GetPostgresqlPhysicalConnectionSettings(
			ctx, listedDatabase.ID,
		)
		if err != nil {
			logger.WarnContext(ctx, "startup slot cleanup: skip, cannot read the connection settings",
				"database_id", listedDatabase.ID, "error", err)
			skippedCount.Store(listedDatabase.ID, struct{}{})

			continue
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(connectionSettings databases.PostgresqlPhysicalConnectionSettings) {
			defer wg.Done()
			defer func() { <-sem }()

			databaseID := connectionSettings.DatabaseID
			slotName := connectionSettings.BackupSlotName
			scopedLogger := logger.With("database_id", databaseID, "slot_name", slotName)

			// Decided from the in-flight claim, not the source PG, so it runs
			// before we open a connection.
			protected, protErr := isSlotProtected(databaseID)
			if protErr != nil {
				scopedLogger.Warn("startup slot cleanup: skip, cannot determine owner liveness", "error", protErr)
				skippedCount.Store(databaseID, struct{}{})
				return
			}

			if protected {
				scopedLogger.Info("startup slot cleanup: skip, backup still in flight")
				skippedCount.Store(databaseID, struct{}{})
				return
			}

			cleanupCtx, cancel := context.WithTimeout(ctx, startupCleanupTimeout)
			defer cancel()

			// A bastioned source has to fit its SSH handshake into the same budget. Missing it is a
			// skip, not a leak: WithBackupSlot drops any survivor before it creates the next one.
			tunnel, err := pgphysical.NewSource(connectionSettings.Engine).OpenTunnel(cleanupCtx, scopedLogger)
			if err != nil {
				scopedLogger.Warn("startup slot cleanup: skip unreachable bastion", "error", err)
				skippedCount.Store(databaseID, struct{}{})
				return
			}
			defer tunnel.Close()

			conn, err := tunnel.GetSource().OpenInspectionConn(cleanupCtx)
			if err != nil {
				scopedLogger.Warn("startup slot cleanup: skip unreachable source", "error", err)
				skippedCount.Store(databaseID, struct{}{})
				return
			}
			defer func() { _ = conn.Close(context.Background()) }()

			if err := dropBackupSlotIfExists(cleanupCtx, conn, slotName); err != nil {
				scopedLogger.Warn("startup slot cleanup: drop failed", "error", err)
				failureCount.Store(databaseID, struct{}{})
				return
			}

			droppedCount.Store(databaseID, struct{}{})
		}(connectionSettings)
	}

	wg.Wait()

	var dropped, skipped, failed int
	droppedCount.Range(func(_, _ any) bool { dropped++; return true })
	skippedCount.Range(func(_, _ any) bool { skipped++; return true })
	failureCount.Range(func(_, _ any) bool { failed++; return true })

	logger.InfoContext(ctx, fmt.Sprintf(
		"startup physical backup slot cleanup complete: %d dropped, %d skipped (unreachable), %d failed",
		dropped, skipped, failed,
	))

	if failed > 0 {
		return errors.New("one or more databases failed startup slot cleanup; see logs")
	}

	return nil
}
