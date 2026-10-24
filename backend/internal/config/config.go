// Package config hands the configuration internal/app loaded to the code that has not moved into
// internal/modules yet. A test binary built with the testhelpers tag loads it on first use instead.
package config

import (
	"sync"
	"sync/atomic"

	lib_config "databasus-backend/internal/lib/config"
)

const (
	AppModeWeb        = "web"
	AppModeBackground = "background"
)

type EnvVariables struct {
	lib_config.Config

	IsTesting       bool
	TestDatabaseDsn string `env:"TEST_DATABASE_DSN"`
	TestLocalhost   string `env:"TEST_LOCALHOST"`

	// TestLogicalPostgres16Port is the shared always-on logical-Postgres fixture used by
	// GetTestPostgresConfig / CreateTestDatabase across the test suite. The per-version PG
	// logical tests (14-18 + SSL + mTLS) run on testcontainers and need no fixed port.
	TestLogicalPostgres16Port string `env:"TEST_LOGICAL_POSTGRES_16_PORT"`

	// The physical primary sources are the shared always-on fixture (like the logical PG-16
	// fixture); the per-version, no-summary, tablespace, mTLS and restore-target containers all run
	// on testcontainers and need no fixed port.
	TestPhysicalPostgres17Port string `env:"TEST_PHYSICAL_POSTGRES_17_PORT"`
	TestPhysicalPostgres18Port string `env:"TEST_PHYSICAL_POSTGRES_18_PORT"`
}

var installedEnv atomic.Pointer[EnvVariables]

var getEnvWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(loadedConfig lib_config.Config) {
	installedEnv.Store(&EnvVariables{Config: loadedConfig})
}

func GetEnv() *EnvVariables {
	if env := installedEnv.Load(); env != nil {
		return env
	}

	return getEnvWithoutApp()
}
