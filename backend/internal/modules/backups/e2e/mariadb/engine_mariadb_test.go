package mariadb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func getMariadbVersions() []struct {
	name                string
	version             apiclient.MariadbVersion
	image               string
	isRunningExclusions bool
} {
	return []struct {
		name                string
		version             apiclient.MariadbVersion
		image               string
		isRunningExclusions bool
	}{
		{"MariaDB 5.5", apiclient.MariadbVersionN55, "mariadb:5.5", false},
		{"MariaDB 10.1", apiclient.MariadbVersionN101, "mariadb:10.1", false},
		{"MariaDB 10.6", apiclient.MariadbVersionN106, "mariadb:10.6", false},
		{"MariaDB 10.11", apiclient.MariadbVersionN1011, "mariadb:10.11", true},
		{"MariaDB 11.4", apiclient.MariadbVersionN114, "mariadb:11.4", true},
		{"MariaDB 11.8", apiclient.MariadbVersionN118, "mariadb:11.8", false},
		{"MariaDB 12.0", apiclient.MariadbVersionN120, "mariadb:12.0", false},
		{"MariaDB 13.0", apiclient.MariadbVersionN130, "mariadb:13.0", true},
	}
}

// TestBackupAndRestoreMariadb boots each supported version once and runs every scenario against it,
// so only one server of the matrix is alive at a time.
//
//nolint:tparallel // Each version's scenarios share one server, and a version stops before the next starts.
func TestBackupAndRestoreMariadb(t *testing.T) {
	t.Parallel()

	for _, mariadbVersion := range getMariadbVersions() {
		t.Run(mariadbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMariadb(t, mariadbVersion.image)

			t.Run("restores the rows", func(t *testing.T) {
				testMariadbRowsRestored(t, endpoint, mariadbVersion.version, apiclient.BackupEncryptionNONE)
			})
			t.Run("restores the rows of an encrypted backup", func(t *testing.T) {
				testMariadbRowsRestored(t, endpoint, mariadbVersion.version, apiclient.BackupEncryptionENCRYPTED)
			})
			t.Run("restores the rows of a backup taken by a limited user", func(t *testing.T) {
				testMariadbLimitedUserBackupRestored(t, endpoint, mariadbVersion.version)
			})

			if !mariadbVersion.isRunningExclusions {
				return
			}

			t.Run("leaves out an excluded table", func(t *testing.T) {
				testMariadbExcludedTableLeftOut(t, endpoint, mariadbVersion.version)
			})
			t.Run("leaves out the events when the dump excludes them", func(t *testing.T) {
				testMariadbExcludedEventsLeftOut(t, endpoint, mariadbVersion.version)
			})
		})
	}
}

func TestBackupAndRestoreMariadbOverTls(t *testing.T) {
	t.Parallel()

	t.Run("restores the rows over a connection that requires TLS", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, enginetest.RegisterSkipVerifyMysqlTLSConfig())

		endpoint := containers.StartMariadbSSL(t, "mariadb:11.8")
		server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MariadbRootPassword,
			"tls="+enginetest.SkipVerifyMysqlTLSConfigName)
		server.CreateTestDataTable(t)

		settings := newMariadbSettings(server, apiclient.MariadbVersionN118)
		settings.IsHttps = true

		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mariadb_ssl")
		target := newMariadbTarget(server, restoredDatabaseName)
		target.IsHttps = true
		tm.Restore(t, backup.Id, toMariadbRestoreTarget(t, target))

		enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
	})
}

func TestBackupAndRestoreMariadbOverSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run("restores the rows through the bastion", func(t *testing.T) {
		t.Parallel()

		topology := containers.StartMariadbBehindSSHBastion(t, "mariadb:11.4")
		server := enginetest.ConnectToMysqlFamily(
			t,
			enginetest.OpenForwardedEndpoint(t, topology),
			containers.MariadbRootPassword,
			"",
		)
		server.CreateTestDataTable(t)

		bastionServer := enginetest.MysqlFamilyServer{
			Endpoint: topology.Database, RootPassword: containers.MariadbRootPassword,
			DSNOptions: "", Admin: nil,
		}
		settings := newMariadbSettings(bastionServer, apiclient.MariadbVersionN114)
		settings.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)

		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mariadb_ssh_tunnel")
		target := newMariadbTarget(bastionServer, restoredDatabaseName)
		target.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)
		tm.Restore(t, backup.Id, toMariadbRestoreTarget(t, target))

		enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
	})
}

func TestBackupMariadbToFailingStorage(t *testing.T) {
	t.Parallel()

	t.Run("fails the backup without hanging when the upload fails", func(t *testing.T) {
		t.Parallel()

		server := enginetest.ConnectToMysqlFamily(
			t,
			containers.StartMariadb(t, "mariadb:10.11"),
			containers.MariadbRootPassword,
			"",
		)
		server.CreateTestDataTable(t)

		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, newMariadbSettings(server, apiclient.MariadbVersionN1011))

		tm.RequireBackupFailsWithoutHanging(t, database.Id)
	})
}

func TestStartMariadbRestoreVersionCheck(t *testing.T) {
	t.Parallel()

	server := enginetest.ConnectToMysqlFamily(
		t,
		containers.StartMariadb(t, "mariadb:10.11"),
		containers.MariadbRootPassword,
		"",
	)
	server.CreateTestDataTable(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, newMariadbSettings(server, apiclient.MariadbVersionN1011))
	backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
	restoredDatabaseName, _ := server.CreateRestoreDatabase(t, "restored_mariadb_version")

	tm.RequireRestoreVersionCheck(t, enginetest.RestoreVersionCheck{
		DatabaseID:    database.Id,
		BackupID:      backup.Id,
		SettingsTable: "databases_mariadb",
		Target:        toMariadbRestoreTarget(t, newMariadbTarget(server, restoredDatabaseName)),
		TargetVersion: "10.11",
	})
}

