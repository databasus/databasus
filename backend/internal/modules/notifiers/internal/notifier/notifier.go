// Package notifier creates, lists, reads, changes, tests, transfers and deletes notifiers, keeps
// their parent rows and records the outcome of each send.
package notifier

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers"
)

// Notifier is a notification channel as other modules see it, without its settings.
type Notifier struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Type        notification.NotifierType
	// LastSendError is nil when the latest send was delivered or nothing was sent yet.
	LastSendError *string
}

// WithSettings is a notifier with the settings of its provider. Secrets in them stay encrypted.
type WithSettings struct {
	Notifier
	Settings providers.Settings
}

// Creation is a new notifier; the type of its settings becomes the notifier's type.
type Creation struct {
	WorkspaceID uuid.UUID
	Name        string
	Settings    providers.Settings
}

// Change replaces the name and the settings of a notifier. A secret the settings leave out keeps its
// stored value while the resource address stays the stored one.
type Change struct {
	NotifierID uuid.UUID
	Name       string
	Settings   providers.Settings
}
