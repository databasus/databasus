package usecases_logical_mysql

import (
	"sync"

	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMysqlBackupUsecase = sync.OnceValue(func() *CreateMysqlBackupUsecase {
	return &CreateMysqlBackupUsecase{
		logger.GetLogger(),
		secrets.GetSecretKeyService(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreateMysqlBackupUsecase() *CreateMysqlBackupUsecase {
	return getCreateMysqlBackupUsecase()
}
