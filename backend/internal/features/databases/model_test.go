package databases

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	postgresql_shared "databasus-backend/internal/features/databases/databases/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/tools"
)

func Test_TestConnection_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking(
	t *testing.T,
) {
	missingConfigCases := []struct {
		databaseType DatabaseType
		errorMessage string
	}{
		{DatabaseTypePostgresLogical, "postgresql logical config is not set"},
		{DatabaseTypePostgresPhysical, "postgresql physical config is not set"},
		{DatabaseTypeMysql, "mysql config is not set"},
		{DatabaseTypeMariadb, "mariadb config is not set"},
		{DatabaseTypeMongodb, "mongodb config is not set"},
		{DatabaseType("CASSANDRA"), "connection test not supported for database type: CASSANDRA"},
	}

	for _, missingConfigCase := range missingConfigCases {
		t.Run(string(missingConfigCase.databaseType), func(t *testing.T) {
			databaseWithoutConfig := Database{Type: missingConfigCase.databaseType}

			err := databaseWithoutConfig.TestConnection(
				logger.GetLogger(),
				encryption.GetFieldEncryptor(),
			)

			assert.EqualError(t, err, missingConfigCase.errorMessage)
		})
	}
}

func Test_ProbeLiveness_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking(
	t *testing.T,
) {
	missingConfigCases := []struct {
		databaseType DatabaseType
		errorMessage string
	}{
		{DatabaseTypePostgresLogical, "postgresql logical config is not set"},
		{DatabaseTypePostgresPhysical, "postgresql physical config is not set"},
		{DatabaseTypeMysql, "mysql config is not set"},
		{DatabaseTypeMariadb, "mariadb config is not set"},
		{DatabaseTypeMongodb, "mongodb config is not set"},
		{DatabaseType("CASSANDRA"), "connection test not supported for database type: CASSANDRA"},
	}

	for _, missingConfigCase := range missingConfigCases {
		t.Run(string(missingConfigCase.databaseType), func(t *testing.T) {
			databaseWithoutConfig := Database{Type: missingConfigCase.databaseType}

			err := databaseWithoutConfig.ProbeLiveness(
				t.Context(),
				logger.GetLogger(),
				encryption.GetFieldEncryptor(),
			)

			assert.EqualError(t, err, missingConfigCase.errorMessage)
		})
	}
}

func Test_GetRawDbSizeMb_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking(
	t *testing.T,
) {
	missingConfigCases := []struct {
		databaseType DatabaseType
		errorMessage string
	}{
		{DatabaseTypePostgresLogical, "postgresql logical config is not set"},
		{DatabaseTypeMysql, "mysql config is not set"},
		{DatabaseTypeMariadb, "mariadb config is not set"},
		{DatabaseTypeMongodb, "mongodb config is not set"},
		{
			DatabaseTypePostgresPhysical,
			"logical backup not supported for database type: POSTGRES_PHYSICAL",
		},
		{
			DatabaseType("CASSANDRA"),
			"logical backup not supported for database type: CASSANDRA",
		},
	}

	for _, missingConfigCase := range missingConfigCases {
		t.Run(string(missingConfigCase.databaseType), func(t *testing.T) {
			databaseWithoutConfig := Database{Type: missingConfigCase.databaseType}

			sizeMb, err := databaseWithoutConfig.GetRawDbSizeMb(
				t.Context(),
				logger.GetLogger(),
				encryption.GetFieldEncryptor(),
			)

			assert.EqualError(t, err, missingConfigCase.errorMessage)
			assert.Zero(t, sizeMb)
		})
	}
}

func Test_PopulateVersion_ForEveryDatabaseType_WithoutEngineConfig_ReturnsErrorInsteadOfPanicking(
	t *testing.T,
) {
	missingConfigCases := []struct {
		databaseType DatabaseType
		errorMessage string
	}{
		{DatabaseTypePostgresLogical, "postgresql logical config is not set"},
		{DatabaseTypeMysql, "mysql config is not set"},
		{DatabaseTypeMariadb, "mariadb config is not set"},
		{DatabaseTypeMongodb, "mongodb config is not set"},
		{
			DatabaseTypePostgresPhysical,
			"version detection not supported for database type: POSTGRES_PHYSICAL",
		},
		{
			DatabaseType("CASSANDRA"),
			"version detection not supported for database type: CASSANDRA",
		},
	}

	for _, missingConfigCase := range missingConfigCases {
		t.Run(string(missingConfigCase.databaseType), func(t *testing.T) {
			databaseWithoutConfig := Database{Type: missingConfigCase.databaseType}

			err := databaseWithoutConfig.PopulateVersion(
				logger.GetLogger(),
				encryption.GetFieldEncryptor(),
			)

			assert.EqualError(t, err, missingConfigCase.errorMessage)
		})
	}
}

