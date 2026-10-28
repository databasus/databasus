package walstreaming

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases"
)

const slotCleanupDeadline = 20 * time.Second

type PhysicalSlotCleanupListener struct {
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
	logger := l.logger.With(slog.Any("database_id", connectionSettings.DatabaseID))
	slotName := connectionSettings.BackupSlotName
	walSlotName := connectionSettings.WalStreamSlotName

	cleanupCtx, cancel := context.WithTimeout(ctx, slotCleanupDeadline)
	defer cancel()

	tunnel, err := pgphysical.NewSource(connectionSettings.Engine).OpenTunnel(cleanupCtx, logger)
	if err != nil {

		logger.WarnContext(
			ctx,
			"physical slot cleanup: bastion unreachable, leaving slots orphaned",
			slog.Any("slot_name", slotName),
			slog.Any("wal_slot_name", walSlotName),
			slog.Any("error", err),
		)

		return
	}

	defer tunnel.Close()

	source := tunnel.GetSource()

	conn, err := source.OpenInspection(cleanupCtx)
	if err != nil {
		logger.WarnContext(
			ctx,
			"physical slot cleanup: source pg unreachable, leaving slots orphaned",
			slog.Any("slot_name", slotName),
			slog.Any("wal_slot_name", walSlotName),
			slog.Any("error", err),
		)

		return
	}
	defer func() { _ = conn.Close(context.WithoutCancel(ctx)) }()

	err = conn.DropBackupSlotIfExists(cleanupCtx, slotName)
	if err != nil {
		logger.WarnContext(
			ctx,
			"physical slot cleanup: per-backup slot drop failed", slog.Any("slot_name",
				slotName), slog.Any("error",
				err),
		)
	}

	err = source.DropSlotForRemoval(cleanupCtx, logger, walSlotName)
	if err != nil {
		logger.WarnContext(
			ctx,
			"physical slot cleanup: wal streamer slot drop failed",
			slog.Any("slot_name", walSlotName),
			slog.Any("error", err),
		)
	}
}
