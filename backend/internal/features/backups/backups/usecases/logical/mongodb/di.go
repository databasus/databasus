package usecases_logical_mongodb

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMongodbBackupUsecase = sync.OnceValue(func() *CreateMongodbBackupUsecase {
	return &CreateMongodbBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		databases_bridge.GetModule,
	}
})

func GetCreateMongodbBackupUsecase() *CreateMongodbBackupUsecase {
	return getCreateMongodbBackupUsecase()
}
