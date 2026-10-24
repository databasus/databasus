package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/internal/lib/testwait"
)

const (
	operatorPassword      = "operator-password-123"
	firstMigration        = "20250605090323_init.sql"
	newerMigrationVersion = 99991231235959
)

// binaryPath is the server binary built without the testhelpers tag, as the image builds it. A
// getter that the bridge does not feed panics there instead of loading test settings.
var binaryPath string

func TestMain(m *testing.M) {
	buildDirectory, err := os.MkdirTemp("", "databasus-binary-")
	if err != nil {
		panic(err)
	}

	binaryPath = filepath.Join(buildDirectory, "databasus")

	buildCommand := exec.CommandContext(context.Background(), "go", "build", "-o", binaryPath, ".")
	//nolint:forbidigo // go build needs the toolchain environment of the test process
	buildCommand.Env = append(os.Environ(), "GOFLAGS=")

	output, err := buildCommand.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("build the server binary: %v\n%s", err, output))
	}

	exitCode := m.Run()

	_ = os.RemoveAll(buildDirectory)
	os.Exit(exitCode)
}

func TestBinary(t *testing.T) {
	t.Run("reports the server state of the healthcheck subcommand without any setting", func(t *testing.T) {
		isServerRunning := checkServerAlive(t.Context(), "http://localhost"+serverAddress) == nil

		binaryRun := runBinary(t, newOperatorHost(t), nil, healthcheckSubcommand)

		assert.NotContains(t, binaryRun.output, "panic")

		if isServerRunning {
			assert.Equal(t, 0, binaryRun.exitCode, binaryRun.output)
			assert.Contains(t, binaryRun.output, "healthcheck ok")

			return
		}

		assert.Equal(t, 1, binaryRun.exitCode, binaryRun.output)
		assert.Contains(t, binaryRun.output, "healthcheck failed")
	})

	t.Run("stops without an environment file when the connection string is empty", func(t *testing.T) {
		binaryRun := runBinary(t, newOperatorHost(t), map[string]string{"ENV_MODE": "production", "DATABASE_DSN": ""})

		assert.NotEqual(t, 0, binaryRun.exitCode, binaryRun.output)
		assert.Contains(t, binaryRun.output, "no .env file found")
		assert.Contains(t, binaryRun.output, "required environment variable is not set: DATABASE_DSN")
		assert.NotContains(t, binaryRun.output, "http server listening")
	})

	t.Run("saves and deletes the storage probe without a connection string", func(t *testing.T) {
		host := newOperatorHost(t)

		binaryRun := runBinary(t, host, map[string]string{"ENV_MODE": "production"}, "--test-storage")

		require.Equal(t, 0, binaryRun.exitCode, binaryRun.output)
		assert.Contains(t, binaryRun.output, "storage test passed")

		leftProbes, err := filepath.Glob(filepath.Join(host.dataDirectory, "backups", ".databasus-storage-test-*"))
		require.NoError(t, err)
		assert.Empty(t, leftProbes)
	})

	t.Run("runs every console command on a new database without opening the server port", func(t *testing.T) {
		host := newOperatorHost(t)
		environment := newDatabaseEnvironment(testdb.OpenEmpty(t))
		email := "owner@databasus.local"

		consoleRuns := []struct {
			arguments      []string
			expectedOutput string
		}{
			{
				arguments: []string{
					"--create-user",
					"--email=" + email,
					"--password=" + operatorPassword,
					"--role=admin",
				},
				expectedOutput: email,
			},
			{
				arguments:      []string{"--new-password=rotated-password-456", "--email=" + email},
				expectedOutput: "password reset successfully",
			},
			{arguments: []string{"--disable-2fa"}, expectedOutput: "Two-factor authentication"},
			{arguments: []string{"--list-admins"}, expectedOutput: email},
		}

		for _, consoleRun := range consoleRuns {
			binaryRun := runBinary(t, host, environment, consoleRun.arguments...)

			require.Equal(t, 0, binaryRun.exitCode, "%v: %s", consoleRun.arguments, binaryRun.output)
			assert.Contains(t, binaryRun.output, consoleRun.expectedOutput, consoleRun.arguments)
			assert.NotContains(t, binaryRun.output, "http server listening", consoleRun.arguments)
		}
	})

	t.Run("exits non-zero when the password reset names an unknown address", func(t *testing.T) {
		environment := newDatabaseEnvironment(testdb.Open(t))

		binaryRun := runBinary(t, newOperatorHost(t), environment,
			"--new-password=rotated-password-456", "--email=nobody@databasus.local")

		assert.NotEqual(t, 0, binaryRun.exitCode, binaryRun.output)
	})

	t.Run("refuses --create-user with --new-password and changes no account", func(t *testing.T) {
		host := newOperatorHost(t)
		databaseDsn := testdb.Open(t)
		environment := newDatabaseEnvironment(databaseDsn)
		email := "refused@databasus.local"

		createRun := runBinary(t, host, environment,
			"--create-user", "--email="+email, "--password="+operatorPassword, "--role=admin")
		require.Equal(t, 0, createRun.exitCode, createRun.output)

		accountsBeforeRefusal := listAccountRows(t, databaseDsn)

		binaryRun := runBinary(t, host, environment,
			"--create-user", "--email="+email, "--new-password=rotated-password-456")

		assert.NotEqual(t, 0, binaryRun.exitCode, binaryRun.output)
		assert.Contains(t, binaryRun.output, "--password")
		assert.Equal(t, accountsBeforeRefusal, listAccountRows(t, databaseDsn))
	})

	t.Run("refuses --new-password without --email before reading the connection string", func(t *testing.T) {
		binaryRun := runBinary(t, newOperatorHost(t), map[string]string{"ENV_MODE": "production"},
			"--new-password=rotated-password-456")

		assert.NotEqual(t, 0, binaryRun.exitCode, binaryRun.output)
		assert.Contains(t, binaryRun.output, "--email")
		assert.NotContains(t, binaryRun.output, "DATABASE_DSN")
	})

	t.Run("resets a password while the server runs on the same database", func(t *testing.T) {
		host := newOperatorHost(t)
		databaseDsn := testdb.Open(t)
		environment := newDatabaseEnvironment(databaseDsn)
		email := "running@databasus.local"
		serverAddress := startInProcessServer(t, databaseDsn)

		createRun := runBinary(t, host, environment,
			"--create-user", "--email="+email, "--password="+operatorPassword, "--role=admin")
		require.Equal(t, 0, createRun.exitCode, createRun.output)

		requireVersionAnswers(t, serverAddress)

		resetRun := runBinary(t, host, environment, "--new-password=rotated-password-456", "--email="+email)

		require.Equal(t, 0, resetRun.exitCode, resetRun.output)
		requireVersionAnswers(t, serverAddress)
	})

	t.Run("stops the server and a console command without changes when the schema is newer", func(t *testing.T) {
		host := newOperatorHost(t)
		databaseDsn := testdb.Open(t)
		environment := newDatabaseEnvironment(databaseDsn)
		email := "rolled-back@databasus.local"

		createRun := runBinary(t, host, environment,
			"--create-user", "--email="+email, "--password="+operatorPassword, "--role=admin")
		require.Equal(t, 0, createRun.exitCode, createRun.output)

		executeStatement(t, databaseDsn, fmt.Sprintf(
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES (%d, TRUE)", newerMigrationVersion))

		accountsBeforeRollback := listAccountRows(t, databaseDsn)
		versionsBeforeRollback := listMigrationVersions(t, databaseDsn)

		binaryRuns := map[string]binaryRun{
			"server":         runBinary(t, host, environment),
			"password reset": runBinary(t, host, environment, "--new-password=rotated-password-456", "--email="+email),
		}

		for runName, binaryRun := range binaryRuns {
			assert.NotEqual(t, 0, binaryRun.exitCode, "%s: %s", runName, binaryRun.output)
			assert.Contains(t, binaryRun.output, strconv.Itoa(newerMigrationVersion), runName)
			assert.Contains(t, binaryRun.output, "after an upgrade is not supported", runName)
			assert.NotContains(t, binaryRun.output, "http server listening", runName)
			assert.NotContains(t, binaryRun.output, "password reset successfully", runName)
		}

		assert.Equal(t, accountsBeforeRollback, listAccountRows(t, databaseDsn))
		assert.Equal(t, versionsBeforeRollback, listMigrationVersions(t, databaseDsn))
	})

	t.Run("stops on a failed migration without opening the port and applies it after the fix", func(t *testing.T) {
		host := newOperatorHost(t)
		databaseDsn := testdb.OpenEmpty(t)
		environment := newDatabaseEnvironment(databaseDsn)
		executeStatement(t, databaseDsn, "CREATE TABLE users (id INT)")

		serverRun := runBinary(t, host, environment)

		assert.NotEqual(t, 0, serverRun.exitCode, serverRun.output)
		assert.Contains(t, serverRun.output, firstMigration)
		assert.NotContains(t, serverRun.output, "http server listening")

		executeStatement(t, databaseDsn, "DROP TABLE users")

		listRun := runBinary(t, host, environment, "--list-admins")

		require.Equal(t, 0, listRun.exitCode, listRun.output)
	})
}

// operatorHost is a working directory whose data volume and .env belong to one test. The binary finds
// its root through the go.mod above its working directory, as in a checkout.
type operatorHost struct {
	workingDirectory string
	dataDirectory    string
}

func newOperatorHost(t *testing.T) operatorHost {
	t.Helper()

	rootDirectory := t.TempDir()
	workingDirectory := filepath.Join(rootDirectory, "backend")
	require.NoError(t, os.MkdirAll(workingDirectory, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, "go.mod"), []byte("module host\n"), 0o600))

	return operatorHost{
		workingDirectory: workingDirectory,
		dataDirectory:    filepath.Join(rootDirectory, "databasus-data"),
	}
}

type binaryRun struct {
	exitCode int
	output   string
}

// runBinary starts the binary with only the given variables, so nothing from the test process or the
// checkout's .env leaks into it.
func runBinary(t *testing.T, host operatorHost, variables map[string]string, arguments ...string) binaryRun {
	t.Helper()

	runCtx, cancelRun := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancelRun()

	command := exec.CommandContext(runCtx, binaryPath, arguments...)
	command.Dir = host.workingDirectory
	//nolint:forbidigo // the binary looks up nothing but PATH from the test process
	command.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + host.workingDirectory}

	for name, value := range variables {
		command.Env = append(command.Env, name+"="+value)
	}

	output, err := command.CombinedOutput()

	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return binaryRun{exitCode: exitError.ExitCode(), output: string(output)}
	}

	require.NoError(t, err, string(output))

	return binaryRun{exitCode: 0, output: string(output)}
}

