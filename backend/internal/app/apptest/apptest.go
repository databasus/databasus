// Package apptest builds the real application for backend API tests: the routes and middleware of
// the server, an internal database of the test's own, a clock the test moves and background jobs
// that normally run only when the test runs their passes. Scheduling tests start selected jobs
// with StartJobRunner. Physical backup loops run only with WithPhysicalBackupLoops.
package apptest

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"gorm.io/gorm"

	"databasus-backend/internal/app"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/internal/lib/testwait"
)

type App struct {
	// URL is the address of the server without /api/v1, the base URL apiclient.New takes.
	URL string
	// Clock is nil with live physical jobs; otherwise it starts at real time and moves with the test.
	Clock *clock.Manual
	// Mail holds every message the application sent.
	Mail *MailRecorder
	// GitHub and Google are the fake identity providers of WithGitHub and WithGoogle, nil without
	// the option.
	GitHub *FakeGitHub
	Google *FakeGoogle

	dataFolder                string
	tempFolder                string
	databaseDsn               string
	secretKeyPath             string
	app                       *app.App
	environment               *app.Environment
	stopModules               context.CancelFunc
	cancelPhysicalLoops       context.CancelFunc
	physicalRunner            *background.Runner
	physicalBackupStopTimeout time.Duration
}

// backupRunsWaitTimeout bounds the wait for the backup runs a test leaves behind; a run whose context
// ended writes its outcome at once.
const backupRunsWaitTimeout = 30 * time.Second

