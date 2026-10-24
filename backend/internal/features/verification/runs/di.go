package verification_runs

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/features/audit_logs"
	backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_services "databasus-backend/internal/features/backups/backups/services"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	"databasus-backend/internal/util/logger"
)

var getVerificationRepository = sync.OnceValue(func() *VerificationRepository {
	return &VerificationRepository{}
})

var getVerificationService = sync.OnceValue(func() *VerificationService {
	return &VerificationService{
		getVerificationRepository(),
		databases.GetDatabaseService(),
		backups_services.GetBackupService(),
		verification_config.GetVerificationConfigService(),
		notifiers.GetNotifierService(),
		workspaces_services.GetWorkspaceService(),
		audit_logs.GetAuditLogService(),
		logger.GetLogger(),
	}
})

var getVerificationScheduler = sync.OnceValue(func() *VerificationScheduler {
	return &VerificationScheduler{
		getVerificationRepository(),
		getVerificationService(),
		verification_config.GetVerificationConfigService(),
		verification_agents.GetAgentService(),
		backups_services.GetBackupService(),
		databases.GetDatabaseService(),
		logger.GetLogger(),
		atomic.Bool{},
	}
})

var getVerificationController = sync.OnceValue(func() *VerificationController {
	return &VerificationController{
		getVerificationService(),
	}
})

var getVerificationAgentController = sync.OnceValue(func() *VerificationAgentController {
	return &VerificationAgentController{
		getVerificationService(),
		verification_agents.GetAgentService(),
		logger.GetLogger(),
	}
})

func GetVerificationService() *VerificationService {
	return getVerificationService()
}

func GetVerificationScheduler() *VerificationScheduler {
	return getVerificationScheduler()
}

func GetVerificationController() *VerificationController {
	return getVerificationController()
}

func GetVerificationAgentController() *VerificationAgentController {
	return getVerificationAgentController()
}

var SetupDependencies = sync.OnceFunc(func() {
	verification_agents.GetAgentService().AddAgentHeartbeatedListener(getVerificationService())
	backuping_logical.GetBackupsScheduler().AddBackupCompletionListener(getVerificationService())
})