// Dispatching on the first non-nil sub-model instead would open the tunnel for Type and connect
// with a different engine, straight past the bastion.
func Test_PopulateVersion_WhenTheDatabaseCarriesAStaleEngineConfig_DispatchesOnTheType(t *testing.T) {
	databaseWithStaleConfig := Database{
		Type:    DatabaseTypeMysql,
		Mongodb: filledSourceDatabase(DatabaseTypeMongodb).Mongodb,
	}

	err := databaseWithStaleConfig.PopulateVersion(
		logger.GetLogger(),
		encryption.GetFieldEncryptor(),
	)

	assert.EqualError(t, err, "mysql config is not set")
}

func Test_HideSensitiveData_ForEveryTunnelCapableType_ClearsTheTunnelSecrets(t *testing.T) {
	for _, databaseType := range []DatabaseType{
		DatabaseTypePostgresLogical,
		DatabaseTypePostgresPhysical,
		DatabaseTypeMysql,
		DatabaseTypeMariadb,
		DatabaseTypeMongodb,
	} {
		t.Run(string(databaseType), func(t *testing.T) {
			database := filledSourceDatabase(databaseType)

			database.HideSensitiveData()

			tunnel := getSshTunnelConfig(t, database)
			assert.Empty(t, tunnel.Password)
			assert.Empty(t, tunnel.PrivateKey)
			assert.Empty(t, tunnel.PrivateKeyPassphrase)
			assert.Equal(t, "tunneluser", tunnel.Username, "the address stays readable in the UI")
			assert.True(t, tunnel.IsEnabled)
		})
	}
}

func Test_ClearSshTunnelConfig_ForEveryTunnelCapableType_RemovesTheWholeBlock(t *testing.T) {
	for _, databaseType := range []DatabaseType{
		DatabaseTypePostgresLogical,
		DatabaseTypePostgresPhysical,
		DatabaseTypeMysql,
		DatabaseTypeMariadb,
		DatabaseTypeMongodb,
	} {
		t.Run(string(databaseType), func(t *testing.T) {
			database := filledSourceDatabase(databaseType)

			database.ClearSshTunnelConfig()

			assert.Equal(t, sshtunnel.Config{}, getSshTunnelConfig(t, database))
		})
	}
}

func getSshTunnelConfig(t *testing.T, database *Database) sshtunnel.Config {
	t.Helper()

	switch database.Type {
	case DatabaseTypePostgresLogical:
		return database.PostgresqlLogical.SshTunnel
	case DatabaseTypePostgresPhysical:
		return database.PostgresqlPhysical.SshTunnel
	case DatabaseTypeMysql:
		return database.Mysql.SshTunnel
	case DatabaseTypeMariadb:
		return database.Mariadb.SshTunnel
	case DatabaseTypeMongodb:
		return database.Mongodb.SshTunnel
	}

	t.Fatalf("no ssh tunnel config for database type %s", database.Type)

	return sshtunnel.Config{}
}

// These compare whole structs on purpose: a field added to an engine model tomorrow has to fail
// here rather than quietly vanish from every copied database.

func filledSshTunnel() sshtunnel.Config {
	return sshtunnel.Config{
		IsEnabled:            true,
		Host:                 "bastion.example.com",
		Port:                 2222,
		Username:             "tunneluser",
		AuthType:             sshtunnel.AuthTypePrivateKey,
		Password:             "enc:tunnelpassword",
		PrivateKey:           "enc:privatekey",
		PrivateKeyPassphrase: "enc:passphrase",
	}
}

