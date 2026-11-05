package rclone

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rclone/rclone/fs"
)

const (
	rcloneRemoteIdleTimeout     = 5 * time.Minute
	rcloneRemoteShutdownTimeout = 30 * time.Second
)

// Building a remote runs fs.NewFs under the configuration lock of the provider, which would serialize every
// write of every rclone storage. A remote is kept per storage and rebuilt only when the fingerprint
// of its stored configuration changes, so a write never reaches the remote through old settings.
type remoteCache struct {
	idleTimeout time.Duration

	mutex   sync.Mutex
	remotes map[uuid.UUID]*cachedRemote
}

// A remote is shut down only once nobody uses it, so a full backup streaming for hours through a
// remote whose configuration changed meanwhile is never cut off by the change.
type cachedRemote struct {
	// shutdownCtx carries the values of the call that built the remote to its shutdown, which may run
	// from an idle timer long after that call ended.
	shutdownCtx context.Context
	storageID   uuid.UUID
	remoteFs    fs.Fs
	fingerprint string
	activeUses  int
	isRetired   bool
	idleTimer   *time.Timer
}

func newRemoteCache(idleTimeout time.Duration) *remoteCache {
	return &remoteCache{
		idleTimeout: idleTimeout,
		remotes:     make(map[uuid.UUID]*cachedRemote),
	}
}

func (c *remoteCache) acquire(
	ctx context.Context,
	storageID uuid.UUID,
	fingerprint string,
	build func(ctx context.Context) (fs.Fs, error),
) (fs.Fs, func(), error) {
	if cached, isHit := c.takeCurrent(storageID, fingerprint); isHit {
		return cached.remoteFs, c.releaseFunc(cached), nil
	}

	builtFs, err := build(ctx)
	if err != nil {
		return nil, nil, err
	}

	stored := c.storeBuilt(context.WithoutCancel(ctx), storageID, fingerprint, builtFs)
	if stored.remoteFs != builtFs {
		shutdownRemote(ctx, builtFs)
	}

	return stored.remoteFs, c.releaseFunc(stored), nil
}

func (c *remoteCache) takeCurrent(storageID uuid.UUID, fingerprint string) (*cachedRemote, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cached := c.remotes[storageID]
	if cached == nil || cached.fingerprint != fingerprint {
		return nil, false
	}

	c.markUsed(cached)

	return cached, true
}

// Two writes can miss at once and both build. The first one stored wins, and the loser's remote is
// shut down by the caller, so no more than one remote per configuration outlives the race.
func (c *remoteCache) storeBuilt(
	shutdownCtx context.Context,
	storageID uuid.UUID,
	fingerprint string,
	builtFs fs.Fs,
) *cachedRemote {
	c.mutex.Lock()

	if cached := c.remotes[storageID]; cached != nil && cached.fingerprint == fingerprint {
		c.markUsed(cached)
		c.mutex.Unlock()

		return cached
	}

	replaced := c.remotes[storageID]
	stored := &cachedRemote{
		shutdownCtx: shutdownCtx,
		storageID:   storageID,
		remoteFs:    builtFs,
		fingerprint: fingerprint,
		activeUses:  1,
		isRetired:   false,
		idleTimer:   nil,
	}
	c.remotes[storageID] = stored

	isReplacedIdle := replaced != nil && c.retire(replaced)
	c.mutex.Unlock()

	if isReplacedIdle {
		shutdownRemote(replaced.shutdownCtx, replaced.remoteFs)
	}

	return stored
}

func (c *remoteCache) markUsed(cached *cachedRemote) {
	cached.activeUses++

	if cached.idleTimer != nil {
		cached.idleTimer.Stop()
		cached.idleTimer = nil
	}
}

// Reports whether the retired remote is idle, so the caller shuts it down outside the mutex.
func (c *remoteCache) retire(cached *cachedRemote) bool {
	cached.isRetired = true

	if cached.idleTimer != nil {
		cached.idleTimer.Stop()
		cached.idleTimer = nil
	}

	return cached.activeUses == 0
}

func (c *remoteCache) releaseFunc(cached *cachedRemote) func() {
	return sync.OnceFunc(func() {
		c.release(cached)
	})
}

func (c *remoteCache) release(cached *cachedRemote) {
	c.mutex.Lock()

	cached.activeUses--
	if cached.activeUses > 0 {
		c.mutex.Unlock()
		return
	}

	if cached.isRetired {
		c.mutex.Unlock()
		shutdownRemote(cached.shutdownCtx, cached.remoteFs)

		return
	}

	cached.idleTimer = time.AfterFunc(c.idleTimeout, func() {
		c.expire(cached)
	})
	c.mutex.Unlock()
}

// A use that started after the timer fired already holds the remote, so the expiry yields to it.
func (c *remoteCache) expire(cached *cachedRemote) {
	c.mutex.Lock()

	if cached.activeUses > 0 || cached.isRetired || c.remotes[cached.storageID] != cached {
		c.mutex.Unlock()
		return
	}

	delete(c.remotes, cached.storageID)
	c.retire(cached)
	c.mutex.Unlock()

	shutdownRemote(cached.shutdownCtx, cached.remoteFs)
}

// shutdownRemote runs after the caller's work ended, so it outlives a cancellation of ctx under its
// own deadline.
func shutdownRemote(ctx context.Context, remoteFs fs.Fs) {
	shutdowner, isShutdowner := remoteFs.(fs.Shutdowner)
	if !isShutdowner {
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), rcloneRemoteShutdownTimeout)
	defer cancel()

	_ = shutdowner.Shutdown(ctx)
}
