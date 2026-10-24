package usecases_logical_mongodb

import (
	"sync"

	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMongodbBackupUsecase = sync.OnceValue(func() *CreateMongodbBackupUsecase {
	return &CreateMongodbBackupUsecase{
		logger.GetLogger(),
		encryption_secrets.GetSecretKeyService(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreateMongodbBackupUsecase() *CreateMongodbBackupUsecase {
	return getCreateMongodbBackupUsecase()
}
