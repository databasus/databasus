package usecases_logical_postgresql

import (
	"sync"

	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreatePostgresqlBackupUsecase = sync.OnceValue(func() *CreatePostgresqlBackupUsecase {
	return &CreatePostgresqlBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		encryption.GetFieldEncryptor(),
	}
})

func GetCreatePostgresqlBackupUsecase() *CreatePostgresqlBackupUsecase {
	return getCreatePostgresqlBackupUsecase()
}
