package usecases_logical_mysql

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMysqlBackupUsecase = sync.OnceValue(func() *CreateMysqlBackupUsecase {
	return &CreateMysqlBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreateMysqlBackupUsecase() *CreateMysqlBackupUsecase {
	return getCreateMysqlBackupUsecase()
}
