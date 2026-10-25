package usecases_postgresql

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getRestorePostgresqlBackupUsecase = sync.OnceValue(func() *RestorePostgresqlBackupUsecase {
	return &RestorePostgresqlBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
	}
})

func GetRestorePostgresqlBackupUsecase() *RestorePostgresqlBackupUsecase {
	return getRestorePostgresqlBackupUsecase()
}
