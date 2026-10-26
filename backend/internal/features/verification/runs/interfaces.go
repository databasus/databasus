package verification_runs

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers"
)

type NotificationSender interface {
	Notify(ctx context.Context, notifierIDs []uuid.UUID, notification notifiers.Notification)
}
