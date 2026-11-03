package cache

import (
	"context"
	"sync"
	"time"

	"databasus-backend/internal/lib/clock"
)

type Deps struct {
	// PayloadBudgetBytes caps the keys and payloads held at once, so a flood of entries refuses new
	// writes instead of exhausting the host's memory.
	PayloadBudgetBytes int64
	Clock              clock.Source
}

type memoryEntry struct {
	payload   []byte
	expiresAt time.Time
	sizeBytes int64
}

type MemoryStore struct {
	mutex              sync.Mutex
	entries            map[string]memoryEntry
	payloadBytes       int64
	payloadBudgetBytes int64
	clock              clock.Source
}

func NewMemoryStore(deps Deps) *MemoryStore {
	if deps.PayloadBudgetBytes <= 0 {
		panic("cache payload budget must be positive")
	}

	return &MemoryStore{
		mutex:              sync.Mutex{},
		entries:            make(map[string]memoryEntry),
		payloadBytes:       0,
		payloadBudgetBytes: deps.PayloadBudgetBytes,
		clock:              deps.Clock,
	}
}

func (s *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry, isFound := s.getLiveEntry(key, s.clock.Now())
	if !isFound {
		return nil, false, nil
	}

	return append([]byte(nil), entry.payload...), true, nil
}

func (s *MemoryStore) Set(ctx context.Context, entry Entry) error {
	if err := validateWrite(ctx, entry); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.store(entry, s.clock.Now())
}

func (s *MemoryStore) CreateIfAbsent(ctx context.Context, entry Entry) (bool, error) {
	if err := validateWrite(ctx, entry); err != nil {
		return false, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := s.clock.Now()
	if _, isFound := s.getLiveEntry(entry.Key, now); isFound {
		return false, nil
	}

	if err := s.store(entry, now); err != nil {
		return false, err
	}

	return true, nil
}

func (s *MemoryStore) ReadAndDelete(ctx context.Context, key string) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	entry, isFound := s.getLiveEntry(key, s.clock.Now())
	if !isFound {
		return nil, false, nil
	}

	s.remove(key, entry)

	return append([]byte(nil), entry.payload...), true, nil
}

func (s *MemoryStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if entry, isFound := s.entries[key]; isFound {
		s.remove(key, entry)
	}

	return nil
}

func validateWrite(ctx context.Context, entry Entry) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if entry.Lifetime <= 0 {
		return ErrInvalidLifetime
	}

	return nil
}

func (s *MemoryStore) getLiveEntry(key string, now time.Time) (memoryEntry, bool) {
	entry, isFound := s.entries[key]
	if !isFound {
		return memoryEntry{}, false
	}

	if !now.Before(entry.expiresAt) {
		s.remove(key, entry)

		return memoryEntry{}, false
	}

	return entry, true
}

// reclaimExpired scans every entry, so it runs only when a write would exceed the budget.
func (s *MemoryStore) reclaimExpired(now time.Time) {
	for key, entry := range s.entries {
		if !now.Before(entry.expiresAt) {
			s.remove(key, entry)
		}
	}
}

func (s *MemoryStore) store(entry Entry, now time.Time) error {
	entrySizeBytes := int64(len(entry.Key) + len(entry.Payload))
	previousSizeBytes := s.getStoredEntrySizeBytes(entry.Key)

	if s.payloadBytes-previousSizeBytes+entrySizeBytes > s.payloadBudgetBytes {
		s.reclaimExpired(now)
		previousSizeBytes = s.getStoredEntrySizeBytes(entry.Key)

		if s.payloadBytes-previousSizeBytes+entrySizeBytes > s.payloadBudgetBytes {
			return ErrPayloadBudgetReached
		}
	}

	s.entries[entry.Key] = memoryEntry{
		payload:   append([]byte(nil), entry.Payload...),
		expiresAt: now.Add(entry.Lifetime),
		sizeBytes: entrySizeBytes,
	}
	s.payloadBytes = s.payloadBytes - previousSizeBytes + entrySizeBytes

	return nil
}

func (s *MemoryStore) getStoredEntrySizeBytes(key string) int64 {
	entry, isFound := s.entries[key]
	if !isFound {
		return 0
	}

	return entry.sizeBytes
}

func (s *MemoryStore) remove(key string, entry memoryEntry) {
	delete(s.entries, key)
	s.payloadBytes -= entry.sizeBytes
}
