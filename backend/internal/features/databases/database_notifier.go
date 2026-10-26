package databases

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers"
)

// DatabaseNotifier is a notifier a database uses, with the fields the /databases responses show.
// A request names a notifier by its ID alone.
type DatabaseNotifier struct {
	ID            uuid.UUID      `json:"id"`
	WorkspaceID   uuid.UUID      `json:"workspaceId"`
	Name          string         `json:"name"`
	NotifierType  notifiers.Type `json:"notifierType"`
	LastSendError *string        `json:"lastSendError"`
}

func newDatabaseNotifier(notifier notifiers.Notifier) DatabaseNotifier {
	return DatabaseNotifier{
		ID:            notifier.ID,
		WorkspaceID:   notifier.WorkspaceID,
		Name:          notifier.Name,
		NotifierType:  notifier.Type,
		LastSendError: notifier.LastSendError,
	}
}
