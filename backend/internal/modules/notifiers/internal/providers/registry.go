package providers

import (
	"fmt"

	"databasus-backend/internal/modules/notifiers/internal/notification"
)

// Registry is the only place that maps a notifier type to its provider.
type Registry struct {
	providers map[notification.NotifierType]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	providersByType := make(map[notification.NotifierType]Provider, len(providers))
	for _, provider := range providers {
		providersByType[provider.GetType()] = provider
	}

	return &Registry{providers: providersByType}
}

func (r *Registry) GetProvider(notifierType notification.NotifierType) (Provider, error) {
	provider, isFound := r.providers[notifierType]
	if !isFound {
		return nil, fmt.Errorf("%w: %s", ErrUnknownNotifierType, notifierType)
	}

	return provider, nil
}
