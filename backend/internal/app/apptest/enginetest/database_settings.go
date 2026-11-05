package enginetest

import (
	"testing"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// NewDatabaseSettings wraps the settings of one engine into the contract's settings union, filling in
// the type the union is told apart by.
func NewDatabaseSettings(t *testing.T, settings any) apiclient.DatabaseSettings {
	t.Helper()

	var databaseSettings apiclient.DatabaseSettings

	var err error

	switch typedSettings := settings.(type) {
	case apiclient.PostgresqlPhysicalSettings:
		typedSettings.Type = apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL
		err = databaseSettings.FromPostgresqlPhysicalSettings(typedSettings)
	case apiclient.PostgresqlLogicalSettings:
		typedSettings.Type = apiclient.PostgresqlLogicalSettingsTypePOSTGRESLOGICAL
		err = databaseSettings.FromPostgresqlLogicalSettings(typedSettings)
	case apiclient.MysqlSettings:
		typedSettings.Type = apiclient.MysqlSettingsTypeMYSQL
		err = databaseSettings.FromMysqlSettings(typedSettings)
	case apiclient.MariadbSettings:
		typedSettings.Type = apiclient.MariadbSettingsTypeMARIADB
		err = databaseSettings.FromMariadbSettings(typedSettings)
	case apiclient.MongodbSettings:
		typedSettings.Type = apiclient.MongodbSettingsTypeMONGODB
		err = databaseSettings.FromMongodbSettings(typedSettings)
	default:
		t.Fatalf("unsupported database settings %T", settings)
	}

	if err != nil {
		t.Fatalf("encode the database settings: %v", err)
	}

	return databaseSettings
}
