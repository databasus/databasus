package usecases_physical_postgresql

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
)

// slotCleanupDeadline caps how long we wait on the source PG when a database
// is being removed. If the source is unreachable we log + skip so the
// metadata-level delete still goes through; RunStartupCleanup will not
// recover the WAL streamer slot (which lives until that source disappears),
// but the per-backup slot is recoverable on the next run that touches that
// source. The budget covers terminating a still-attached pg_receivewal and
// waiting for the WAL streamer slot to detach before dropping it, so it is
// larger than a plain query timeout.
const slotCleanupDeadline = 20 * time.Second

// PhysicalSlotCleanupListener drops the per-backup and WAL streamer slots of a
// deleted physical PostgreSQL database once its deletion commits.
//
// Without this hook, slots would be orphaned on the source PG every time a
// database is removed — pinning WAL forever. RunStartupCleanup catches
// per-backup orphans only for databases that still exist; once the metadata
// row is gone there is no record of which source to scan.
type PhysicalSlotCleanupListener struct {
	// getDatabases is read per call: each test application installs its own module.
	getDatabases func() databases.Module
	logger       *slog.Logger
}

func NewPhysicalSlotCleanupListener(
	getDatabases func() databases.Module,
	logger *slog.Logger,
) *PhysicalSlotCleanupListener {
	return &PhysicalSlotCleanupListener{
		getDatabases: getDatabases,
		logger:       logger,
	}
}

// OnDatabaseDeleting reads the connection settings while the rows still exist and drops the slots
// after the commit, so a refused deletion keeps them.
func (l *PhysicalSlotCleanupListener) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	deletedDatabases, err := l.getDatabases().GetDatabases(ctx, []uuid.UUID{event.DatabaseID})
	if err != nil {
		return err
	}

	if len(deletedDatabases) == 0 || deletedDatabases[0].Type != databases.TypePostgresqlPhysical {
		return nil
	}

	connectionSettings, err := l.getDatabases().GetPostgresqlPhysicalConnectionSettings(ctx, event.DatabaseID)
	if err != nil {
		return err
	}

	return database.AfterCommit(ctx, func(ctx context.Context) {
		l.dropSlots(ctx, connectionSettings)
	})
}

func (l *PhysicalSlotCleanupListener) dropSlots(
	ctx context.Context,
	connectionSettings databases.PostgresqlPhysicalConnectionSettings,
) {
	logger := l.logger.With("database_id", connectionSettings.DatabaseID)
	slotName := connectionSettings.BackupSlotName
	walSlotName := connectionSettings.Engine.ReplicationSlotName

	cleanupCtx, cancel := context.WithTimeout(ctx, slotCleanupDeadline)
	defer cancel()

	tunnel, err := pgphysical.NewSource(connectionSettings.Engine).OpenTunnel(cleanupCtx, logger)
	if err != nil {
		// Both names go into the message: the row is gone, and with it any record of which cluster
		// to go looking on, so this line is the operator's only handle on them.
		logger.WarnContext(ctx, "physical slot cleanup: bastion unreachable, leaving slots orphaned",
			"slot_name", slotName,
			"wal_slot_name", walSlotName,
			"error", err,
		)

		return
	}

	defer tunnel.Close()

	source := tunnel.GetSource()

	conn, err := source.OpenInspectionConn(cleanupCtx)
	if err != nil {
		logger.WarnContext(ctx, "physical slot cleanup: source PG unreachable, leaving slots orphaned",
			"slot_name", slotName,
			"wal_slot_name", walSlotName,
			"error", err,
		)

		return
	}
	defer func() { _ = conn.Close(context.Background()) }()

	err = dropBackupSlotIfExists(cleanupCtx, conn, slotName)
	if err != nil {
		logger.WarnContext(
			ctx,
			"physical slot cleanup: per-backup slot drop failed",
			"slot_name",
			slotName,
			"error",
			err,
		)
	}

	err = source.DropSlotForRemoval(cleanupCtx, logger)
	if err != nil {
		logger.WarnContext(ctx, "physical slot cleanup: WAL streamer slot drop failed",
			"slot_name", walSlotName,
			"error", err,
		)
	}
}
