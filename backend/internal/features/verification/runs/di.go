package verification_runs

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/features/audit_logs"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/util/logger"
)

var getVerificationRepository = sync.OnceValue(func() *VerificationRepository {
	return &VerificationRepository{}
})

var getVerificationService = sync.OnceValue(func() *VerificationService {
	return &VerificationService{
		getVerificationRepository(),
		databases_bridge.GetAccess(),
		databases_bridge.GetModule,
		backups_bridge.GetModule,
		verification_config.GetVerificationConfigService(),
		notifiers_bridge.GetSender(),
		workspaces_bridge.GetAccess(),
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
		databases_bridge.GetAccess(),
		backups_bridge.GetModule,
		databases_bridge.GetModule,
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
})
