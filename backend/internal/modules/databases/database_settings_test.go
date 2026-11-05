package databases_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func toSettings(t *testing.T, convert func(settings *apiclient.DatabaseSettings) error) apiclient.DatabaseSettings {
	t.Helper()

	var settings apiclient.DatabaseSettings
	require.NoError(t, convert(&settings))

	return settings
}

func newDisabledSSHTunnel() apiclient.SshTunnelSettings {
	return apiclient.SshTunnelSettings{
		IsEnabled:            false,
		AuthType:             nil,
		Host:                 nil,
		Port:                 nil,
		Username:             nil,
		Password:             nil,
		PrivateKey:           nil,
		PrivateKeyPassphrase: nil,
	}
}

// newLogicalSettings points at the test database of a server from containers.StartPostgres.
func newLogicalSettings(server containers.Endpoint) apiclient.PostgresqlLogicalSettings {
	return apiclient.PostgresqlLogicalSettings{
		Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
		Host:               server.Host,
		Port:               server.Port,
		Username:           containers.PostgresUsername,
		Password:           new(containers.PostgresPassword),
		Database:           new(containers.PostgresDatabase),
		SslMode:            apiclient.PostgresqlSslModeDisable,
		SslClientCert:      nil,
		SslClientKey:       nil,
		SslRootCert:        nil,
		SshTunnel:          newDisabledSSHTunnel(),
		IncludeSchemas:     []string{},
		ExcludeTables:      []string{},
		ExcludeTableData:   []string{},
		CpuCount:           1,
		IsSkipUserMappings: false,
		Version:            nil,
	}
}

func fromLogicalSettings(t *testing.T, logical apiclient.PostgresqlLogicalSettings) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromPostgresqlLogicalSettings(logical)
	})
}

func getLogicalSettings(t *testing.T, database apiclient.Database) apiclient.PostgresqlLogicalSettings {
	t.Helper()

	logical, err := database.Settings.AsPostgresqlLogicalSettings()
	require.NoError(t, err)

	return logical
}

// newPhysicalSettings points at a server from containers.StartPhysicalPostgres.
func newPhysicalSettings(
	server containers.Endpoint,
	backupType apiclient.PhysicalBackupType,
) apiclient.PostgresqlPhysicalSettings {
	return apiclient.PostgresqlPhysicalSettings{
		Type:                apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Host:                server.Host,
		Port:                server.Port,
		Username:            containers.PostgresUsername,
		Password:            new(containers.PostgresPassword),
		SslMode:             apiclient.PostgresqlSslModeDisable,
		SslClientCert:       nil,
		SslClientKey:        nil,
		SslRootCert:         nil,
		SshTunnel:           newDisabledSSHTunnel(),
		BackupType:          backupType,
		SystemIdentifier:    nil,
		Version:             nil,
		WalSegmentSizeBytes: nil,
	}
}

func fromPhysicalSettings(t *testing.T, physical apiclient.PostgresqlPhysicalSettings) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromPostgresqlPhysicalSettings(physical)
	})
}

func getPhysicalSettings(t *testing.T, database apiclient.Database) apiclient.PostgresqlPhysicalSettings {
	t.Helper()

	physical, err := database.Settings.AsPostgresqlPhysicalSettings()
	require.NoError(t, err)

	return physical
}

// newMysqlSettings points at the test database of a server from containers.StartMysql.
func newMysqlSettings(server containers.Endpoint) apiclient.MysqlSettings {
	return apiclient.MysqlSettings{
		Type:          apiclient.MysqlSettingsTypeMYSQL,
		Host:          server.Host,
		Port:          server.Port,
		Username:      containers.MysqlUsername,
		Password:      new(containers.MysqlPassword),
		Database:      new(containers.MysqlDatabase),
		IsHttps:       false,
		ExcludeTables: []string{},
		Privileges:    nil,
		SshTunnel:     newDisabledSSHTunnel(),
		Version:       nil,
	}
}

func fromMysqlSettings(t *testing.T, mysql apiclient.MysqlSettings) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromMysqlSettings(mysql)
	})
}

func getMysqlSettings(t *testing.T, database apiclient.Database) apiclient.MysqlSettings {
	t.Helper()

	mysql, err := database.Settings.AsMysqlSettings()
	require.NoError(t, err)

	return mysql
}

