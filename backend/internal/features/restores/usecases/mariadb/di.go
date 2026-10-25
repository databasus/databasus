package usecases_mariadb

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getRestoreMariadbBackupUsecase = sync.OnceValue(func() *RestoreMariadbBackupUsecase {
	return &RestoreMariadbBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
	}
})

func GetRestoreMariadbBackupUsecase() *RestoreMariadbBackupUsecase {
	return getRestoreMariadbBackupUsecase()
}
