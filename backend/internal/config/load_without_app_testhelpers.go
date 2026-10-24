//go:build testhelpers

package config

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
	_ "github.com/jackc/pgx/v5/stdlib"

	lib_config "databasus-backend/internal/lib/config"
	"databasus-backend/internal/util/logger"
)

const (
	// defaultTestParallelWorkers must equal the `go test -p` value so every running package can
	// claim an isolated metadata database.
	defaultTestParallelWorkers = 8

	// testSlotAdvisoryLockBase is the first pg_advisory_lock key reserved for
	// test-worker slot claims; slot N uses base+N.
	testSlotAdvisoryLockBase = 945_000_000

	// testSlotClaimTimeout bounds the wait for a free slot, absorbing the brief
	// window where `go test -p` starts the next package before the previous
	// process has exited and released its advisory lock.
	testSlotClaimTimeout = 60 * time.Second

	// systemDbName is the always-present database used for slot coordination and as
	// the connection target when (re)creating per-slot databases.
	systemDbName = "postgres"
)

// slotLockConn holds the system-DB connection whose session owns this worker's
// advisory lock. It must live for the whole process: closing it (or letting it
// be garbage-collected) releases the lock and frees the slot for another worker
// mid-run.
//
//nolint:unused // assigned-only: keeps the advisory-lock connection alive for the process lifetime
var slotLockConn *sql.Conn

// loadWithoutApp loads the test configuration: the test database instead of the working one and
// the ports of the shared fixtures. Only a go test binary claims a per-worker slot; cleanup_test_db
// works on every slot, so it keeps the base test DSN.
func loadWithoutApp() *EnvVariables {
	spec := lib_config.LoadSpec{IsDatabaseRequired: false, RootDirectory: "", PublishedDatabaseDsnPath: ""}

	loadedConfig, _, err := lib_config.Load(spec)
	if err != nil {
		panic(fmt.Sprintf("load the test configuration: %v", err))
	}

	variables, _, err := lib_config.ReadVariables(spec)
	if err != nil {
		panic(fmt.Sprintf("read the test settings: %v", err))
	}

	testEnv := EnvVariables{Config: loadedConfig, IsTesting: true}

	err = env.ParseWithOptions(&testEnv, env.Options{Environment: variables})
	if err != nil {
		panic(fmt.Sprintf("parse the test settings: %v", err))
	}

	requireTestSettings(&testEnv)

	testEnv.DatabaseDsn = testEnv.TestDatabaseDsn

	if testEnv.TestParallelWorkers <= 0 {
		testEnv.TestParallelWorkers = defaultTestParallelWorkers
	}

	if testEnv.TestLocalhost == "" {
		testEnv.TestLocalhost = "localhost"
	}

	if testing.Testing() {
		testEnv.DatabaseDsn = claimTestWorkerSlotAndSelectMetadataDatabase(&testEnv)
	}

	return &testEnv
}

func requireTestSettings(testEnv *EnvVariables) {
	requiredSettings := map[string]string{
		"TEST_DATABASE_DSN":              testEnv.TestDatabaseDsn,
		"TEST_LOGICAL_POSTGRES_16_PORT":  testEnv.TestLogicalPostgres16Port,
		"TEST_PHYSICAL_POSTGRES_17_PORT": testEnv.TestPhysicalPostgres17Port,
		"TEST_PHYSICAL_POSTGRES_18_PORT": testEnv.TestPhysicalPostgres18Port,
	}

	for name, value := range requiredSettings {
		if value == "" {
			panic(name + " is empty")
		}
	}
}

func claimTestWorkerSlotAndSelectMetadataDatabase(testEnv *EnvVariables) string {
	baseDbName, _, err := RewriteDbName(testEnv.TestDatabaseDsn, systemDbName)
	if err != nil {
		panic(fmt.Sprintf("parse TEST_DATABASE_DSN for slot isolation: %v", err))
	}

	slot := claimTestWorkerSlot(testEnv.TestDatabaseDsn, testEnv.TestParallelWorkers)

	slotDbName := fmt.Sprintf("%s_w%d", baseDbName, slot)

	_, slotDsn, err := RewriteDbName(testEnv.TestDatabaseDsn, slotDbName)
	if err != nil {
		panic(fmt.Sprintf("build the per-slot DSN: %v", err))
	}

	logger.GetLogger().Info("claimed test worker slot", "slot", slot, "db", slotDbName)

	return slotDsn
}

// claimTestWorkerSlot acquires a session-level advisory lock for the first free
// slot in [0,pool) on the system Postgres DB and returns it. The lock is held by
// slotLockConn until the process exits. It retries until testSlotClaimTimeout to
// absorb the handoff window where `go test -p` overlaps a finishing worker.
func claimTestWorkerSlot(testDsn string, pool int) int {
	_, systemDsn, err := RewriteDbName(testDsn, systemDbName)
	if err != nil {
		panic(fmt.Sprintf("build the system DSN for the slot claim: %v", err))
	}

	db, err := sql.Open("pgx", systemDsn)
	if err != nil {
		panic(fmt.Sprintf("open the system DB for the slot claim: %v", err))
	}

	ctx := context.Background()

	conn, err := db.Conn(ctx)
	if err != nil {
		panic(fmt.Sprintf("connect to the system DB for the slot claim: %v", err))
	}

	deadline := time.Now().Add(testSlotClaimTimeout)
	for {
		for slot := range pool {
			var isLocked bool

			lockErr := conn.QueryRowContext(
				ctx,
				"SELECT pg_try_advisory_lock($1)",
				testSlotAdvisoryLockBase+int64(slot),
			).Scan(&isLocked)
			if lockErr != nil {
				panic(fmt.Sprintf("advisory lock query failed during the slot claim: %v", lockErr))
			}

			if isLocked {
				slotLockConn = conn
				return slot
			}
		}

		if time.Now().After(deadline) {
			panic(fmt.Sprintf(
				"no free test DB slot among %d; TEST_PARALLEL_WORKERS must be >= the `go test -p` value", pool,
			))
		}

		time.Sleep(100 * time.Millisecond)
	}
}
