package postgresql_logical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/sshtunnel"
)

func credentialSpecDatabase() *PostgresqlLogicalDatabase {
	databaseName := "testdb"

	return &PostgresqlLogicalDatabase{
		Host:     "db.internal",
		Port:     5432,
		Username: "testuser",
		Password: "testpassword",
		Database: &databaseName,
		CpuCount: 1,
	}
}

func Test_DecryptCredentialSpec_WithoutATunnel_LeavesHostAddrEmpty(t *testing.T) {
	spec, err := credentialSpecDatabase().DecryptCredentialSpec(nil)
	require.NoError(t, err)

	assert.Equal(t, "db.internal", spec.Host)
	assert.Empty(t, spec.HostAddr)
}

func Test_DecryptCredentialSpec_WithALocalTunnelEndpoint_KeepsTheRealHostAndCarriesTheAddress(t *testing.T) {
	database := credentialSpecDatabase()
	database.LocalTunnelEndpoint = &sshtunnel.Endpoint{Host: "127.0.0.1", Port: 41000}

	spec, err := database.DecryptCredentialSpec(nil)
	require.NoError(t, err)

	assert.Equal(t, "db.internal", spec.Host)
	assert.Equal(t, "127.0.0.1", spec.HostAddr)
}