func newDatabaseEnvironment(databaseDsn string) map[string]string {
	return map[string]string{
		"ENV_MODE":                       "production",
		"DATABASE_DSN":                   databaseDsn,
		"IS_DISABLE_ANONYMOUS_TELEMETRY": "true",
	}
}

// startInProcessServer runs the server in the test process on a free port: the binary always listens
// on :4005, which another agent's `just run` may hold on a shared machine.
func startInProcessServer(t *testing.T, databaseDsn string) string {
	t.Helper()

	t.Setenv("ENV_MODE", "production")
	t.Setenv("DATABASE_DSN", databaseDsn)
	t.Setenv("IS_DISABLE_ANONYMOUS_TELEMETRY", "true")

	rootDirectory := t.TempDir()

	environment, err := app.LoadEnvironment(t.Context(), config.LoadSpec{
		IsDatabaseRequired:       true,
		RootDirectory:            rootDirectory,
		PublishedDatabaseDsnPath: filepath.Join(rootDirectory, "never-published"),
	})
	require.NoError(t, err)

	application, err := app.New(t.Context(), environment)
	require.NoError(t, err)

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	address := listener.Addr().String()
	require.NoError(t, listener.Close())

	serveCtx, stopServing := context.WithCancel(t.Context())
	isServeFinished := make(chan struct{})

	go func() {
		defer close(isServeFinished)

		assert.NoError(t, application.Serve(serveCtx, address))
	}()

	t.Cleanup(func() {
		stopServing()
		<-isServeFinished
	})

	return address
}

