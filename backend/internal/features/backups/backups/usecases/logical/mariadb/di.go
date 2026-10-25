package usecases_logical_mariadb

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMariadbBackupUsecase = sync.OnceValue(func() *CreateMariadbBackupUsecase {
	return &CreateMariadbBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreateMariadbBackupUsecase() *CreateMariadbBackupUsecase {
	return getCreateMariadbBackupUsecase()
}
