// Package providers declares what a notifier provider is and keeps the registry that maps a
// notifier type to its provider. Each provider lives in a subpackage with its settings, its table
// and its send.
package providers

import (
	"context"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/notifiers/internal/notification"
)

// Settings are the provider-specific settings of one notifier. Secrets in settings that a
// repository loaded stay encrypted; only Send decrypts them.
type Settings interface {
	GetType() notification.NotifierType
}

type Provider interface {
	GetType() notification.NotifierType
	// Validate answers an apierror problem naming the setting it refuses.
	Validate(settings Settings) error
	// MergeStoredSecrets fills the secrets the incoming settings left out from the stored ones. A secret
	// tied to the provider's resource address is filled only while that address stays the stored one,
	// so a user who may not see a secret cannot send it to a server of their choice. Left empty, the
	// secret then gets the refusal of Validate when the settings use it, and is removed when they do not.
	MergeStoredSecrets(merge SecretMerge) (Settings, error)
	Send(ctx context.Context, settings Settings, message notification.Notification) error
	GetRepository() SettingsRepository
}

// DeliveryTimeoutProvider is a provider whose deliveries need longer than the general limit, such as
// one that waits out a rate limit. A test send ignores it, because a person is waiting for the answer.
type DeliveryTimeoutProvider interface {
	GetDeliveryTimeout() time.Duration
}

// SecretMerge names both sides, so a swapped call cannot keep the stored settings over the edit.
type SecretMerge struct {
	Incoming Settings
	Stored   Settings
}

// SettingsRepository keeps the settings of one provider in its own table. Upsert encrypts the
// secrets that are not encrypted yet.
type SettingsRepository interface {
	Upsert(ctx context.Context, notifierID uuid.UUID, settings Settings) error
	// ListByNotifierIDs leaves out the IDs that have no row.
	ListByNotifierIDs(ctx context.Context, notifierIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
}

// Endpoints are the addresses of the services a provider calls when its settings name none.
type Endpoints struct {
	SlackPostMessageURL string
	TelegramBotAPIURL   string
}

// CastSettings returns settings as the settings type S of one provider.
func CastSettings[S Settings](settings Settings) (S, error) {
	typedSettings, isOfType := settings.(S)
	if !isOfType {
		return typedSettings, ErrSettingsOfAnotherType
	}

	return typedSettings, nil
}
