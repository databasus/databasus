package report

import (
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
)

func getBackupMethods() map[databases.Type]backups.Method {
	return map[databases.Type]backups.Method{
		databases.TypePostgresqlLogical:  backups.MethodLogical,
		databases.TypePostgresqlPhysical: backups.MethodPhysical,
		databases.TypeMysql:              backups.MethodLogical,
		databases.TypeMariadb:            backups.MethodLogical,
		databases.TypeMongodb:            backups.MethodLogical,
	}
}
