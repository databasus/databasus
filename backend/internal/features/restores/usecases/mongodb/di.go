package usecases_mongodb

import (
	"sync"

	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var getRestoreMongodbBackupUsecase = sync.OnceValue(func() *RestoreMongodbBackupUsecase {
	return &RestoreMongodbBackupUsecase{
		logger.GetLogger(),
		encryption_secrets.GetSecretKeyService(),
	}
})

func GetRestoreMongodbBackupUsecase() *RestoreMongodbBackupUsecase {
	return getRestoreMongodbBackupUsecase()
}
