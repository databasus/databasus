package providers

import "fmt"

// Registry is the only place that maps a database type to its provider.
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

func (r *Registry) GetProvider(databaseType Type) (Provider, error) {
	provider, isFound := r.providers[databaseType]
	if !isFound {
		return nil, fmt.Errorf("%w: %s", ErrUnknownDatabaseType, databaseType)
	}

	return provider, nil
}
