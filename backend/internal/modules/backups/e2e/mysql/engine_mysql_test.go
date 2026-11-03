package mysql

import (
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func getMysqlVersions() []struct {
	name    string
	version apiclient.MysqlVersion
	image   string
} {
	return []struct {
		name    string
		version apiclient.MysqlVersion
		image   string
	}{
		{"MySQL 5.7", apiclient.MysqlVersionN57, "mysql:5.7"},
		{"MySQL 8.0", apiclient.MysqlVersionN80, "mysql:8.0"},
		{"MySQL 8.4", apiclient.MysqlVersionN84, "mysql:8.4"},
		{"MySQL 9", apiclient.MysqlVersionN9, "mysql:9"},
		{"MySQL 26", apiclient.MysqlVersionN26, "mysql:26.7"},
	}
}

// TestBackupAndRestoreMysql boots each supported version once and runs every scenario against it,
// so only one server of the matrix is alive at a time.
//
//nolint:tparallel // Each version's scenarios share one server, and a version stops before the next starts.
func TestBackupAndRestoreMysql(t *testing.T) {
	t.Parallel()

	for _, mysqlVersion := range getMysqlVersions() {
		t.Run(mysqlVersion.name, func(t *testing.T) {
			endpoint := containers.StartMysql(t, mysqlVersion.image)

			t.Run("restores the rows", func(t *testing.T) {
				testMysqlRowsRestored(t, endpoint, mysqlVersion.version, apiclient.BackupEncryptionNONE)
			})
			t.Run("restores the rows of an encrypted backup", func(t *testing.T) {
				testMysqlRowsRestored(t, endpoint, mysqlVersion.version, apiclient.BackupEncryptionENCRYPTED)
			})
			t.Run("leaves out an excluded table", func(t *testing.T) {
				testMysqlExcludedTableLeftOut(t, endpoint, mysqlVersion.version)
			})
			t.Run("restores the rows of a backup taken by a limited user", func(t *testing.T) {
				testMysqlLimitedUserBackupRestored(t, endpoint, mysqlVersion.version)
			})
		})
	}
}

func TestBackupAndRestoreMysqlOverTls(t *testing.T) {
	t.Parallel()

	t.Run("restores the rows over a connection that requires TLS", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, enginetest.RegisterSkipVerifyMysqlTLSConfig())

		endpoint := containers.StartMysqlSSL(t, "mysql:8.4")
		server := enginetest.ConnectToMysqlFamily(
			t,
			endpoint,
			containers.MysqlRootPassword,
			"tls="+enginetest.SkipVerifyMysqlTLSConfigName,
		)
		server.CreateTestDataTable(t)

		settings := newMysqlSettings(server, apiclient.MysqlVersionN84)
		settings.IsHttps = true

		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mysql_ssl")
		target := newMysqlTarget(server, restoredDatabaseName)
		target.IsHttps = true
		tm.Restore(t, backup.Id, toMysqlRestoreTarget(t, target))

		enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
	})
}

func TestBackupAndRestoreMysqlOverSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run("restores the rows through the bastion", func(t *testing.T) {
		t.Parallel()

		topology := containers.StartMysqlBehindSSHBastion(t, "mysql:8.4")
		server := enginetest.ConnectToMysqlFamily(
			t,
			enginetest.OpenForwardedEndpoint(t, topology),
			containers.MysqlRootPassword,
			"",
		)
		server.CreateTestDataTable(t)

		bastionServer := enginetest.MysqlFamilyServer{
			Endpoint: topology.Database, RootPassword: containers.MysqlRootPassword,
			DSNOptions: "", Admin: nil,
		}
		settings := newMysqlSettings(bastionServer, apiclient.MysqlVersionN84)
		settings.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)

		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mysql_ssh_tunnel")
		target := newMysqlTarget(bastionServer, restoredDatabaseName)
		target.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)
		tm.Restore(t, backup.Id, toMysqlRestoreTarget(t, target))

		enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
	})
}

func TestBackupMysqlToFailingStorage(t *testing.T) {
	t.Parallel()

	t.Run("fails the backup without hanging when the upload fails", func(t *testing.T) {
		t.Parallel()

		server := enginetest.ConnectToMysqlFamily(
			t,
			containers.StartMysql(t, "mysql:8.0"),
			containers.MysqlRootPassword,
			"",
		)
		server.CreateTestDataTable(t)

		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, newMysqlSettings(server, apiclient.MysqlVersionN80))

		tm.RequireBackupFailsWithoutHanging(t, database.Id)
	})
}

