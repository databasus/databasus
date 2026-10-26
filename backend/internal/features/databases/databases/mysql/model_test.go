package mysql

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mysql_engine "databasus-backend/internal/engines/mysql"
	"databasus-backend/internal/util/testing/containers"
)

// The engine tests in internal/engines/mysql cover the checks themselves; this one shows the model
// reaches its engine and records what the engine detected.
func Test_TestConnection_WithBackupGrants_RecordsVersionAndPrivileges(t *testing.T) {
	endpoint := containers.StartMysql(t, "mysql:8.0")
	databaseName := containers.MysqlDatabase

	mysqlModel := &MysqlDatabase{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: "root",
		Password: containers.MysqlRootPassword,
		Database: &databaseName,
		IsHttps:  false,
	}

	err := mysqlModel.TestConnection(t.Context(), slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)

	require.NoError(t, err)
	assert.Equal(t, mysql_engine.Version80, mysqlModel.Version)
	assert.True(t, mysqlModel.HasPrivilege("SELECT"))
	assert.True(t, mysqlModel.HasPrivilege("SHOW VIEW"))
}

func Test_TestConnection_WithoutDatabaseName_ReturnsError(t *testing.T) {
	mysqlModel := &MysqlDatabase{Host: "db.example.com", Port: 3306, Username: "root", Password: "secret"}

	err := mysqlModel.TestConnection(t.Context(), slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)

	assert.ErrorIs(t, err, mysql_engine.ErrDatabaseNameRequired)
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	databaseName := "appdb"
	mysqlModel := &MysqlDatabase{
		Version:       mysql_engine.Version80,
		Host:          "db.example.com",
		Port:          3306,
		Username:      "appuser",
		Password:      "supersecret",
		Database:      &databaseName,
		IsHttps:       true,
		ExcludeTables: []string{"audit_logs"},
		Privileges:    "SELECT, INSERT",
	}

	mysqlModel.HideSensitiveData()

	assert.Empty(t, mysqlModel.Password)
	assert.Equal(t, "db.example.com", mysqlModel.Host)
	assert.Equal(t, 3306, mysqlModel.Port)
	assert.Equal(t, "appuser", mysqlModel.Username)
	assert.Equal(t, &databaseName, mysqlModel.Database)
	assert.True(t, mysqlModel.IsHttps)
	assert.Equal(t, []string{"audit_logs"}, mysqlModel.ExcludeTables)
	assert.Equal(t, "SELECT, INSERT", mysqlModel.Privileges)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var mysqlModel *MysqlDatabase

	assert.NotPanics(t, func() {
		mysqlModel.HideSensitiveData()
	})
}
