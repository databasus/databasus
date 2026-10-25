package usecases_mongodb

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getRestoreMongodbBackupUsecase = sync.OnceValue(func() *RestoreMongodbBackupUsecase {
	return &RestoreMongodbBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
	}
})

func GetRestoreMongodbBackupUsecase() *RestoreMongodbBackupUsecase {
	return getRestoreMongodbBackupUsecase()
}