func New(tb testing.TB, options ...Option) *App {
	tb.Helper()

	applied := settings{
		isTelemetryEnabled:           false,
		isPhysicalBackupLoopsEnabled: false,
		physicalBackupStopTimeout:    2 * time.Minute,
		isMailServerMissing:          false,
		isTurnstileEnabled:           false,
		isGitHubEnabled:              false,
		isGoogleEnabled:              false,
		publicURL:                    "",
		clientToolsDir:               "",
		agentBinariesDir:             "",
		logSink:                      nil,
		walStreamTimings:             nil,
		storageFileTimings:           nil,
		notificationDeliveryTimeout:  0,
		// A test never reaches the real Slack or Telegram: an unset address points at a host that
		// does not resolve.
		notifierEndpoints: NotifierEndpoints{
			SlackPostMessageURL: "http://slack.invalid/api/chat.postMessage",
			TelegramBotAPIURL:   "http://telegram.invalid",
		},
	}
	for _, option := range options {
		option(&applied)
	}

	databaseDsn := testdb.Open(tb)

	manualClock := clock.NewManual(time.Now())
	var appClock clock.Source = manualClock
	if applied.isPhysicalBackupLoopsEnabled || applied.agentBinariesDir != "" {
		appClock = clock.NewSystem()
		manualClock = nil
	}

	appConfig := newConfig(tb, databaseDsn)
	appConfig.DatabasusURL = applied.publicURL
	appConfig.IsDisableAnonymousTelemetry = !applied.isTelemetryEnabled
	if applied.telemetryIdentityPath != "" {
		appConfig.TelemetryInstancePath = applied.telemetryIdentityPath
	}

	if applied.isTurnstileEnabled {
		appConfig.CloudflareTurnstileSecretKey = turnstileSecretKey
	}

	if applied.isGitHubEnabled {
		appConfig.GitHubClientID = gitHubClientID
		appConfig.GitHubClientSecret = gitHubClientSecret
	}

	if applied.isGoogleEnabled {
		appConfig.GoogleClientID = googleClientID
		appConfig.GoogleClientSecret = googleClientSecret
	}

	environment, err := app.NewEnvironment(tb.Context(), appConfig, appClock)
	if err != nil {
		tb.Fatalf("build the environment of the test application: %v", err)
	}

	configuredLogging := environment.Logging
	tb.Cleanup(func() {
		flushCtx, cancelFlush := context.WithTimeout(context.WithoutCancel(tb.Context()), 5*time.Second)
		defer cancelFlush()
		if err := configuredLogging.Shutdown(flushCtx); err != nil {
			tb.Errorf("flush test application logs: %v", err)
		}
	})

	if applied.logSink != nil {
		environment.Logging = logging.NewWithSinks(configuredLogging.GetLogger().Handler(), applied.logSink)
	}

	if applied.isTurnstileEnabled {
		environment.ExternalEndpoints.TurnstileVerifyURL = startFakeTurnstile(tb)
	}

	var fakeGitHub *FakeGitHub
	if applied.isGitHubEnabled {
		fakeGitHub = startFakeGitHub(tb)
		environment.ExternalEndpoints.GitHubAuthURL = fakeGitHub.URL + "/login/oauth/authorize"
		environment.ExternalEndpoints.GitHubTokenURL = fakeGitHub.URL + "/login/oauth/access_token"
		environment.ExternalEndpoints.GitHubUserURL = fakeGitHub.URL + "/user"
		environment.ExternalEndpoints.GitHubEmailsURL = fakeGitHub.URL + "/user/emails"
	}

	var fakeGoogle *FakeGoogle
	if applied.isGoogleEnabled {
		fakeGoogle = startFakeGoogle(tb)
		environment.ExternalEndpoints.GoogleAuthURL = fakeGoogle.URL + "/auth"
		environment.ExternalEndpoints.GoogleTokenURL = fakeGoogle.URL + "/token"
		environment.ExternalEndpoints.GoogleUserInfoURL = fakeGoogle.URL + "/userinfo"
	}

	if applied.clientToolsDir != "" {
		environment.ClientToolsDir = applied.clientToolsDir
	}

	if applied.agentBinariesDir != "" {
		environment.AgentBinariesDir = applied.agentBinariesDir
	}

	if applied.telemetryReceiverURL != "" {
		environment.ExternalEndpoints.TelemetryReceiverURL = applied.telemetryReceiverURL
	}
	environment.ExternalEndpoints.SlackPostMessageURL = applied.notifierEndpoints.SlackPostMessageURL
	environment.ExternalEndpoints.TelegramBotAPIURL = applied.notifierEndpoints.TelegramBotAPIURL

	if applied.notificationDeliveryTimeout > 0 {
		environment.NotificationDeliveryTimeout = applied.notificationDeliveryTimeout
	}

	environment.StorageFileTimings = getTestFileTimings()
	if applied.storageFileTimings != nil {
		environment.StorageFileTimings = *applied.storageFileTimings
	}

	if applied.walStreamTimings != nil {
		environment.WalStreamTimings = *applied.walStreamTimings
	}

	mailRecorder := newMailRecorder(!applied.isMailServerMissing)
	environment.Mailer = mailRecorder

	createdApp, err := app.New(tb.Context(), environment)
	if err != nil {
		tb.Fatalf("build the test application: %v", err)
	}

	tb.Cleanup(createdApp.Close)

	stopModules := startModules(tb, createdApp)

	handler, err := createdApp.NewHandler()
	if err != nil {
		stopModules()
		tb.Fatalf("build the handler of the test application: %v", err)
	}

	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)

	testApp := &App{
		URL:                       server.URL,
		Clock:                     manualClock,
		Mail:                      mailRecorder,
		GitHub:                    fakeGitHub,
		Google:                    fakeGoogle,
		dataFolder:                appConfig.DataFolder,
		tempFolder:                appConfig.TempFolder,
		databaseDsn:               databaseDsn,
		secretKeyPath:             appConfig.SecretKeyPath,
		app:                       createdApp,
		environment:               environment,
		stopModules:               stopModules,
		cancelPhysicalLoops:       nil,
		physicalRunner:            nil,
		physicalBackupStopTimeout: applied.physicalBackupStopTimeout,
	}
	if applied.isPhysicalBackupLoopsEnabled {
		testApp.startPhysicalLoops(tb)
	}
	tb.Cleanup(func() { testApp.stop(tb) })

	return testApp
}

