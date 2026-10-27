package usecases_logical

import (
	"sync"

	"databasus-backend/internal/config"
	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateBackupUsecase = sync.OnceValue(func() *CreateBackupUsecase {
	return &CreateBackupUsecase{
		logger:         logger.GetLogger(),
		masterKey:      encryption.GetMasterKey(),
		getDatabases:   databases_bridge.GetModule,
		clientToolsDir: config.GetEnv().ClientToolsDir,
	}
})

func GetCreateBackupUsecase() *CreateBackupUsecase {
	return getCreateBackupUsecase()
}
