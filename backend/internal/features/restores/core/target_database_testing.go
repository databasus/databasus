//go:build testhelpers

package restores_core

import (
	"fmt"
	"strconv"

	"databasus-backend/internal/config"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// GetTestPostgresTarget is a restore target on the shared PostgreSQL 16 test server.
func GetTestPostgresTarget() *PostgresqlTargetDatabase {
	env := config.GetEnv()

	port, err := strconv.Atoi(env.TestLogicalPostgres16Port)
	if err != nil {
		panic(fmt.Sprintf("failed to parse TEST_LOGICAL_POSTGRES_16_PORT: %v", err))
	}

	databaseName := "testdb"

	return &PostgresqlTargetDatabase{
		Version:  pgshared.Version16,
		Host:     env.TestLocalhost,
		Port:     port,
		Username: "testuser",
		Password: "testpassword",
		Database: &databaseName,
		CpuCount: 1,
	}
}