// requireVersionAnswers waits until the server answers, since Serve opens its port in its own
// goroutine.
func requireVersionAnswers(t *testing.T, address string) {
	t.Helper()

	testwait.Until(t, "GET /api/v1/system/version to answer", func(ctx context.Context) error {
		return checkServerAlive(ctx, "http://"+address)
	})
}

func executeStatement(t *testing.T, databaseDsn, statement string) {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)

	require.NoError(t, db.WithContext(t.Context()).Exec(statement).Error)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

type accountRow struct {
	Email          string
	HashedPassword string
	Role           string
}

func listAccountRows(t *testing.T, databaseDsn string) []accountRow {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)

	defer func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	}()

	var accountRows []accountRow
	require.NoError(t, db.WithContext(t.Context()).
		Raw("SELECT email, hashed_password, role FROM users ORDER BY email").
		Scan(&accountRows).Error)

	return accountRows
}

func listMigrationVersions(t *testing.T, databaseDsn string) []int64 {
	t.Helper()

	db, err := database.Open(t.Context(), databaseDsn)
	require.NoError(t, err)

	defer func() {
		sqlDB, sqlErr := db.DB()
		if sqlErr == nil {
			_ = sqlDB.Close()
		}
	}()

	var migrationVersions []int64
	require.NoError(t, db.WithContext(t.Context()).
		Raw("SELECT version_id FROM goose_db_version ORDER BY id").
		Scan(&migrationVersions).Error)

	return migrationVersions
}
