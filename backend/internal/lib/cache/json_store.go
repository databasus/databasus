package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type JSONStore[T any] struct {
	store           Store
	namespace       string
	defaultLifetime time.Duration
}

type ExpiringValue[T any] struct {
	Key      string
	Value    T
	Lifetime time.Duration
}

func NewJSONStore[T any](store Store, namespace string) *JSONStore[T] {
	return NewJSONStoreWithLifetime[T](store, namespace, DefaultLifetime)
}

func NewJSONStoreWithLifetime[T any](
	store Store,
	namespace string,
	defaultLifetime time.Duration,
) *JSONStore[T] {
	return &JSONStore[T]{store, namespace, defaultLifetime}
}

func (s *JSONStore[T]) Get(ctx context.Context, key string) (*T, error) {
	payload, isFound, err := s.store.Get(ctx, s.key(key))
	if err != nil {
		return nil, err
	}
	if !isFound {
		return nil, nil
	}

	return decode[T](payload)
}

func (s *JSONStore[T]) Set(ctx context.Context, key string, item T) error {
	return s.SetWithLifetime(ctx, ExpiringValue[T]{
		Key:      key,
		Value:    item,
		Lifetime: s.defaultLifetime,
	})
}

func (s *JSONStore[T]) SetWithLifetime(
	ctx context.Context,
	expiringValue ExpiringValue[T],
) error {
	payload, err := json.Marshal(expiringValue.Value)
	if err != nil {
		return fmt.Errorf("encode cached value: %w", err)
	}

	return s.store.Set(ctx, Entry{
		Key:      s.key(expiringValue.Key),
		Payload:  payload,
		Lifetime: expiringValue.Lifetime,
	})
}

func (s *JSONStore[T]) CreateIfAbsent(
	ctx context.Context,
	expiringValue ExpiringValue[T],
) (bool, error) {
	payload, err := json.Marshal(expiringValue.Value)
	if err != nil {
		return false, fmt.Errorf("encode cached value: %w", err)
	}

	return s.store.CreateIfAbsent(ctx, Entry{
		Key:      s.key(expiringValue.Key),
		Payload:  payload,
		Lifetime: expiringValue.Lifetime,
	})
}

func (s *JSONStore[T]) ReadAndDelete(ctx context.Context, key string) (*T, error) {
	payload, isFound, err := s.store.ReadAndDelete(ctx, s.key(key))
	if err != nil {
		return nil, err
	}
	if !isFound {
		return nil, nil
	}

	return decode[T](payload)
}

func (s *JSONStore[T]) Delete(ctx context.Context, key string) error {
	return s.store.Delete(ctx, s.key(key))
}

func (s *JSONStore[T]) key(key string) string {
	return strconv.Itoa(len(s.namespace)) + ":" + s.namespace + ":" + key
}

func decode[T any](payload []byte) (*T, error) {
	var item T
	if err := json.Unmarshal(payload, &item); err != nil {
		return nil, fmt.Errorf("decode cached value: %w", err)
	}

	return &item, nil
}
