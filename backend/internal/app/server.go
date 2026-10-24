package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	old_config "databasus-backend/internal/config"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/httpserver"
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/middleware"
	env_utils "databasus-backend/internal/util/env"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/tools"
)

// The drain and the log flush together stay under 8 seconds, which leaves the embedded PostgreSQL
// server time for a clean shutdown within Docker's default stop timeout of 10 seconds.
const (
	shutdownDrainTimeout    = 6 * time.Second
	shutdownLogFlushTimeout = 2 * time.Second
)

const frontendDirectory = "./ui/build"

// Serve opens the port, starts the background loops and the HTTP server and blocks until ctx ends.
// Then it stops accepting work and waits for requests, registered tasks and loops within the
// shutdown limit. A busy port returns an error before any loop starts.
func (a *App) Serve(ctx context.Context, address string) error {
	tools.LogAndExitIfClientToolsBroken(a.logger, a.environment.Config.ShowDbInstallationVerificationLogs)

	err := files_utils.CleanFolder(a.environment.Config.TempFolder)
	if err != nil {
		a.logger.ErrorContext(ctx, "failed to clean temp folder", slog.Any("error", err))
	}

	handler, err := a.NewHandler()
	if err != nil {
		return err
	}

	server, err := httpserver.Listen(ctx, address, handler)
	if err != nil {
		return err
	}

	a.logger.InfoContext(ctx, "http server listening", slog.String("address", server.GetAddress()))

	a.backgroundGroup = background.New(background.Deps{Logger: a.logger})
	loopsCtx, cancelLoops := context.WithCancel(context.WithoutCancel(ctx))

	defer cancelLoops()

	a.startBackgroundLoops(loopsCtx)

	a.backgroundGroup.Go(loopsCtx, "http server", func(context.Context) error {
		return server.Serve()
	})

	<-ctx.Done()

	a.logger.InfoContext(ctx, "shutdown signal received")

	old_config.BeginShutdown()
	cancelLoops()

	shutdown.Drain(ctx, shutdown.DrainSpec{
		Steps: []shutdown.Step{
			{Name: "http server", Run: server.Shutdown},
			{Name: "running tasks", Run: task_cancellation.GetRegistry().WaitUntilNoTasksRegistered},
			{Name: "background loops", Run: a.backgroundGroup.Wait},
		},
		DrainTimeout: shutdownDrainTimeout,
		FlushLogs:    a.environment.Logging.Shutdown,
		FlushTimeout: shutdownLogFlushTimeout,
		Logger:       a.logger,
	})

	return nil
}

// NewHandler builds the router with every route and middleware of the server. Serve and the test
// application share it, and it starts no background loop.
func (a *App) NewHandler() (http.Handler, error) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.AssignRequestID())
	router.Use(middleware.LogAccess(a.logger))
	router.Use(httpserver.Recover(a.logger))
	router.Use(middleware.NoStoreCacheControl())
	router.Use(httpserver.Compress())

	if a.environment.Config.EnvMode == env_utils.EnvModeDevelopment {
		router.Use(httpserver.AllowAnyOrigin([]string{middleware.RequestIDHeader}))
	}

	err := a.registerContractRoutes(router)
	if err != nil {
		return nil, err
	}

	registerRoutes(router)

	router.NoRoute(httpserver.HandleUnknownPath(frontendDirectory))

	return router, nil
}
