package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"databasus-backend/internal/config"
	"databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/backups/backups/backuping/logical"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	"databasus-backend/internal/features/dashboard"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/disk"
	"databasus-backend/internal/features/encryption/secrets"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/restores"
	"databasus-backend/internal/features/restores/restoring"
	"databasus-backend/internal/features/storages"
	system_agent "databasus-backend/internal/features/system/agent"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	system_version "databasus-backend/internal/features/system/version"
	task_cancellation "databasus-backend/internal/features/tasks/cancellation"
	"databasus-backend/internal/features/telemetry"
	users_controllers "databasus-backend/internal/features/users/controllers"
	users_middleware "databasus-backend/internal/features/users/middleware"
	users_services "databasus-backend/internal/features/users/services"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	"databasus-backend/internal/middleware"
	env_utils "databasus-backend/internal/util/env"
	files_utils "databasus-backend/internal/util/files"
	"databasus-backend/internal/util/logger"
	_ "databasus-backend/swagger" // swagger docs
)

// @title Databasus Backend API
// @version 1.0
// @description API for Databasus
// @termsOfService http://swagger.io/terms/

// @host localhost:4005
// @BasePath /api/v1
// @schemes http

const serverAddr = ":4005"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		runHealthcheckCommand() // exits the process
		return
	}

	commandLineOptions := parseCommandLineOptions()
	ctx := context.Background()
	log := logger.GetLogger()

	if commandLineOptions.shouldTestStorage {
		storageTestCommand := newStorageTestCommand(log, storageTestCommandWriters{
			standardOutput: os.Stdout,
			errorOutput:    os.Stderr,
		})
		logger.ExitAfterFlush(storageTestCommand.SaveAndDeleteProbe(ctx))
		return
	}

	runMigrations(log)

	// create directories that used for backups and restore
	err := files_utils.EnsureDirectories([]string{
		config.GetEnv().TempFolder,
		config.GetEnv().DataFolder,
	})
	if err != nil {
		log.Error("failed to ensure directories", "error", err)
		logger.ExitAfterFlush(1)
	}

	err = secrets.GetSecretKeyService().MigrateKeyFromDbToFileIfExist()
	if err != nil {
		log.Error("failed to migrate secret key from database to file", "error", err)
		logger.ExitAfterFlush(1)
	}

	listAdminsIfRequested(ctx, log, commandLineOptions)

	resetPasswordIfRequested(ctx, log, commandLineOptions)

	disableTwoFactorAuthIfRequested(ctx, log, commandLineOptions)

	go generateSwaggerDocs(log)

	gin.SetMode(gin.ReleaseMode)
	ginApp := gin.New()
	ginApp.Use(middleware.AssignRequestID())
	ginApp.Use(middleware.LogAccess(log))
	ginApp.Use(ginRecoveryWithLogger(log))
	ginApp.Use(middleware.NoStoreCacheControl())

	// Add GZIP compression middleware
	ginApp.Use(gzip.Gzip(
		gzip.DefaultCompression,
		// Don't compress already compressed files
		gzip.WithExcludedExtensions(
			[]string{".png", ".gif", ".jpeg", ".jpg", ".ico", ".svg", ".pdf", ".mp4"},
		),
	))

	enableCors(ginApp)
	setUpRoutes(ginApp)
	setUpDependencies()

	announceTelemetry()

	runBackgroundTasks(log)

	mountFrontend(ginApp)

	startServerWithGracefulShutdown(log, ginApp)
}

type commandLineOptions struct {
	shouldTestStorage      bool
	shouldListAdmins       bool
	shouldDisableTwoFactor bool
	newPassword            string
	passwordResetEmail     string
}

func parseCommandLineOptions() commandLineOptions {
	shouldTestStorage := flag.Bool(
		config.StorageProbeFlagName,
		false,
		"Save and delete a local storage probe",
	)
	shouldListAdmins := flag.Bool("list-admins", false, "Print every administrator account of this instance")
	// The name carries a digit, unlike every other flag here, because it is the
	// term an owner locked out by a dead mail server searches for.
	shouldDisableTwoFactor := flag.Bool(
		"disable-2fa",
		false,
		"Stop requiring an emailed code at sign-in",
	)
	newPassword := flag.String("new-password", "", "Set a new password for the user")
	passwordResetEmail := flag.String("email", "", "Email of the user to reset password")
	flag.Parse()

	return commandLineOptions{
		shouldTestStorage:      *shouldTestStorage,
		shouldListAdmins:       *shouldListAdmins,
		shouldDisableTwoFactor: *shouldDisableTwoFactor,
		newPassword:            *newPassword,
		passwordResetEmail:     *passwordResetEmail,
	}
}

