package backups_core_logical

import "sync"

var getBackupRepository = sync.OnceValue(func() *BackupRepository {
	return &BackupRepository{}
})

func GetBackupRepository() *BackupRepository {
	return getBackupRepository()
}