// ExecuteSQL runs a statement on the database of the test application. A test uses it only for a
// state the API cannot create.
func (a *App) ExecuteSQL(tb testing.TB, statement string, values ...any) {
	tb.Helper()

	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the database of the test application: %v", err)
	}

	defer closeConnections(db)

	err = db.WithContext(tb.Context()).Exec(statement, values...).Error
	if err != nil {
		tb.Fatalf("run %q: %v", statement, err)
	}
}

// QuerySQL reads rows of the database of the test application into destination. A test uses it
// only for data the API does not return.
func (a *App) QuerySQL(tb testing.TB, destination any, query string, values ...any) {
	tb.Helper()

	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the database of the test application: %v", err)
	}

	defer closeConnections(db)

	err = db.WithContext(tb.Context()).Raw(query, values...).Scan(destination).Error
	if err != nil {
		tb.Fatalf("run %q: %v", query, err)
	}
}

func getPhysicalLoopJobs() []string {
	return []string{
		"backups_physical_scheduler",
		"backups_physical_request_scheduler",
		"backups_physical_wal_stream_manager",
	}
}

// RunJobPass runs one pass of the named background job and fails the test when the pass fails or
// no job has the name.
func (a *App) RunJobPass(tb testing.TB, name string) {
	tb.Helper()

	if name == "backups_physical_wal_stream_manager" && a.physicalRunner == nil {
		tb.Fatalf("backups_physical_wal_stream_manager runs only with live physical jobs")
		return
	}
	var err error
	if a.physicalRunner != nil && slices.Contains(getPhysicalLoopJobs(), name) {
		err = a.physicalRunner.RunPass(tb.Context(), name)
	} else {
		err = a.app.RunJobPass(tb.Context(), name)
	}
	if err != nil {
		tb.Fatalf("run a pass of the background job %s: %v", name, err)
	}
}

// GetPhysicalLoopSnapshots returns the state of the live physical jobs, and nothing without them.
func (a *App) GetPhysicalLoopSnapshots() []background.JobSnapshot {
	if a.physicalRunner == nil {
		return nil
	}

	return a.physicalRunner.GetSnapshots()
}

// CountInternalDatabaseQueries returns how many statements the application sends to its internal database
// while run works. The test keeps other work of the application out of that time.
func (a *App) CountInternalDatabaseQueries(tb testing.TB, run func()) int64 {
	tb.Helper()

	queryCount, err := a.app.CountInternalDatabaseQueries(run)
	if err != nil {
		tb.Fatalf("count internal database queries: %v", err)
	}

	return queryCount
}

func (a *App) StopModules() {
	a.stopModules()
}

// RestartModules stops the modules as a shutdown does, waits for the runs to write their outcome and
// starts the modules again on the same database, as the next start of the application does.
func (a *App) RestartModules(tb testing.TB) {
	tb.Helper()

	hasPhysicalLoops := a.physicalRunner != nil
	waitCtx, cancelWait := context.WithTimeout(context.WithoutCancel(tb.Context()), app.ShutdownDrainTimeout)
	defer cancelWait()

	a.StopModules()
	if hasPhysicalLoops {
		if err := a.stopPhysicalWork(waitCtx); err != nil {
			tb.Fatalf("stop physical backup loops before restarting modules: %v", err)
		}
	}
	if err := a.app.WaitForBackupRuns(waitCtx); err != nil {
		tb.Fatalf("wait for the backup runs of the test application: %v", err)
	}
	if err := a.app.WaitForNotificationDeliveries(waitCtx); err != nil {
		tb.Fatalf("wait for notification deliveries before restarting modules: %v", err)
	}

	a.stopModules = startModules(tb, a.app)
	if hasPhysicalLoops {
		a.startPhysicalLoops(tb)
	}
}

