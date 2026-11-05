package pipeline

import "databasus-backend/internal/modules/databases"

// newEngineRegistry is the only place in the module that maps a database type to its engine. A
// physical PostgreSQL database has no entry, because it has no logical backups.
func newEngineRegistry(deps Deps) map[databases.Type]logicalEngine {
	return map[databases.Type]logicalEngine{
		databases.TypePostgresqlLogical: &postgresqlEngine{
			databases:      deps.Databases,
			clientToolsDir: deps.ClientToolsDir,
			tempDir:        deps.TempDir,
		},
		databases.TypeMysql:   &mysqlEngine{databases: deps.Databases, clientToolsDir: deps.ClientToolsDir},
		databases.TypeMariadb: &mariadbEngine{databases: deps.Databases, clientToolsDir: deps.ClientToolsDir},
		databases.TypeMongodb: &mongodbEngine{databases: deps.Databases, clientToolsDir: deps.ClientToolsDir},
	}
}
