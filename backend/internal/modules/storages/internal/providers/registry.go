package providers

import "fmt"

// Registry is the only place that maps a storage type to its provider.
type Registry struct {
	providers map[Type]Provider
}

func NewRegistry(providers ...Provider) *Registry {
	providersByType := make(map[Type]Provider, len(providers))
	for _, provider := range providers {
		providersByType[provider.GetType()] = provider
	}

	return &Registry{providers: providersByType}
}

func (r *Registry) GetProvider(storageType Type) (Provider, error) {
	provider, isFound := r.providers[storageType]
	if !isFound {
		return nil, fmt.Errorf("%w: %s", ErrUnknownStorageType, storageType)
	}

	return provider, nil
}