// newMariadbSettings points at the test database of a server from containers.StartMariadb.
func newMariadbSettings(server containers.Endpoint) apiclient.MariadbSettings {
	return apiclient.MariadbSettings{
		Type:                apiclient.MariadbSettingsTypeMARIADB,
		Host:                server.Host,
		Port:                server.Port,
		Username:            containers.MariadbUsername,
		Password:            new(containers.MariadbPassword),
		Database:            new(containers.MariadbDatabase),
		IsHttps:             false,
		IsExcludeEvents:     false,
		IsSkipGaleraDisable: false,
		ExcludeTables:       []string{},
		Privileges:          nil,
		SshTunnel:           newDisabledSSHTunnel(),
		Version:             nil,
	}
}

func fromMariadbSettings(t *testing.T, mariadb apiclient.MariadbSettings) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromMariadbSettings(mariadb)
	})
}

func getMariadbSettings(t *testing.T, database apiclient.Database) apiclient.MariadbSettings {
	t.Helper()

	mariadb, err := database.Settings.AsMariadbSettings()
	require.NoError(t, err)

	return mariadb
}

// newMongodbSettings points at the test database of a server from containers.StartMongodb.
func newMongodbSettings(server containers.Endpoint) apiclient.MongodbSettings {
	return apiclient.MongodbSettings{
		Type:               apiclient.MongodbSettingsTypeMONGODB,
		Host:               server.Host,
		Port:               new(server.Port),
		Username:           containers.MongodbUsername,
		Password:           new(containers.MongodbPassword),
		Database:           containers.MongodbDatabase,
		AuthDatabase:       containers.MongodbAuthDatabase,
		CpuCount:           1,
		ExcludeCollections: []string{},
		IsDirectConnection: false,
		IsHttps:            false,
		IsSrv:              false,
		SshTunnel:          newDisabledSSHTunnel(),
		Version:            nil,
	}
}

func fromMongodbSettings(t *testing.T, mongodb apiclient.MongodbSettings) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromMongodbSettings(mongodb)
	})
}

func getMongodbSettings(t *testing.T, database apiclient.Database) apiclient.MongodbSettings {
	t.Helper()

	mongodb, err := database.Settings.AsMongodbSettings()
	require.NoError(t, err)

	return mongodb
}

// postgresqlTarget describes the logical PostgreSQL server that settings point at.
type postgresqlTarget struct {
	host     string
	port     int
	database string
	username string
	// password is nil when the settings leave the password out.
	password *string
}

func newPostgresqlContainerTarget(server containers.Endpoint) postgresqlTarget {
	return postgresqlTarget{
		host:     server.Host,
		port:     server.Port,
		database: containers.PostgresDatabase,
		username: containers.PostgresUsername,
		password: new(containers.PostgresPassword),
	}
}

func newPostgresqlLogicalSettings(t *testing.T, target postgresqlTarget) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromPostgresqlLogicalSettings(apiclient.PostgresqlLogicalSettings{
			Type:               apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL,
			Host:               target.host,
			Port:               target.port,
			Username:           target.username,
			Password:           target.password,
			Database:           new(target.database),
			SslMode:            apiclient.PostgresqlSslModeDisable,
			SshTunnel:          apiclient.SshTunnelSettings{IsEnabled: false},
			IncludeSchemas:     []string{},
			ExcludeTables:      []string{},
			ExcludeTableData:   []string{},
			CpuCount:           1,
			IsSkipUserMappings: false,
		})
	})
}

func newPostgresqlPhysicalSettings(
	t *testing.T,
	server containers.Endpoint,
	backupType apiclient.PhysicalBackupType,
) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromPostgresqlPhysicalSettings(apiclient.PostgresqlPhysicalSettings{
			Type:       apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
			BackupType: backupType,
			Host:       server.Host,
			Port:       server.Port,
			Username:   containers.PostgresUsername,
			Password:   new(containers.PostgresPassword),
			SslMode:    apiclient.PostgresqlSslModeDisable,
			SshTunnel:  apiclient.SshTunnelSettings{IsEnabled: false},
		})
	})
}

func newMysqlServerSettings(
	t *testing.T,
	server containers.Endpoint,
	username, password string,
) apiclient.DatabaseSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.DatabaseSettings) error {
		return settings.FromMysqlSettings(apiclient.MysqlSettings{
			Type:          apiclient.MysqlSettingsTypeMYSQL,
			Host:          server.Host,
			Port:          server.Port,
			Username:      username,
			Password:      new(password),
			Database:      new(containers.MysqlDatabase),
			IsHttps:       false,
			SshTunnel:     apiclient.SshTunnelSettings{IsEnabled: false},
			ExcludeTables: []string{},
		})
	})
}
