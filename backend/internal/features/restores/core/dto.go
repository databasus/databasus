package restores_core

type RestoreBackupRequest struct {
	PostgresqlLogicalDatabase *PostgresqlTargetDatabase `json:"postgresqlDatabase"`
	MysqlDatabase             *MysqlTargetDatabase      `json:"mysqlDatabase"`
	MariadbDatabase           *MariadbTargetDatabase    `json:"mariadbDatabase"`
	MongodbDatabase           *MongodbTargetDatabase    `json:"mongodbDatabase"`
}

type RestoreOptions struct {
	IsExcludeExtensions bool
	IsSkipUserMappings  bool
}
