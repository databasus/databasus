package usecases_mysql

import (
	"sync"

	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var getRestoreMysqlBackupUsecase = sync.OnceValue(func() *RestoreMysqlBackupUsecase {
	return &RestoreMysqlBackupUsecase{
		logger.GetLogger(),
		secrets.GetSecretKeyService(),
	}
})

func GetRestoreMysqlBackupUsecase() *RestoreMysqlBackupUsecase {
	return getRestoreMysqlBackupUsecase()
}
