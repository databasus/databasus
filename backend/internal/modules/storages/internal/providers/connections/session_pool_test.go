package connections

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

const conditionWaitLimit = 5 * time.Second

var errSessionStillOpen = errors.New("the session is still open")

func TestFingerprint(t *testing.T) {
	t.Parallel()

	t.Run("returns the same value for the same settings and another value for changed ones", func(t *testing.T) {
		t.Parallel()

		baseline := Fingerprint("host", 22, "user", "encrypted-password")

		assert.Equal(t, baseline, Fingerprint("host", 22, "user", "encrypted-password"))
		assert.NotEqual(t, baseline, Fingerprint("other-host", 22, "user", "encrypted-password"))
		assert.NotEqual(t, baseline, Fingerprint("host", 2222, "user", "encrypted-password"))
		assert.NotEqual(t, baseline, Fingerprint("host", 22, "user", "changed-password"))
	})
}

func TestCheckout(t *testing.T) {
	t.Parallel()

	t.Run("reuses one session when checkouts follow each other", func(t *testing.T) {
		t.Parallel()

		pool := newTestPool(GetDefaultPoolSettings())
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		firstLease := checkoutSession(t, pool, server, key)
		firstSession := firstLease.GetSession()
		firstLease.Release(nil)

		secondLease := checkoutSession(t, pool, server, key)
		defer secondLease.Release(nil)

		assert.Same(t, firstSession, secondLease.GetSession())
		assert.EqualValues(t, 1, server.dialCount.Load())
	})

	t.Run("hands out different sessions when two leases are held at once", func(t *testing.T) {
		t.Parallel()

		pool := newTestPool(GetDefaultPoolSettings())
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		firstLease := checkoutSession(t, pool, server, key)
		defer firstLease.Release(nil)

		secondLease := checkoutSession(t, pool, server, key)
		defer secondLease.Release(nil)

		assert.NotSame(t, firstLease.GetSession(), secondLease.GetSession())
		assert.EqualValues(t, 2, server.dialCount.Load())
	})

	t.Run("dials a new session when the idle one fails its probe", func(t *testing.T) {
		t.Parallel()

		settings := GetDefaultPoolSettings()
		settings.ProbeAfter = time.Millisecond
		pool := newTestPool(settings)
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		staleLease := checkoutSession(t, pool, server, key)
		staleSession := staleLease.GetSession()
		staleLease.Release(nil)
		releasedAt := time.Now()

		staleSession.isBroken.Store(true)
		testwait.Until(t, "the probe interval to pass", func(context.Context) error {
			if time.Since(releasedAt) <= settings.ProbeAfter {
				return errSessionStillOpen
			}

			return nil
		})

		freshLease := checkoutSession(t, pool, server, key)
		defer freshLease.Release(nil)

		assert.EqualValues(t, 1, staleSession.probeCount.Load())
		assert.True(t, staleSession.isClosed())
		assert.NotSame(t, staleSession, freshLease.GetSession())
	})

	t.Run("closes the session and lets a blocked call return when the context ends", func(t *testing.T) {
		t.Parallel()

		pool := newTestPool(GetDefaultPoolSettings())
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		lease, err := pool.Checkout(ctx, key, server.dial)
		require.NoError(t, err)

		blockedCallReturned := make(chan struct{})
		go func() {
			<-lease.GetSession().closed
			close(blockedCallReturned)
		}()

		cancel()

		select {
		case <-blockedCallReturned:
		case <-time.After(conditionWaitLimit):
			t.Fatal("blocked call did not return after the lease context was cancelled")
		}

		lease.Release(errors.New("call failed after its session was closed"))

		nextLease := checkoutSession(t, pool, server, key)
		defer nextLease.Release(nil)

		assert.NotSame(t, lease.GetSession(), nextLease.GetSession())
	})

	t.Run("closes the old sessions and a stale lease on release when the fingerprint changes", func(t *testing.T) {
		t.Parallel()

		pool := newTestPool(GetDefaultPoolSettings())
		server := &fakeServer{}
		storageID := uuid.New()
		oldKey := newTestKey(storageID, "old-password")
		newKey := newTestKey(storageID, "new-password")

		idleOldLease := checkoutSession(t, pool, server, oldKey)
		heldOldLease := checkoutSession(t, pool, server, oldKey)
		idleOldLease.Release(nil)

		newLease := checkoutSession(t, pool, server, newKey)
		defer newLease.Release(nil)

		assert.True(t, idleOldLease.GetSession().isClosed())
		assert.NotSame(t, idleOldLease.GetSession(), newLease.GetSession())

		heldOldLease.Release(nil)

		assert.True(t, heldOldLease.GetSession().isClosed())
	})
}

