package usecases_logical_postgresql

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreatePostgresqlBackupUsecase = sync.OnceValue(func() *CreatePostgresqlBackupUsecase {
	return &CreatePostgresqlBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		databases_bridge.GetModule,
	}
})

func GetCreatePostgresqlBackupUsecase() *CreatePostgresqlBackupUsecase {
	return getCreatePostgresqlBackupUsecase()
}