func startModules(tb testing.TB, createdApp *app.App) context.CancelFunc {
	tb.Helper()
	modulesCtx, stopModules := context.WithCancel(context.WithoutCancel(tb.Context()))
	if err := createdApp.StartModules(modulesCtx); err != nil {
		stopModules()
		tb.Fatalf("start the modules of the test application: %v", err)
	}
	return stopModules
}

func (a *App) WaitForPhysicalBackups(tb testing.TB) {
	tb.Helper()
	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open physical backup catalog: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		tb.Fatalf("open physical backup catalog connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	testwait.Until(tb, "physical backups record their outcome", func(ctx context.Context) error {
		var runningCount int64
		err := db.WithContext(ctx).
			Raw("SELECT count(*) FROM backups_physical WHERE status = 'IN_PROGRESS'").
			Scan(&runningCount).
			Error
		if err != nil {
			return err
		}
		if runningCount > 0 {
			return fmt.Errorf("%d physical backups are still running", runningCount)
		}
		return nil
	})
}

func (a *App) startPhysicalLoops(tb testing.TB) {
	tb.Helper()
	loopsCtx, cancelLoops := context.WithCancel(context.WithoutCancel(tb.Context()))
	a.cancelPhysicalLoops = cancelLoops
	a.physicalRunner = a.app.NewJobRunner(getPhysicalLoopJobs())
	a.physicalRunner.Start(loopsCtx)
}

func (a *App) stopPhysicalWork(ctx context.Context) error {
	a.cancelPhysicalLoops()
	if err := a.physicalRunner.Wait(ctx); err != nil {
		return err
	}
	a.cancelPhysicalLoops = nil
	a.physicalRunner = nil
	return nil
}

// newConfig sets the database after loading, so a DATABASE_DSN in the environment of the test run
// can never point a test at another database.
func newConfig(tb testing.TB, databaseDsn string) config.Config {
	tb.Helper()

	rootDirectory := tb.TempDir()
	if err := os.WriteFile(filepath.Join(rootDirectory, ".env"), []byte("ENV_MODE=development\n"), 0o600); err != nil {
		tb.Fatalf("write test configuration: %v", err)
	}

	loadedConfig, _, err := config.Load(config.LoadSpec{
		IsDatabaseRequired:       false,
		RootDirectory:            rootDirectory,
		PublishedDatabaseDsnPath: filepath.Join(rootDirectory, "never-published"),
	})
	if err != nil {
		tb.Fatalf("load the configuration of the test application: %v", err)
	}

	loadedConfig.DatabaseDsn = databaseDsn

	// Settings of the developer's environment must not reach a test: each fake is turned on by an
	// option of New.
	loadedConfig.CloudflareTurnstileSecretKey = ""
	loadedConfig.GitHubClientID = ""
	loadedConfig.GitHubClientSecret = ""
	loadedConfig.GoogleClientID = ""
	loadedConfig.GoogleClientSecret = ""

	return loadedConfig
}

func closeConnections(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func (a *App) stop(tb testing.TB) {
	tb.Helper()
	if a.physicalRunner != nil {
		stopCtx, cancelStop := context.WithTimeout(context.WithoutCancel(tb.Context()), a.physicalBackupStopTimeout)
		err := a.stopPhysicalWork(stopCtx)
		cancelStop()
		if err != nil {
			tb.Errorf("stop physical backup loops and runs: %v", err)
		}
	}

	a.stopModules()
	waitCtx, cancelWait := context.WithTimeout(context.WithoutCancel(tb.Context()), backupRunsWaitTimeout)
	defer cancelWait()
	if err := a.app.WaitForBackupRuns(waitCtx); err != nil {
		tb.Errorf("wait for the backup runs of the test application: %v", err)
	}

	if err := a.app.WaitForNotificationDeliveries(waitCtx); err != nil {
		tb.Errorf("wait for notification deliveries of the test application: %v", err)
	}
}