func filledSourceDatabase(databaseType DatabaseType) *Database {
	lastBackupTime := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	lastBackupErrorMessage := "previous failure"
	healthStatus := HealthStatusAvailable
	workspaceID := uuid.New()
	databaseID := uuid.New()
	databaseName := "sourcedb"
	systemIdentifier := "7300000000000000000"
	walSegmentSizeBytes := int64(16 * 1024 * 1024)
	mongodbPort := 27017

	database := &Database{
		ID:          databaseID,
		WorkspaceID: &workspaceID,
		Name:        "Source",
		Type:        databaseType,
		Notifiers: []DatabaseNotifier{
			{ID: uuid.New(), WorkspaceID: workspaceID, Name: "Ops channel"},
		},
		LastBackupTime:         &lastBackupTime,
		LastBackupErrorMessage: &lastBackupErrorMessage,
		HealthStatus:           &healthStatus,
	}

	switch databaseType {
	case DatabaseTypePostgresLogical:
		database.PostgresqlLogical = &postgresql_logical.PostgresqlLogicalDatabase{
			ID:            uuid.New(),
			DatabaseID:    &databaseID,
			Version:       tools.PostgresqlVersion16,
			Host:          "db.internal",
			Port:          5432,
			Username:      "testuser",
			Password:      "enc:testpassword",
			Database:      &databaseName,
			SslMode:       postgresql_shared.PostgresSslModeRequire,
			SslClientCert: "clientcert",
			SslClientKey:  "enc:clientkey",
			SslRootCert:   "rootcert",
			SshTunnel:     filledSshTunnel(),
			// Set as if the source had just been reached through a tunnel, which is the state the
			// copy must not inherit.
			LocalTunnelEndpoint: &sshtunnel.Endpoint{Host: "127.0.0.1", Port: 41000},
			IncludeSchemas:      []string{"public", "reporting"},
			ExcludeTables:       []string{"audit_log"},
			ExcludeTableData:    []string{"event_log"},
			CpuCount:            4,
			IsSkipUserMappings:  true,
		}
	case DatabaseTypePostgresPhysical:
		database.PostgresqlPhysical = &postgresql_physical.PostgresqlPhysicalDatabase{
			ID:                  uuid.New(),
			DatabaseID:          &databaseID,
			Version:             tools.PostgresqlVersion17,
			BackupType:          postgresql_physical.BackupTypeFullOnly,
			Host:                "db.internal",
			Port:                5432,
			Username:            "testuser",
			Password:            "enc:testpassword",
			SslMode:             postgresql_shared.PostgresSslModeRequire,
			SslClientCert:       "clientcert",
			SslClientKey:        "enc:clientkey",
			SslRootCert:         "rootcert",
			SshTunnel:           filledSshTunnel(),
			LocalTunnelEndpoint: &sshtunnel.Endpoint{Host: "127.0.0.1", Port: 41000},
			ReplicationSlotName: "databasus_slot_deadbeef",
			SystemIdentifier:    &systemIdentifier,
			WalSegmentSizeBytes: &walSegmentSizeBytes,
		}
	case DatabaseTypeMysql:
		database.Mysql = &mysql.MysqlDatabase{
			ID:            uuid.New(),
			DatabaseID:    &databaseID,
			Version:       tools.MysqlVersion84,
			Host:          "db.internal",
			Port:          3306,
			Username:      "testuser",
			Password:      "enc:testpassword",
			Database:      &databaseName,
			IsHttps:       true,
			SshTunnel:     filledSshTunnel(),
			ExcludeTables: []string{"audit_log"},
			Privileges:    "SELECT,LOCK TABLES",
		}
	case DatabaseTypeMariadb:
		database.Mariadb = &mariadb.MariadbDatabase{
			ID:                  uuid.New(),
			DatabaseID:          &databaseID,
			Version:             tools.MariadbVersion114,
			Host:                "db.internal",
			Port:                3306,
			Username:            "testuser",
			Password:            "enc:testpassword",
			Database:            &databaseName,
			IsHttps:             true,
			IsExcludeEvents:     true,
			IsSkipGaleraDisable: true,
			SshTunnel:           filledSshTunnel(),
			ExcludeTables:       []string{"audit_log"},
			Privileges:          "SELECT,LOCK TABLES",
		}
	case DatabaseTypeMongodb:
		database.Mongodb = &mongodb.MongodbDatabase{
			ID:                 uuid.New(),
			DatabaseID:         &databaseID,
			Version:            tools.MongodbVersion7,
			Host:               "db.internal",
			Port:               &mongodbPort,
			Username:           "testuser",
			Password:           "enc:testpassword",
			Database:           "sourcedb",
			AuthDatabase:       "admin",
			IsHttps:            true,
			IsDirectConnection: true,
			CpuCount:           4,
			SshTunnel:          filledSshTunnel(),
			ExcludeCollections: []string{"sessions"},
		}
	}

	return database
}

