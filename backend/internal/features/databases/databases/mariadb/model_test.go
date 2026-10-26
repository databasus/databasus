package mariadb

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/util/testing/containers"
)

// The engine tests in internal/engines/mariadb cover the checks themselves; this one shows the model
// reaches its engine and records what the engine detected.
func Test_TestConnection_WithBackupGrants_RecordsVersionAndPrivileges(t *testing.T) {
	endpoint := containers.StartMariadb(t, "mariadb:10.11")
	databaseName := containers.MariadbDatabase

	mariadbModel := &MariadbDatabase{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: "root",
		Password: containers.MariadbRootPassword,
		Database: &databaseName,
		IsHttps:  false,
	}

	err := mariadbModel.TestConnection(t.Context(), slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)

	require.NoError(t, err)
	assert.Equal(t, mariadb_engine.Version1011, mariadbModel.Version)
	assert.True(t, mariadbModel.HasPrivilege("SELECT"))
	assert.True(t, mariadbModel.HasPrivilege("SHOW VIEW"))
}

func Test_TestConnection_WithoutDatabaseName_ReturnsError(t *testing.T) {
	mariadbModel := &MariadbDatabase{Host: "db.example.com", Port: 3306, Username: "root", Password: "secret"}

	err := mariadbModel.TestConnection(t.Context(), slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)

	assert.ErrorIs(t, err, mariadb_engine.ErrDatabaseNameRequired)
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	databaseName := "appdb"
	mariadbModel := &MariadbDatabase{
		Version:         mariadb_engine.Version114,
		Host:            "db.example.com",
		Port:            3306,
		Username:        "appuser",
		Password:        "supersecret",
		Database:        &databaseName,
		IsHttps:         true,
		IsExcludeEvents: true,
		ExcludeTables:   []string{"audit_logs"},
		Privileges:      "SELECT, INSERT",
	}

	mariadbModel.HideSensitiveData()

	assert.Empty(t, mariadbModel.Password)
	assert.Equal(t, "db.example.com", mariadbModel.Host)
	assert.Equal(t, 3306, mariadbModel.Port)
	assert.Equal(t, "appuser", mariadbModel.Username)
	assert.Equal(t, &databaseName, mariadbModel.Database)
	assert.True(t, mariadbModel.IsHttps)
	assert.True(t, mariadbModel.IsExcludeEvents)
	assert.Equal(t, []string{"audit_logs"}, mariadbModel.ExcludeTables)
	assert.Equal(t, "SELECT, INSERT", mariadbModel.Privileges)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var mariadbModel *MariadbDatabase

	assert.NotPanics(t, func() {
		mariadbModel.HideSensitiveData()
	})
}
