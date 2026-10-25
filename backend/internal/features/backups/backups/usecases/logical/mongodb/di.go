package usecases_logical_mongodb

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMongodbBackupUsecase = sync.OnceValue(func() *CreateMongodbBackupUsecase {
	return &CreateMongodbBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreateMongodbBackupUsecase() *CreateMongodbBackupUsecase {
	return getCreateMongodbBackupUsecase()
}
