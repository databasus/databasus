package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/lib/shutdown"
)

// The drain and the log flush together stay under 8 seconds, which leaves the embedded PostgreSQL
// server time for a clean shutdown within Docker's default stop timeout of 10 seconds.
const (
	ShutdownDrainTimeout    = 6 * time.Second
	shutdownLogFlushTimeout = 2 * time.Second
)

const frontendDirectory = "./ui/build"

// ServeOutcome tells the caller why Serve returned.
type ServeOutcome struct {
	// IsRebuildRequested is a stop for an accepted recovery rather than for the stop signal: the
	// caller closes the application, publishes the recovery and serves a new application.
	IsRebuildRequested bool
	// IsDrained is whether every request, task, loop and job pass ended within the shutdown limit.
	IsDrained bool
}

// Serve opens the port, starts the background loops, the background jobs and the HTTP server and
// blocks until ctx ends or an accepted recovery requests a rebuild. Then it stops accepting work and
// waits for requests, registered tasks, loops and job passes within the shutdown limit. A busy port
// returns an error before any loop starts. A rebuild leaves the logs open, because the process goes
// on.
func (a *App) Serve(ctx context.Context, address string) (ServeOutcome, error) {
	err := cleanFolder(a.environment.Config.TempFolder)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to clean temp folder", slog.Any("error", err))
	}

	handler, err := a.NewHandler()
	if err != nil {
		return ServeOutcome{}, err
	}

	loopsCtx, cancelLoops := context.WithCancel(context.WithoutCancel(ctx))

	defer cancelLoops()

	// The recovery fails the backups the previous run left in progress, so it runs before a request
	// can start a new one.
	err = a.StartModules(loopsCtx)
	if err != nil {
		return ServeOutcome{}, err
	}

	server, err := httpserver.Listen(ctx, address, handler)
	if err != nil {
		return ServeOutcome{}, err
	}

	a.logger.InfoContext(ctx, "http server listening", slog.String("address", server.GetAddress()))

	a.backgroundGroup = background.New(background.Deps{Logger: a.logger})

	a.runner.Start(loopsCtx)

	a.backgroundGroup.Go(loopsCtx, "http server", func(context.Context) error {
		return server.Serve()
	})

	isRebuildRequested := false

	select {
	case <-ctx.Done():
		a.logger.InfoContext(ctx, "shutdown signal received")
	case <-a.environment.RebuildRequest.GetSignal():
		isRebuildRequested = true
		a.logger.InfoContext(ctx, "stopping the application to publish an accepted recovery")
	}

	drainDeadline := a.shutdownBudget.Begin(ShutdownDrainTimeout)

	cancelLoops()

	flushLogs := a.environment.Logging.Shutdown
	if isRebuildRequested {
		flushLogs = nil
	}

	isDrained := shutdown.Drain(ctx, shutdown.DrainSpec{
		Steps: []shutdown.Step{
			{Name: "http server", Run: server.Shutdown},
			{Name: "background jobs", Run: a.runner.Wait},
			{Name: "backup runs", Run: a.backupsModule.WaitForRuns},
			{Name: "notification deliveries", Run: a.notifiersModule.WaitForDeliveries},
			{Name: "background loops", Run: a.backgroundGroup.Wait},
		},
		Deadline:     drainDeadline,
		FlushLogs:    flushLogs,
		FlushTimeout: shutdownLogFlushTimeout,
		Logger:       a.logger,
	})

	return ServeOutcome{IsRebuildRequested: isRebuildRequested, IsDrained: isDrained}, nil
}

// NewHandler builds the router with every route and middleware of the server. Serve and the test
// application share it, and it starts no background loop.
func (a *App) NewHandler() (http.Handler, error) {
	router := a.newRouter()

	err := a.registerContractRoutes(router)
	if err != nil {
		return nil, err
	}

	router.NoRoute(httpserver.HandleUnknownPath(frontendDirectory))

	return router, nil
}

func (a *App) newRouter() *gin.Engine {
	router := gin.New()
	router.ContextWithFallback = true
	router.Use(httpserver.AssignRequestID())
	router.Use(httpserver.RememberClientIP())
	router.Use(httpserver.LogAccess(a.logger))
	router.Use(httpserver.Recover(a.logger))
	router.Use(httpserver.NoStoreCacheControl())
	router.Use(httpserver.Compress([]string{
		`^/api/v1/logical-backups/[^/]+/file$`,
		`^/api/v1/physical-restore-sets$`,
		`^/api/v1/system/verification-agent$`,
	}))

	if a.environment.Config.EnvMode == config.EnvModeDevelopment {
		router.Use(httpserver.AllowAnyOrigin([]string{httpserver.RequestIDHeader}))
	}

	return router
}
