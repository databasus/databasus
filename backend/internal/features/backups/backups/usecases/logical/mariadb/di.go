package usecases_logical_mariadb

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMariadbBackupUsecase = sync.OnceValue(func() *CreateMariadbBackupUsecase {
	return &CreateMariadbBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		databases_bridge.GetModule,
	}
})

func GetCreateMariadbBackupUsecase() *CreateMariadbBackupUsecase {
	return getCreateMariadbBackupUsecase()
}
