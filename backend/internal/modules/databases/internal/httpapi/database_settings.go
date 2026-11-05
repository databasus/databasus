package httpapi

import (
	"fmt"

	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/providers/mariadb"
	"databasus-backend/internal/modules/databases/internal/providers/mongodb"
	"databasus-backend/internal/modules/databases/internal/providers/mysql"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqllogical"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqlphysical"
)

// toProviderSettings picks the provider's conversion by the discriminator of the settings. A
// provider cannot convert itself, because it would import the generated types of this package.
func toProviderSettings(body DatabaseSettings) (providers.Settings, error) {
	settingsBody, err := body.ValueByDiscriminator()
	if err != nil {
		return nil, errUnknownSettingsValue
	}

	switch typedBody := settingsBody.(type) {
	case PostgresqlLogicalSettings:
		return toPostgresqlLogicalSettings(typedBody), nil
	case PostgresqlPhysicalSettings:
		return toPostgresqlPhysicalSettings(typedBody), nil
	case MysqlSettings:
		return toMysqlSettings(typedBody), nil
	case MariadbSettings:
		return toMariadbSettings(typedBody), nil
	case MongodbSettings:
		return toMongodbSettings(typedBody), nil
	}

	return nil, errUnknownSettingsValue
}

// fromProviderSettings leaves every secret out, because the contract marks them writeOnly.
func fromProviderSettings(settings providers.Settings) (DatabaseSettings, error) {
	var body DatabaseSettings

	var err error

	switch typedSettings := settings.(type) {
	case postgresqllogical.Settings:
		err = body.FromPostgresqlLogicalSettings(fromPostgresqlLogicalSettings(typedSettings))
	case postgresqlphysical.Settings:
		err = body.FromPostgresqlPhysicalSettings(fromPostgresqlPhysicalSettings(typedSettings))
	case mysql.Settings:
		err = body.FromMysqlSettings(fromMysqlSettings(typedSettings))
	case mariadb.Settings:
		err = body.FromMariadbSettings(fromMariadbSettings(typedSettings))
	case mongodb.Settings:
		err = body.FromMongodbSettings(fromMongodbSettings(typedSettings))
	default:
		return DatabaseSettings{}, fmt.Errorf("%w: %T", errUnknownSettingsType, settings)
	}

	if err != nil {
		return DatabaseSettings{}, fmt.Errorf("encode the database settings: %w", err)
	}

	return body, nil
}
