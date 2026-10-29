package upgrade

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"databasus-backend/internal/agent/api"
)

const backgroundCheckInterval = 10 * time.Second

type BackgroundUpgrader struct {
	apiClient      *api.Client
	currentVersion string
	isDev          bool
	cancel         context.CancelFunc
	hasRun         atomic.Bool
	isUpgraded     atomic.Bool
	log            *slog.Logger
	done           chan struct{}
}

type Deps struct {
	APIClient      *api.Client
	CurrentVersion string
	IsDev          bool
	Cancel         context.CancelFunc
	Logger         *slog.Logger
}

func NewBackgroundUpgrader(deps Deps) *BackgroundUpgrader {
	return &BackgroundUpgrader{
		deps.APIClient,
		deps.CurrentVersion,
		deps.IsDev,
		deps.Cancel,
		atomic.Bool{},
		atomic.Bool{},
		deps.Logger,
		make(chan struct{}),
	}
}

func (u *BackgroundUpgrader) Run(ctx context.Context) {
	if u.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", u))
	}

	defer close(u.done)

	ticker := time.NewTicker(backgroundCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if u.checkAndUpgrade(ctx) {
				return
			}
		}
	}
}

func (u *BackgroundUpgrader) IsUpgraded() bool {
	return u.isUpgraded.Load()
}

func (u *BackgroundUpgrader) WaitForCompletion(timeout time.Duration) {
	select {
	case <-u.done:
	case <-time.After(timeout):
	}
}

func (u *BackgroundUpgrader) checkAndUpgrade(ctx context.Context) bool {
	isUpgraded, err := CheckAndUpdate(ctx, u.apiClient, u.currentVersion, u.isDev, u.log)
	if err != nil {
		u.log.WarnContext(ctx, "background update check failed", slog.Any("error", err))

		return false
	}

	if !isUpgraded {
		return false
	}

	u.log.InfoContext(ctx, "background upgrade complete, restarting...")
	u.isUpgraded.Store(true)
	u.cancel()

	return true
}
