package usecases

import (
	"sync"

	"databasus-backend/internal/config"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getRestoreBackupUsecase = sync.OnceValue(func() *RestoreBackupUsecase {
	return &RestoreBackupUsecase{
		logger:         logger.GetLogger(),
		masterKey:      encryption.GetMasterKey(),
		clientToolsDir: config.GetEnv().ClientToolsDir,
		tempDir:        config.GetEnv().TempFolder,
	}
})

func GetRestoreBackupUsecase() *RestoreBackupUsecase {
	return getRestoreBackupUsecase()
}
