package cache

import "errors"

var (
	ErrInvalidLifetime      = errors.New("cache entry lifetime must be positive")
	ErrPayloadBudgetReached = errors.New("cache payload budget reached")
)