func testMariadbRowsRestored(
	t *testing.T,
	endpoint containers.Endpoint,
	version apiclient.MariadbVersion,
	encryption apiclient.BackupEncryption,
) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MariadbRootPassword, "")
	server.CreateTestDataTable(t)

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, newMariadbSettings(server, version)).Id, encryption)

	assert.Equal(t, encryption, backup.Encryption)
	tm.RequireBackupSizeMatchesDownload(t, backup)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mariadb")
	tm.Restore(t, backup.Id, toMariadbRestoreTarget(t, newMariadbTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
}

func testMariadbLimitedUserBackupRestored(
	t *testing.T,
	endpoint containers.Endpoint,
	version apiclient.MariadbVersion,
) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MariadbRootPassword, "")
	server.CreateTestDataTable(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, newMariadbSettings(server, version))

	limitedUser := tm.CreateLimitedUser(t, database)
	assert.NotEmpty(t, limitedUser.Username)
	assert.NotEmpty(t, limitedUser.Password)

	settings, err := database.Settings.AsMariadbSettings()
	require.NoError(t, err)

	settings.Username = limitedUser.Username
	settings.Password = &limitedUser.Password

	limitedDatabase := tm.UpdateDatabaseSettings(t, database, settings)
	backup := tm.BackUp(t, limitedDatabase.Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mariadb_readonly")
	tm.Restore(t, backup.Id, toMariadbRestoreTarget(t, newMariadbTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)
}

func testMariadbExcludedTableLeftOut(t *testing.T, endpoint containers.Endpoint, version apiclient.MariadbVersion) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MariadbRootPassword, "")
	server.CreateTestDataTable(t)
	server.CreateExtraTable(t)

	settings := newMariadbSettings(server, version)
	settings.ExcludeTables = []string{"\nextra_table"}

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mariadb_excl_tables")
	tm.Restore(t, backup.Id, toMariadbRestoreTarget(t, newMariadbTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	assert.Equal(t, 0, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "extra_table"),
		"the excluded extra_table is left out")
}

func testMariadbExcludedEventsLeftOut(t *testing.T, endpoint containers.Endpoint, version apiclient.MariadbVersion) {
	t.Helper()

	server := enginetest.ConnectToMysqlFamily(t, endpoint, containers.MariadbRootPassword, "")
	server.CreateTestDataTable(t)
	server.Execute(t, `
		CREATE EVENT IF NOT EXISTS test_event
		ON SCHEDULE EVERY 1 DAY
		DO BEGIN
			INSERT INTO test_data (name, value) VALUES ('event_test', 999);
		END`)
	server.CleanUp(t, "DROP EVENT IF EXISTS test_event")

	settings := newMariadbSettings(server, version)
	settings.IsExcludeEvents = true

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName, restoredDB := server.CreateRestoreDatabase(t, "restored_mariadb_no_events")
	tm.Restore(t, backup.Id, toMariadbRestoreTarget(t, newMariadbTarget(server, restoredDatabaseName)))

	assert.Equal(t, 1, enginetest.CountMysqlFamilyTables(t, restoredDB, restoredDatabaseName, "test_data"),
		"test_data is restored")
	enginetest.VerifyMysqlFamilyRows(t, server.Admin, restoredDB)

	var eventCount int
	require.NoError(t, restoredDB.GetContext(t.Context(), &eventCount,
		"SELECT COUNT(*) FROM information_schema.events WHERE event_schema = ? AND event_name = 'test_event'",
		restoredDatabaseName))
	assert.Equal(t, 0, eventCount, "test_event is left out when the dump excludes events")
}

func newMariadbSettings(
	server enginetest.MysqlFamilyServer,
	version apiclient.MariadbVersion,
) apiclient.MariadbSettings {
	password := server.RootPassword
	database := containers.MariadbDatabase

	return apiclient.MariadbSettings{
		Type:                apiclient.MariadbSettingsTypeMARIADB,
		Host:                server.Endpoint.Host,
		Port:                server.Endpoint.Port,
		Username:            "root",
		Password:            &password,
		Database:            &database,
		Version:             &version,
		IsHttps:             false,
		IsExcludeEvents:     false,
		IsSkipGaleraDisable: false,
		ExcludeTables:       []string{},
		Privileges:          nil,
		SshTunnel:           enginetest.NewDisabledSSHTunnelSettings(),
	}
}

func newMariadbTarget(server enginetest.MysqlFamilyServer, databaseName string) apiclient.MariadbRestoreTarget {
	password := server.RootPassword

	return apiclient.MariadbRestoreTarget{
		Type:                apiclient.MariadbRestoreTargetTypeMARIADB,
		Host:                server.Endpoint.Host,
		Port:                server.Endpoint.Port,
		Username:            "root",
		Password:            &password,
		Database:            databaseName,
		IsHttps:             false,
		IsSkipGaleraDisable: false,
		SshTunnel:           enginetest.NewDisabledSSHTunnelSettings(),
	}
}

func toMariadbRestoreTarget(t *testing.T, target apiclient.MariadbRestoreTarget) apiclient.RestoreTarget {
	t.Helper()

	var restoreTarget apiclient.RestoreTarget
	require.NoError(t, restoreTarget.FromMariadbRestoreTarget(target))

	return restoreTarget
}
