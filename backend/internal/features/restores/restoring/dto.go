package restoring

import restores_core "databasus-backend/internal/features/restores/core"

type RestoreDatabaseCache struct {
	PostgresqlLogicalDatabase *restores_core.PostgresqlTargetDatabase `json:"postgresqlDatabase,omitzero"`
	MysqlDatabase             *restores_core.MysqlTargetDatabase      `json:"mysqlDatabase,omitzero"`
	MariadbDatabase           *restores_core.MariadbTargetDatabase    `json:"mariadbDatabase,omitzero"`
	MongodbDatabase           *restores_core.MongodbTargetDatabase    `json:"mongodbDatabase,omitzero"`
}

func (c *RestoreDatabaseCache) HasDatabaseConfiguration() bool {
	return c.PostgresqlLogicalDatabase != nil ||
		c.MysqlDatabase != nil ||
		c.MariadbDatabase != nil ||
		c.MongodbDatabase != nil
}

func (c *RestoreDatabaseCache) getRequest() restores_core.RestoreBackupRequest {
	return restores_core.RestoreBackupRequest{
		PostgresqlLogicalDatabase: c.PostgresqlLogicalDatabase,
		MysqlDatabase:             c.MysqlDatabase,
		MariadbDatabase:           c.MariadbDatabase,
		MongodbDatabase:           c.MongodbDatabase,
	}
}