// The listing only reads, so unlike the password reset below it installs no
// audit writer - the console path has none, and UserService dereferences its
// writer without a nil check.
func listAdminsIfRequested(
	ctx context.Context,
	log *slog.Logger,
	commandLineOptions commandLineOptions,
) {
	if !commandLineOptions.shouldListAdmins {
		return
	}

	listing, err := users_services.GetManagementService().RenderAdminListing(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to list administrators", "error", err)
		logger.ExitAfterFlush(1)
	}

	// Printed rather than logged: the owner runs this to read the addresses, and
	// the logger masks them as the PII they otherwise are.
	fmt.Println(listing)

	logger.ExitAfterFlush(0)
}

func disableTwoFactorAuthIfRequested(
	ctx context.Context,
	log *slog.Logger,
	commandLineOptions commandLineOptions,
) {
	if !commandLineOptions.shouldDisableTwoFactor {
		return
	}

	audit_logs.SetupDependencies()

	isChanged, err := users_services.GetSettingsService().DisableTwoFactorAuth(ctx)
	if err != nil {
		log.ErrorContext(ctx, "failed to switch the second factor off", "error", err)
		logger.ExitAfterFlush(1)
	}

	if isChanged {
		fmt.Println("Two-factor authentication is now off. The next password sign-in needs no code.")
	} else {
		fmt.Println("Two-factor authentication was already off. Nothing changed.")
	}

	logger.ExitAfterFlush(0)
}

func resetPasswordIfRequested(
	ctx context.Context,
	log *slog.Logger,
	commandLineOptions commandLineOptions,
) {
	if commandLineOptions.newPassword == "" {
		return
	}

	audit_logs.SetupDependencies()
	log.InfoContext(ctx, "found reset password command, resetting password")

	if commandLineOptions.passwordResetEmail == "" {
		log.InfoContext(ctx, "no email provided, pass one via --email=\"some@email.com\"")
		logger.ExitAfterFlush(1)
	}

	resetPassword(
		ctx,
		commandLineOptions.passwordResetEmail,
		commandLineOptions.newPassword,
		log,
	)
}

func resetPassword(ctx context.Context, email, newPassword string, log *slog.Logger) {
	log.InfoContext(ctx, "resetting password")

	userService := users_services.GetUserService()
	err := userService.ChangeUserPasswordByEmail(ctx, email, newPassword)
	if err != nil {
		log.ErrorContext(ctx, "failed to reset password", "error", err)
		logger.ExitAfterFlush(1)
	}

	log.InfoContext(ctx, "password reset successfully")
	logger.ExitAfterFlush(0)
}

func startServerWithGracefulShutdown(log *slog.Logger, app *gin.Engine) {
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: app,
	}

	log.Info(fmt.Sprintf("http server listening on %s", serverAddr))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("http server failed to listen", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Info("shutdown signal received")

	// The context is used to inform the server it has 10 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}

	log.Info("server gracefully stopped")

	// Closed after the drain so in-flight request logs are included, on its own deadline so a
	// slow drain cannot hand the flush an already-expired context.
	logger.FlushAndCloseSinks()
}

func setUpRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// Mount Swagger UI
	v1.GET("/docs/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes (only user auth routes and healthcheck should be public)
	userController := users_controllers.GetUserController()
	userController.RegisterRoutes(v1)
	system_healthcheck.GetHealthcheckController().RegisterRoutes(v1)
	system_version.GetVersionController().RegisterRoutes(v1)
	system_agent.GetAgentController().RegisterRoutes(v1)
	backups_controllers_logical.GetBackupController().RegisterPublicRoutes(v1)
	backups_controllers_physical.GetPhysicalBackupController().RegisterPublicRoutes(v1)
	databases.GetDatabaseController().RegisterPublicRoutes(v1)
	verification_agents.GetAgentFacingController().RegisterRoutes(v1)
	verification_runs.GetVerificationAgentController().RegisterRoutes(v1)

	// Setup auth middleware
	userService := users_services.GetUserService()
	authMiddleware := users_middleware.AuthMiddleware(userService)

	// Protected routes
	protected := v1.Group("")
	protected.Use(authMiddleware)

	userController.RegisterProtectedRoutes(protected)
	workspaces_controllers.GetWorkspaceController().RegisterRoutes(protected)
	workspaces_controllers.GetMembershipController().RegisterRoutes(protected)
	disk.GetDiskController().RegisterRoutes(protected)
	notifiers.GetNotifierController().RegisterRoutes(protected)
	storages.GetStorageController().RegisterRoutes(protected)
	databases.GetDatabaseController().RegisterRoutes(protected)
	backups_controllers_logical.GetBackupController().RegisterRoutes(protected)
	backups_controllers_physical.GetPhysicalBackupController().RegisterRoutes(protected)
	restores.GetRestoreController().RegisterRoutes(protected)
	healthcheck_config.GetHealthcheckConfigController().RegisterRoutes(protected)
	healthcheck_attempt.GetHealthcheckAttemptController().RegisterRoutes(protected)
	dashboard.GetDashboardController().RegisterRoutes(protected)
	backups_config_logical.GetBackupConfigController().RegisterRoutes(protected)
	backups_config_physical.GetBackupConfigController().RegisterRoutes(protected)
	audit_logs.GetAuditLogController().RegisterRoutes(protected)
	users_controllers.GetManagementController().RegisterRoutes(protected)
	users_controllers.GetSettingsController().RegisterRoutes(protected)
	verification_agents.GetAgentController().RegisterRoutes(protected)
	verification_config.GetVerificationConfigController().RegisterRoutes(protected)
	verification_runs.GetVerificationController().RegisterRoutes(protected)
}

func setUpDependencies() {
	databases.SetupDependencies()
	backups_services.SetupDependencies()
	restores.SetupDependencies()
	healthcheck_config.SetupDependencies()
	audit_logs.SetupDependencies()
	notifiers.SetupDependencies()
	storages.SetupDependencies()
	backups_config_logical.SetupDependencies()
	backups_config_physical.SetupDependencies()
	backuping_physical.SetupDependencies()
	verification_config.SetupDependencies()
	verification_runs.SetupDependencies()
	task_cancellation.SetupDependencies()

	telemetry.SetupDependencies()
}

func announceTelemetry() {
	if config.GetEnv().IsDisableAnonymousTelemetry {
		return
	}

	fmt.Println(
		"Anonymous telemetry collected (Databasus version, OS/arch, etc.). No DB contents, no user data. " +
			"To disable, set IS_DISABLE_ANONYMOUS_TELEMETRY=true in your .env",
	)
}

func runBackgroundTasks(log *slog.Logger) {
	log.Info("preparing to run background tasks")

	// Create context that will be cancelled on shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Set up signal handling for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		log.Info("shutdown signal received, cancelling all background tasks")
		cancel()
	}()

	err := files_utils.CleanFolder(config.GetEnv().TempFolder)
	if err != nil {
		log.Error("failed to clean temp folder", "error", err)
	}

	log.Info("starting background tasks")

	go runWithPanicLogging(log, "backup background service", func() {
		backuping_logical.GetBackupsScheduler().Run(ctx)
	})

	go runWithPanicLogging(log, "verification scheduler", func() {
		verification_runs.GetVerificationScheduler().Run(ctx)
	})

	go runWithPanicLogging(log, "backup cleaner background service", func() {
		backuping_logical.GetBackupCleaner().Run(ctx)
	})

	go runWithPanicLogging(log, "physical backup scheduler background service", func() {
		backuping_physical.GetPhysicalBackupsScheduler().Run(ctx)
	})

	go runWithPanicLogging(log, "physical backup cleaner background service", func() {
		backuping_physical.GetPhysicalBackupCleaner().Run(ctx)
	})

	go runWithPanicLogging(log, "restore background service", func() {
		restoring.GetRestoresScheduler().Run(ctx)
	})

	go runWithPanicLogging(log, "healthcheck attempt background service", func() {
		healthcheck_attempt.GetHealthcheckAttemptBackgroundService().Run(ctx)
	})

	go runWithPanicLogging(log, "audit log cleanup background service", func() {
		audit_logs.GetAuditLogBackgroundService().Run(ctx)
	})

	go runWithPanicLogging(log, "download token cleanup background service", func() {
		backups_download.GetDownloadTokenBackgroundService().Run(ctx)
	})

	go runWithPanicLogging(log, "physical wal stream supervisor background service", func() {
		backuping_physical.GetPhysicalWalStreamSupervisor().Run(ctx)
	})

	go runWithPanicLogging(log, "storage file deletion background service", func() {
		storages.GetStorageFileDeletionWorker().Run(ctx)
	})

	go runWithPanicLogging(log, "sign-in code cleanup background service", func() {
		users_services.GetSignInCodeBackgroundService().Run(ctx)
	})

	go runWithPanicLogging(log, "telemetry background service", func() {
		telemetry.GetTelemetryBackgroundService().Run(ctx)
	})
}

