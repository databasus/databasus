package usecases_logical

import (
	"sync"

	usecases_logical_mariadb "databasus-backend/internal/features/backups/backups/usecases/logical/mariadb"
	usecases_logical_mongodb "databasus-backend/internal/features/backups/backups/usecases/logical/mongodb"
	usecases_logical_mysql "databasus-backend/internal/features/backups/backups/usecases/logical/mysql"
	usecases_logical_postgresql "databasus-backend/internal/features/backups/backups/usecases/logical/postgresql"
)

var getCreateBackupUsecase = sync.OnceValue(func() *CreateBackupUsecase {
	return &CreateBackupUsecase{
		usecases_logical_postgresql.GetCreatePostgresqlBackupUsecase(),
		usecases_logical_mysql.GetCreateMysqlBackupUsecase(),
		usecases_logical_mariadb.GetCreateMariadbBackupUsecase(),
		usecases_logical_mongodb.GetCreateMongodbBackupUsecase(),
	}
})

func GetCreateBackupUsecase() *CreateBackupUsecase {
	return getCreateBackupUsecase()
}
