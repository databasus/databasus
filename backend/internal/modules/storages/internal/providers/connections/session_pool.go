// Package connections keeps the sessions of the providers that dial a server, so consecutive writes to
// one storage reuse a connection instead of paying a handshake each.
package connections

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Four matches the WAL uploader's concurrent writes per database, so a busy database keeps every
// session it needs between segments.
const (
	defaultIdleCap      = 4
	defaultProbeAfter   = 30 * time.Second
	defaultProbeTimeout = 5 * time.Second
	defaultIdleTimeout  = 5 * time.Minute
)

type PoolSettings struct {
	IdleCap int

	// A server may drop a quiet session without telling us, so a session idle longer than this is
	// probed before a write trusts it.
	ProbeAfter   time.Duration
	ProbeTimeout time.Duration

	// Frees server-side connection slots for storages nobody writes to, deleted ones included.
	IdleTimeout time.Duration
}

func GetDefaultPoolSettings() PoolSettings {
	return PoolSettings{
		IdleCap:      defaultIdleCap,
		ProbeAfter:   defaultProbeAfter,
		ProbeTimeout: defaultProbeTimeout,
		IdleTimeout:  defaultIdleTimeout,
	}
}

type SessionOperations[T any] struct {
	Close func(session T)
	Probe func(ctx context.Context, session T) error
}

// Each checkout gets a session nobody else uses until it is released, so one slow transfer never
// makes another wait behind it on the same connection, and one dead connection fails one write.
type SessionPool[T any] struct {
	operations SessionOperations[T]
	settings   PoolSettings

	mutex               sync.Mutex
	currentFingerprints map[uuid.UUID]string
	idleSessions        map[Key][]*idleSession[T]
}

type idleSession[T any] struct {
	session   T
	idleSince time.Time
	idleTimer *time.Timer
}

func NewSessionPool[T any](operations SessionOperations[T], settings PoolSettings) *SessionPool[T] {
	return &SessionPool[T]{
		operations:          operations,
		settings:            settings,
		currentFingerprints: make(map[uuid.UUID]string),
		idleSessions:        make(map[Key][]*idleSession[T]),
	}
}

// The dial comes with each checkout because only the caller holds the decrypted settings the key was
// fingerprinted from.
func (p *SessionPool[T]) Checkout(
	ctx context.Context,
	key Key,
	dial func(ctx context.Context) (T, error),
) (*Lease[T], error) {
	p.closeSessions(p.retireOtherFingerprints(key))

	for {
		candidate, isFound := p.takeIdleSession(key)
		if !isFound {
			break
		}

		if p.isSessionAlive(ctx, candidate) {
			return p.lease(ctx, key, candidate.session), nil
		}

		p.operations.Close(candidate.session)
	}

	session, err := dial(ctx)
	if err != nil {
		return nil, err
	}

	return p.lease(ctx, key, session), nil
}

func (p *SessionPool[T]) retireOtherFingerprints(key Key) []T {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	previousFingerprint, isKnown := p.currentFingerprints[key.StorageID]
	p.currentFingerprints[key.StorageID] = key.Fingerprint

	if !isKnown || previousFingerprint == key.Fingerprint {
		return nil
	}

	previousKey := Key{StorageID: key.StorageID, Fingerprint: previousFingerprint}
	retiredSessions := make([]T, 0, len(p.idleSessions[previousKey]))

	for _, retired := range p.idleSessions[previousKey] {
		retired.idleTimer.Stop()
		retiredSessions = append(retiredSessions, retired.session)
	}

	delete(p.idleSessions, previousKey)

	return retiredSessions
}

func (p *SessionPool[T]) takeIdleSession(key Key) (*idleSession[T], bool) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	sessions := p.idleSessions[key]
	if len(sessions) == 0 {
		return nil, false
	}

	newest := sessions[len(sessions)-1]
	p.setIdleSessions(key, sessions[:len(sessions)-1])
	newest.idleTimer.Stop()

	return newest, true
}

func (p *SessionPool[T]) isSessionAlive(ctx context.Context, candidate *idleSession[T]) bool {
	if time.Since(candidate.idleSince) < p.settings.ProbeAfter {
		return true
	}

	probeCtx, cancel := context.WithTimeout(ctx, p.settings.ProbeTimeout)
	defer cancel()

	return p.operations.Probe(probeCtx, candidate.session) == nil
}

func (p *SessionPool[T]) lease(ctx context.Context, key Key, session T) *Lease[T] {
	lease := &Lease[T]{pool: p, key: key, session: session}

	// Neither the SSH nor the SMB library honors a context once a request is on the wire. Closing the
	// session is what makes a call blocked on a silent server return.
	lease.stopCloseOnCancel = context.AfterFunc(ctx, func() {
		p.operations.Close(session)
	})

	return lease
}

func (p *SessionPool[T]) giveBack(key Key, session T) {
	p.mutex.Lock()

	isStale := p.currentFingerprints[key.StorageID] != key.Fingerprint
	isFull := len(p.idleSessions[key]) >= p.settings.IdleCap

	if isStale || isFull {
		p.mutex.Unlock()
		p.operations.Close(session)

		return
	}

	returned := &idleSession[T]{session: session, idleSince: time.Now()}
	returned.idleTimer = time.AfterFunc(p.settings.IdleTimeout, func() {
		p.expireIdleSession(key, returned)
	})

	p.idleSessions[key] = append(p.idleSessions[key], returned)
	p.mutex.Unlock()
}

// A checkout that took the session first removed it under the same mutex, so a timer that fires
// late finds nothing and leaves the session to its new holder.
func (p *SessionPool[T]) expireIdleSession(key Key, expired *idleSession[T]) {
	p.mutex.Lock()

	sessions := p.idleSessions[key]
	expiredIndex := -1

	for index, candidate := range sessions {
		if candidate == expired {
			expiredIndex = index
			break
		}
	}

	if expiredIndex < 0 {
		p.mutex.Unlock()
		return
	}

	p.setIdleSessions(key, append(sessions[:expiredIndex:expiredIndex], sessions[expiredIndex+1:]...))
	p.mutex.Unlock()

	p.operations.Close(expired.session)
}

func (p *SessionPool[T]) setIdleSessions(key Key, sessions []*idleSession[T]) {
	if len(sessions) == 0 {
		delete(p.idleSessions, key)
		return
	}

	p.idleSessions[key] = sessions
}

func (p *SessionPool[T]) closeSessions(sessions []T) {
	for _, session := range sessions {
		p.operations.Close(session)
	}
}

type Lease[T any] struct {
	pool              *SessionPool[T]
	key               Key
	session           T
	stopCloseOnCancel func() bool

	releaseOnce sync.Once
}

func (l *Lease[T]) GetSession() T {
	return l.session
}

// A failed call may leave the protocol mid-exchange, so a session released with an error is closed
// rather than handed to the next write.
func (l *Lease[T]) Release(err error) {
	l.releaseOnce.Do(func() {
		isClosedByCancel := !l.stopCloseOnCancel()
		if isClosedByCancel {
			return
		}

		if err != nil {
			l.pool.operations.Close(l.session)
			return
		}

		l.pool.giveBack(l.key, l.session)
	})
}