func Test_CopyForNewDatabase_WhenTheSourceIsPostgresqlLogical_KeepsEveryConfiguredField(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypePostgresLogical)

	copiedEngine := sourceDatabase.CopyForNewDatabase().PostgresqlLogical
	require.NotNil(t, copiedEngine)

	sourceEngineWithoutIdentity := *sourceDatabase.PostgresqlLogical
	sourceEngineWithoutIdentity.ID = uuid.Nil
	sourceEngineWithoutIdentity.DatabaseID = nil
	sourceEngineWithoutIdentity.LocalTunnelEndpoint = nil

	assert.Equal(t, sourceEngineWithoutIdentity, *copiedEngine)
	assert.NotSame(t, sourceDatabase.PostgresqlLogical.Database, copiedEngine.Database)
}

// The endpoint belongs to the operation that opened the tunnel, so a copy inheriting it would point
// at a forwarder that closed with that operation.
func Test_CopyForNewDatabase_WhenTheSourceIsPostgresqlLogical_DropsTheLocalTunnelEndpoint(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypePostgresLogical)
	require.NotNil(t, sourceDatabase.PostgresqlLogical.LocalTunnelEndpoint)

	copiedEngine := sourceDatabase.CopyForNewDatabase().PostgresqlLogical
	require.NotNil(t, copiedEngine)

	assert.Nil(t, copiedEngine.LocalTunnelEndpoint)
}

func Test_CopyForNewDatabase_WhenTheSourceIsMysql_KeepsEveryConfiguredField(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMysql)

	copiedEngine := sourceDatabase.CopyForNewDatabase().Mysql
	require.NotNil(t, copiedEngine)

	sourceEngineWithoutIdentity := *sourceDatabase.Mysql
	sourceEngineWithoutIdentity.ID = uuid.Nil
	sourceEngineWithoutIdentity.DatabaseID = nil

	assert.Equal(t, sourceEngineWithoutIdentity, *copiedEngine)
	assert.NotSame(t, sourceDatabase.Mysql.Database, copiedEngine.Database)
}

func Test_CopyForNewDatabase_WhenTheSourceIsMariadb_KeepsEveryConfiguredField(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMariadb)

	copiedEngine := sourceDatabase.CopyForNewDatabase().Mariadb
	require.NotNil(t, copiedEngine)

	sourceEngineWithoutIdentity := *sourceDatabase.Mariadb
	sourceEngineWithoutIdentity.ID = uuid.Nil
	sourceEngineWithoutIdentity.DatabaseID = nil

	assert.Equal(t, sourceEngineWithoutIdentity, *copiedEngine)
	assert.NotSame(t, sourceDatabase.Mariadb.Database, copiedEngine.Database)
}

func Test_CopyForNewDatabase_WhenTheSourceIsMongodb_KeepsEveryConfiguredField(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMongodb)

	copiedEngine := sourceDatabase.CopyForNewDatabase().Mongodb
	require.NotNil(t, copiedEngine)

	sourceEngineWithoutIdentity := *sourceDatabase.Mongodb
	sourceEngineWithoutIdentity.ID = uuid.Nil
	sourceEngineWithoutIdentity.DatabaseID = nil

	assert.Equal(t, sourceEngineWithoutIdentity, *copiedEngine)
}

func Test_CopyForNewDatabase_WhenTheSourceIsMongodb_GivesTheCopyItsOwnPortPointer(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMongodb)

	copiedEngine := sourceDatabase.CopyForNewDatabase().Mongodb
	require.NotNil(t, copiedEngine)

	*copiedEngine.Port = 27018

	assert.Equal(t, 27017, *sourceDatabase.Mongodb.Port)
}

func Test_CopyForNewDatabase_WhenTheMongodbSourceUsesSrv_KeepsTheSrvFlag(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMongodb)
	// SRV and a tunnel are mutually exclusive, so the flag can only be exercised untunneled.
	sourceDatabase.Mongodb.SshTunnel = sshtunnel.Config{}
	sourceDatabase.Mongodb.IsSrv = true
	sourceDatabase.Mongodb.Port = nil

	copiedEngine := sourceDatabase.CopyForNewDatabase().Mongodb
	require.NotNil(t, copiedEngine)

	assert.True(t, copiedEngine.IsSrv)
	assert.Nil(t, copiedEngine.Port)
}

