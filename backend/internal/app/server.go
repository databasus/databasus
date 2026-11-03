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

// Serve opens the port, starts the background loops, the background jobs and the HTTP server and
// blocks until ctx ends. Then it stops accepting work and waits for requests, registered tasks,
// loops and job passes within the shutdown limit. A busy port returns an error before any loop
// starts.
func (a *App) Serve(ctx context.Context, address string) error {
	err := cleanFolder(a.environment.Config.TempFolder)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to clean temp folder", slog.Any("error", err))
	}

	handler, err := a.NewHandler()
	if err != nil {
		return err
	}

	loopsCtx, cancelLoops := context.WithCancel(context.WithoutCancel(ctx))

	defer cancelLoops()

	// The recovery fails the backups the previous run left in progress, so it runs before a request
	// can start a new one.
	err = a.StartModules(loopsCtx)
	if err != nil {
		return err
	}

	server, err := httpserver.Listen(ctx, address, handler)
	if err != nil {
		return err
	}

	a.logger.InfoContext(ctx, "http server listening", slog.String("address", server.GetAddress()))

	a.backgroundGroup = background.New(background.Deps{Logger: a.logger})

	a.runner.Start(loopsCtx)

	a.backgroundGroup.Go(loopsCtx, "http server", func(context.Context) error {
		return server.Serve()
	})

	<-ctx.Done()

	drainDeadline := a.shutdownBudget.Begin(ShutdownDrainTimeout)
	a.logger.InfoContext(ctx, "shutdown signal received")

	cancelLoops()

	shutdown.Drain(ctx, shutdown.DrainSpec{
		Steps: []shutdown.Step{
			{Name: "http server", Run: server.Shutdown},
			{Name: "background jobs", Run: a.runner.Wait},
			{Name: "backup runs", Run: a.backupsModule.WaitForRuns},
			{Name: "notification deliveries", Run: a.notifiersModule.WaitForDeliveries},
			{Name: "background loops", Run: a.backgroundGroup.Wait},
		},
		Deadline:     drainDeadline,
		FlushLogs:    a.environment.Logging.Shutdown,
		FlushTimeout: shutdownLogFlushTimeout,
		Logger:       a.logger,
	})

	return nil
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

// newRouter builds the engine with the middleware every route shares. The strict handlers receive
// the *gin.Context as their context.Context, so ContextWithFallback lets them read the values the
// middleware put into the request context, such as the actor and the client IP.
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
