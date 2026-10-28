package backups_services

import (
	"sync"

	audit_logs "databasus-backend/internal/features/audit_logs"
	backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_core_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

var getPhysicalBackupService = sync.OnceValue(func() *PhysicalBackupService {
	return &PhysicalBackupService{
		databases_bridge.GetAccess(),
		workspaces_bridge.GetAccess(),
		backups_bridge.GetModule,
		chain_view.GetChainViewService(),
		physical_core_service.GetPhysicalBackupService(),
		physical_repositories.GetFullBackupRepository(),
		physical_repositories.GetIncrementalBackupRepository(),
		physical_repositories.GetWalSegmentRepository(),
		physical_repositories.GetInFlightBackupRepository(),
		backuping_physical.GetPhysicalBackupCanceller(),
		backups_download.GetRestoreStreamWriter(),
		backups_download.GetRestoreTokenService(),
		encryption.GetMasterKey(),
		audit_logs.GetAuditLogService(),
		logger.GetLogger(),
	}
})

func GetPhysicalBackupService() *PhysicalBackupService {
	return getPhysicalBackupService()
}
