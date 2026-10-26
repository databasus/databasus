package usecases_logical_mysql

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getCreateMysqlBackupUsecase = sync.OnceValue(func() *CreateMysqlBackupUsecase {
	return &CreateMysqlBackupUsecase{
		logger.GetLogger(),
		encryption.GetMasterKey(),
		databases_bridge.GetModule,
	}
})

func GetCreateMysqlBackupUsecase() *CreateMysqlBackupUsecase {
	return getCreateMysqlBackupUsecase()
}
