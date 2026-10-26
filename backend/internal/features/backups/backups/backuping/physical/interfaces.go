package backuping_physical

import (
	"context"

	"github.com/google/uuid"

	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/modules/notifiers"
)

type NotificationSender interface {
	Notify(ctx context.Context, notifierIDs []uuid.UUID, notification notifiers.Notification)
}

type FullBackupExecutor interface {
	Execute(
		ctx context.Context,
		spec postgresql_executor.FullBackupSpec,
	) (postgresql_executor.PhysicalBackupResult, error)
}

type IncrementalBackupExecutor interface {
	Execute(
		ctx context.Context,
		spec postgresql_executor.IncrementalBackupSpec,
	) (postgresql_executor.PhysicalBackupResult, error)
}