func TestStartMysqlRestoreVersionCheck(t *testing.T) {
	t.Parallel()

	server := enginetest.ConnectToMysqlFamily(
		t,
		containers.StartMysql(t, "mysql:8.0"),
		containers.MysqlRootPassword,
		"",
	)
	server.CreateTestDataTable(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, newMysqlSettings(server, apiclient.MysqlVersionN80))
	backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
	restoredDatabaseName, _ := server.CreateRestoreDatabase(t, "restored_mysql_version")

	tm.RequireRestoreVersionCheck(t, enginetest.RestoreVersionCheck{
		DatabaseID:    database.Id,
		BackupID:      backup.Id,
		SettingsTable: "databases_mysql",
		Target:        toMysqlRestoreTarget(t, newMysqlTarget(server, restoredDatabaseName)),
		TargetVersion: "8.0",
	})
}

func testMysqlRowsRestored(
	t *testing.T,
	endpoint containers.Endpoint,
	version apiclient.MysqlVersion,
	encryption apiclient.BackupEncryption,
) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MysqlRootPassword, "")
	server.CreateTestDataTable(t)

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, newMysqlSettings(server, version)).Id, encryption)

	assert.Equal(t, encryption, backup.Encryption)
	tm.RequireBackupSizeMatchesDownload(t, backup)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mysql")
	tm.Restore(t, backup.Id, toMysqlRestoreTarget(t, newMysqlTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
}

func testMysqlExcludedTableLeftOut(t *testing.T, endpoint containers.Endpoint, version apiclient.MysqlVersion) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MysqlRootPassword, "")
	server.CreateTestDataTable(t)
	server.CreateExtraTable(t)

	settings := newMysqlSettings(server, version)
	settings.ExcludeTables = []string{"\nextra_table"}

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mysql_excl_tables")
	tm.Restore(t, backup.Id, toMysqlRestoreTarget(t, newMysqlTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	assert.Equal(t, 0, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "extra_table"),
		"the excluded extra_table is left out")
}

func testMysqlLimitedUserBackupRestored(t *testing.T, endpoint containers.Endpoint, version apiclient.MysqlVersion) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MysqlRootPassword, "")
	server.CreateTestDataTable(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, newMysqlSettings(server, version))

	limitedUser := tm.CreateLimitedUser(t, database)
	assert.NotEmpty(t, limitedUser.Username)
	assert.NotEmpty(t, limitedUser.Password)

	settings, err := database.Settings.AsMysqlSettings()
	require.NoError(t, err)

	settings.Username = limitedUser.Username
	settings.Password = &limitedUser.Password

	limitedDatabase := tm.UpdateDatabaseSettings(t, database, settings)
	backup := tm.BackUp(t, limitedDatabase.Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mysql_readonly")
	tm.Restore(t, backup.Id, toMysqlRestoreTarget(t, newMysqlTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
}

func newMysqlSettings(server enginetest.MysqlFamilyServer, version apiclient.MysqlVersion) apiclient.MysqlSettings {
	password := server.RootPassword
	database := containers.MysqlDatabase

	return apiclient.MysqlSettings{
		Type:          apiclient.MysqlSettingsTypeMYSQL,
		Host:          server.Endpoint.Host,
		Port:          server.Endpoint.Port,
		Username:      "root",
		Password:      &password,
		Database:      &database,
		Version:       &version,
		IsHttps:       false,
		ExcludeTables: []string{},
		Privileges:    nil,
		SshTunnel:     enginetest.NewDisabledSSHTunnelSettings(),
	}
}

func newMysqlTarget(server enginetest.MysqlFamilyServer, databaseName string) apiclient.MysqlRestoreTarget {
	password := server.RootPassword

	return apiclient.MysqlRestoreTarget{
		Type:      apiclient.MysqlRestoreTargetTypeMYSQL,
		Host:      server.Endpoint.Host,
		Port:      server.Endpoint.Port,
		Username:  "root",
		Password:  &password,
		Database:  databaseName,
		IsHttps:   false,
		SshTunnel: enginetest.NewDisabledSSHTunnelSettings(),
	}
}

func toMysqlRestoreTarget(t *testing.T, target apiclient.MysqlRestoreTarget) apiclient.RestoreTarget {
	t.Helper()

	var restoreTarget apiclient.RestoreTarget
	require.NoError(t, restoreTarget.FromMysqlRestoreTarget(target))

	return restoreTarget
}
