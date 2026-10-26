package notifiers_bridge

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers"
)

// Sender notifies through the installed module, which it reads per call, because each test
// application installs its own.
type Sender struct{}

func GetSender() *Sender {
	return &Sender{}
}

func (s *Sender) Notify(ctx context.Context, notifierIDs []uuid.UUID, notification notifiers.Notification) {
	GetModule().Notify(ctx, notifierIDs, notification)
}