func runWithPanicLogging(log *slog.Logger, serviceName string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			log.Error("panic in "+serviceName, "error", r, "stacktrace", string(debug.Stack()))
		}
	}()
	fn()
}

// Keep in mind: docs appear after second launch, because Swagger
// is generated into Go files. So if we changed files, we generate
// new docs, but still need to restart the server to see them.
func generateSwaggerDocs(log *slog.Logger) {
	if config.GetEnv().EnvMode == env_utils.EnvModeProduction {
		return
	}

	// Run swag from the current directory instead of parent
	// Use the current directory as the base for swag init
	// This ensures swag can find the files regardless of where the command is run from
	currentDir, err := os.Getwd()
	if err != nil {
		log.Error("failed to get current directory", "error", err)
		return
	}

	cmd := exec.CommandContext(
		context.Background(), "swag", "init", "-d", currentDir, "-g", "cmd/main.go", "-o", "swagger",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("failed to generate Swagger docs", "error", err, "output", string(output))
		return
	}

	log.Info("swagger documentation generated successfully")
}

func runMigrations(log *slog.Logger) {
	log.Info("running database migrations")

	cmd := exec.CommandContext(context.Background(), "goose", "-dir", "./migrations", "up")
	cmd.Env = append(
		os.Environ(),
		"GOOSE_DRIVER=postgres",
		"GOOSE_DBSTRING="+config.GetEnv().DatabaseDsn,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Error("failed to run migrations", "error", err, "output", string(output))
		logger.ExitAfterFlush(1)
	}

	log.Info("database migrations completed successfully", "output", string(output))
}

func enableCors(ginApp *gin.Engine) {
	if config.GetEnv().EnvMode == env_utils.EnvModeDevelopment {
		// Setup CORS
		ginApp.Use(cors.New(cors.Config{
			AllowOrigins: []string{"*"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
			AllowHeaders: []string{
				"Origin",
				"Content-Length",
				"Content-Type",
				"Authorization",
				"Accept",
				"Accept-Language",
				"Accept-Encoding",
				"Access-Control-Request-Method",
				"Access-Control-Request-Headers",
				"Access-Control-Allow-Methods",
				"Access-Control-Allow-Headers",
				"Access-Control-Allow-Origin",
			},
			ExposeHeaders:    []string{middleware.RequestIDHeader},
			AllowCredentials: true,
		}))
	}
}

func ginRecoveryWithLogger(log *slog.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.ErrorContext(ctx.Request.Context(), "panic recovered in HTTP handler",
					"error", r,
					"stacktrace", string(debug.Stack()),
					"method", ctx.Request.Method,
					"path", ctx.Request.URL.Path,
				)

				ctx.AbortWithStatus(http.StatusInternalServerError)
			}
		}()

		ctx.Next()
	}
}

func mountFrontend(ginApp *gin.Engine) {
	staticDir := "./ui/build"
	ginApp.NoRoute(func(c *gin.Context) {
		path := filepath.Join(staticDir, c.Request.URL.Path)

		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			c.File(path)
			return
		}

		c.File(filepath.Join(staticDir, "index.html"))
	})
}
