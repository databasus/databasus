package restores_core

import (
	"context"

	"databasus-backend/internal/modules/databases"
)

// SourceDatabase is the database a backup was taken from, with the connection settings of its
// type; only the settings of that type are set.
type SourceDatabase struct {
	Database          databases.Database
	PostgresqlLogical *databases.PostgresqlLogicalConnectionSettings
	Mysql             *databases.MysqlConnectionSettings
	Mariadb           *databases.MariadbConnectionSettings
	Mongodb           *databases.MongodbConnectionSettings
}

func ReadSourceDatabase(
	ctx context.Context,
	databasesModule databases.Module,
	database databases.Database,
) (SourceDatabase, error) {
	source := SourceDatabase{Database: database}

	switch database.Type {
	case databases.TypePostgresqlLogical:
		settings, err := databasesModule.GetPostgresqlLogicalConnectionSettings(ctx, database.ID)
		if err != nil {
			return SourceDatabase{}, err
		}

		source.PostgresqlLogical = &settings
	case databases.TypeMysql:
		settings, err := databasesModule.GetMysqlConnectionSettings(ctx, database.ID)
		if err != nil {
			return SourceDatabase{}, err
		}

		source.Mysql = &settings
	case databases.TypeMariadb:
		settings, err := databasesModule.GetMariadbConnectionSettings(ctx, database.ID)
		if err != nil {
			return SourceDatabase{}, err
		}

		source.Mariadb = &settings
	case databases.TypeMongodb:
		settings, err := databasesModule.GetMongodbConnectionSettings(ctx, database.ID)
		if err != nil {
			return SourceDatabase{}, err
		}

		source.Mongodb = &settings
	default:
		return SourceDatabase{}, ErrRestoreTypeNotSupported
	}

	return source, nil
}

// GetDatabaseName is the name of the logical database the backup holds.
func (s SourceDatabase) GetDatabaseName() string {
	switch {
	case s.PostgresqlLogical != nil:
		return s.PostgresqlLogical.Engine.Database
	case s.Mysql != nil:
		return s.Mysql.Engine.Database
	case s.Mariadb != nil:
		return s.Mariadb.Engine.Database
	case s.Mongodb != nil:
		return s.Mongodb.Engine.Database
	default:
		return ""
	}
}