// The inverse of the other engines: inheriting the slot name would leave two databases driving one
// replication slot on the source cluster, and BeforeCreate only mints a new one when it is empty.
func Test_CopyForNewDatabase_WhenTheSourceIsPostgresqlPhysical_DoesNotInheritServerManagedState(
	t *testing.T,
) {
	sourceDatabase := filledSourceDatabase(DatabaseTypePostgresPhysical)

	copiedEngine := sourceDatabase.CopyForNewDatabase().PostgresqlPhysical
	require.NotNil(t, copiedEngine)

	assert.Empty(t, copiedEngine.ReplicationSlotName)
	assert.Nil(t, copiedEngine.SystemIdentifier)
	assert.Nil(t, copiedEngine.WalSegmentSizeBytes)

	sourceEngineWithoutIdentity := *sourceDatabase.PostgresqlPhysical
	sourceEngineWithoutIdentity.ID = uuid.Nil
	sourceEngineWithoutIdentity.DatabaseID = nil
	sourceEngineWithoutIdentity.ReplicationSlotName = ""
	sourceEngineWithoutIdentity.SystemIdentifier = nil
	sourceEngineWithoutIdentity.WalSegmentSizeBytes = nil
	sourceEngineWithoutIdentity.LocalTunnelEndpoint = nil

	assert.Equal(t, sourceEngineWithoutIdentity, *copiedEngine)
	assert.Equal(t, "databasus_slot_deadbeef", sourceDatabase.PostgresqlPhysical.ReplicationSlotName)
}

func Test_CopyForNewDatabase_WhenTheSourceIsPostgresqlPhysical_DropsTheLocalTunnelEndpoint(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypePostgresPhysical)
	require.NotNil(t, sourceDatabase.PostgresqlPhysical.LocalTunnelEndpoint)

	copiedEngine := sourceDatabase.CopyForNewDatabase().PostgresqlPhysical
	require.NotNil(t, copiedEngine)

	assert.Nil(t, copiedEngine.LocalTunnelEndpoint)
	assert.Equal(t, filledSshTunnel(), copiedEngine.SshTunnel,
		"a clone behind the same bastion still goes through it")
}

func Test_CopyForNewDatabase_DropsIdentityAndBackupHistory(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMysql)

	copiedDatabase := sourceDatabase.CopyForNewDatabase()

	assert.Equal(t, uuid.Nil, copiedDatabase.ID)
	assert.Nil(t, copiedDatabase.LastBackupTime)
	assert.Nil(t, copiedDatabase.LastBackupErrorMessage)
	assert.Equal(t, sourceDatabase.WorkspaceID, copiedDatabase.WorkspaceID)
	assert.Equal(t, *sourceDatabase.HealthStatus, *copiedDatabase.HealthStatus)
	assert.NotSame(t, sourceDatabase.HealthStatus, copiedDatabase.HealthStatus)
}

// The only many2many on Database: a copy that lost its notifiers would back up in silence.
func Test_CopyForNewDatabase_WhenTheSourceHasNotifiers_CopiesThemIntoItsOwnSlice(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMysql)
	require.NotEmpty(t, sourceDatabase.Notifiers)

	copiedDatabase := sourceDatabase.CopyForNewDatabase()

	assert.Equal(t, sourceDatabase.Notifiers, copiedDatabase.Notifiers)
	assert.NotSame(t, &sourceDatabase.Notifiers[0], &copiedDatabase.Notifiers[0])
}

// A shallow copy of Database carries all five engine pointers, and a copy that answers to two
// engines at once would confuse every type switch downstream.
func Test_CopyForNewDatabase_WhenTheSourceCarriesAStaleEngineConfig_KeepsOnlyTheMatchingOne(t *testing.T) {
	sourceDatabase := filledSourceDatabase(DatabaseTypeMysql)
	sourceDatabase.Mongodb = filledSourceDatabase(DatabaseTypeMongodb).Mongodb

	copiedDatabase := sourceDatabase.CopyForNewDatabase()

	assert.NotNil(t, copiedDatabase.Mysql)
	assert.Nil(t, copiedDatabase.Mongodb)
	assert.Nil(t, copiedDatabase.PostgresqlLogical)
	assert.Nil(t, copiedDatabase.PostgresqlPhysical)
	assert.Nil(t, copiedDatabase.Mariadb)
}