func TestRelease(t *testing.T) {
	t.Parallel()

	t.Run("closes the session and the next checkout dials when the call failed", func(t *testing.T) {
		t.Parallel()

		pool := newTestPool(GetDefaultPoolSettings())
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		failedLease := checkoutSession(t, pool, server, key)
		failedSession := failedLease.GetSession()
		failedLease.Release(errors.New("write failed"))

		nextLease := checkoutSession(t, pool, server, key)
		defer nextLease.Release(nil)

		assert.True(t, failedSession.isClosed())
		assert.NotSame(t, failedSession, nextLease.GetSession())
		assert.EqualValues(t, 2, server.dialCount.Load())
	})

	t.Run("closes the fifth session when four already wait idle", func(t *testing.T) {
		t.Parallel()

		pool := newTestPool(GetDefaultPoolSettings())
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		leases := make([]*Lease[*fakeSession], 0, 5)
		for range 5 {
			leases = append(leases, checkoutSession(t, pool, server, key))
		}

		for _, lease := range leases {
			lease.Release(nil)
		}

		for _, lease := range leases[:4] {
			assert.False(t, lease.GetSession().isClosed())
		}

		assert.True(t, leases[4].GetSession().isClosed())
	})

	t.Run("closes the session when it stays idle past the idle timeout", func(t *testing.T) {
		t.Parallel()

		settings := GetDefaultPoolSettings()
		settings.IdleTimeout = 50 * time.Millisecond
		pool := newTestPool(settings)
		server := &fakeServer{}
		key := newTestKey(uuid.New(), "settings")

		lease := checkoutSession(t, pool, server, key)
		lease.Release(nil)

		testwait.Until(t, "the idle session to close", func(context.Context) error {
			if !lease.GetSession().isClosed() {
				return errSessionStillOpen
			}

			return nil
		})

		nextLease := checkoutSession(t, pool, server, key)
		defer nextLease.Release(nil)

		assert.NotSame(t, lease.GetSession(), nextLease.GetSession())
	})
}

type fakeSession struct {
	dialNumber int
	closed     chan struct{}
	closeOnce  sync.Once
	isBroken   atomic.Bool
	probeCount atomic.Int32
}

func (s *fakeSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

type fakeServer struct {
	dialCount atomic.Int32
}

func (f *fakeServer) dial(_ context.Context) (*fakeSession, error) {
	return &fakeSession{dialNumber: int(f.dialCount.Add(1)), closed: make(chan struct{})}, nil
}

func newTestPool(settings PoolSettings) *SessionPool[*fakeSession] {
	return NewSessionPool(SessionOperations[*fakeSession]{
		Close: func(session *fakeSession) {
			session.closeOnce.Do(func() { close(session.closed) })
		},
		Probe: func(_ context.Context, session *fakeSession) error {
			session.probeCount.Add(1)

			if session.isBroken.Load() {
				return errors.New("session is broken")
			}

			return nil
		},
	}, settings)
}

func newTestKey(storageID uuid.UUID, fingerprintParts ...any) Key {
	return Key{StorageID: storageID, Fingerprint: Fingerprint(fingerprintParts...)}
}

func checkoutSession(
	t *testing.T,
	pool *SessionPool[*fakeSession],
	server *fakeServer,
	key Key,
) *Lease[*fakeSession] {
	t.Helper()

	lease, err := pool.Checkout(t.Context(), key, server.dial)
	require.NoError(t, err)

	return lease
}
