package chain_view

import (
	"sync"

	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
)

var getChainViewService = sync.OnceValue(func() *ChainViewService {
	return &ChainViewService{
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetWalHistoryRepository(),
	}
})

func GetChainViewService() *ChainViewService {
	return getChainViewService()
}
