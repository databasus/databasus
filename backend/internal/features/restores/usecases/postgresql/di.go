package usecases_postgresql

import (
	"sync"

	"databasus-backend/internal/features/encryption/secrets"
	"databasus-backend/internal/util/logger"
)

var getRestorePostgresqlBackupUsecase = sync.OnceValue(func() *RestorePostgresqlBackupUsecase {
	return &RestorePostgresqlBackupUsecase{
		logger.GetLogger(),
		secrets.GetSecretKeyService(),
	}
})

func GetRestorePostgresqlBackupUsecase() *RestorePostgresqlBackupUsecase {
	return getRestorePostgresqlBackupUsecase()
}
