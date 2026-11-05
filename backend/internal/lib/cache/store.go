// Package cache keeps short-lived values in the memory of the one backend process, such as
// single-use tokens and rate-limit counters.
package cache

import (
	"context"
	"time"
)

const (
	DefaultLifetime           = 10 * time.Minute
	DefaultPayloadBudgetBytes = 256 * 1024 * 1024
)

type Entry struct {
	Key      string
	Payload  []byte
	Lifetime time.Duration
}

type Store interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, entry Entry) error
	CreateIfAbsent(ctx context.Context, entry Entry) (bool, error)
	ReadAndDelete(ctx context.Context, key string) ([]byte, bool, error)
	Delete(ctx context.Context, key string) error
}
