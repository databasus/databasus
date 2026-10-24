package usecases_logical_mariadb

import (
	"sync"

	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMariadbBackupUsecase = sync.OnceValue(func() *CreateMariadbBackupUsecase {
	return &CreateMariadbBackupUsecase{
		logger.GetLogger(),
		secrets.GetSecretKeyService(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreateMariadbBackupUsecase() *CreateMariadbBackupUsecase {
	return getCreateMariadbBackupUsecase()
}
