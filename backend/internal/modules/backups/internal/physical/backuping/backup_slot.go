package backuping

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
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
	conn, err := pgphysical.NewSource(sourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return fmt.Errorf("open conn for backup slot: %w", err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	slotName := sourceDB.BackupSlotName

	if err := conn.DropBackupSlotIfExists(ctx, slotName); err != nil {
		return fmt.Errorf("pre-create drop of backup slot %q: %w", slotName, err)
	}

	if err := conn.CreateSlot(ctx, slotName); err != nil {
		if dropErr := dropBackupSlotAfterFailedCreate(ctx, sourceDB, logger); dropErr != nil {
			logger.WarnContext(
				ctx,
				"slot drop after a failed create failed; will be recovered by next backup or startup cleanup",
				slog.Any("slot_name", slotName),
				slog.Any("error", dropErr),
			)
		}

		return fmt.Errorf("create backup slot %q: %w", slotName, err)
	}

	logger.InfoContext(ctx, "per-backup replication slot created", slog.Any("slot_name", slotName))

	defer func() {
		// Slot cleanup must survive cancellation of the backup.
		if dropErr := conn.DropBackupSlotIfExists(context.WithoutCancel(ctx), slotName); dropErr == nil {
			logger.InfoContext(ctx, "per-backup replication slot dropped", slog.Any("slot_name", slotName))

			return
		}

		// The held connection is the first casualty of a network break, and a bastion in front of the
		// source makes those routine. Leaving the slot pins WAL on the source until the next backup
		// runs, which on a weekly full is a week of retained segments.
		if retryErr := dropBackupSlotOverFreshConn(ctx, sourceDB, slotName); retryErr != nil {
			logger.WarnContext(
				ctx,
				"post-backup slot drop failed; will be recovered by next backup or startup cleanup",
				slog.Any("slot_name", slotName),
				slog.Any("error", retryErr),
			)
		}
	}()

	return fn()
}

// dropBackupSlotAfterFailedCreate drops a slot the server may have created after the client gave
// up: a cancellation breaks the held connection while the server's session still holds the slot,
// so the slot is dropped over a fresh connection once that session lets go of it.
func dropBackupSlotAfterFailedCreate(
	ctx context.Context,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	logger *slog.Logger,
) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotDropRetryTimeout)
	defer cancel()

	return pgphysical.NewSource(sourceDB.Engine).DropSlotForRemoval(ctx, logger, sourceDB.BackupSlotName)
}

func dropBackupSlotOverFreshConn(
	ctx context.Context,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
	slotName string,
) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slotDropRetryTimeout)
	defer cancel()

	conn, err := pgphysical.NewSource(sourceDB.Engine).OpenInspection(ctx)
	if err != nil {
		return fmt.Errorf("reopen conn to drop backup slot %q: %w", slotName, err)
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	return conn.DropBackupSlotIfExists(ctx, slotName)
}
