package usecases_mysql

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getRestoreMysqlBackupUsecase = sync.OnceValue(func() *RestoreMysqlBackupUsecase {
	return &RestoreMysqlBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
	}
})

func GetRestoreMysqlBackupUsecase() *RestoreMysqlBackupUsecase {
	return getRestoreMysqlBackupUsecase()
}
