package usecases

import (
	"sync"

	usecases_mariadb "databasus-backend/internal/features/restores/usecases/mariadb"
	usecases_mongodb "databasus-backend/internal/features/restores/usecases/mongodb"
	usecases_mysql "databasus-backend/internal/features/restores/usecases/mysql"
	usecases_postgresql "databasus-backend/internal/features/restores/usecases/postgresql"
)

var getRestoreBackupUsecase = sync.OnceValue(func() *RestoreBackupUsecase {
	return &RestoreBackupUsecase{
		usecases_postgresql.GetRestorePostgresqlBackupUsecase(),
		usecases_mysql.GetRestoreMysqlBackupUsecase(),
		usecases_mariadb.GetRestoreMariadbBackupUsecase(),
		usecases_mongodb.GetRestoreMongodbBackupUsecase(),
	}
})

func GetRestoreBackupUsecase() *RestoreBackupUsecase {
	return getRestoreBackupUsecase()
}
