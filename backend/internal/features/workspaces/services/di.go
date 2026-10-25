package workspaces_services

import (
	"sync"

	"databasus-backend/internal/features/audit_logs"
	"databasus-backend/internal/features/email"
	users_services "databasus-backend/internal/features/users/services"
	workspaces_interfaces "databasus-backend/internal/features/workspaces/interfaces"
	workspaces_repositories "databasus-backend/internal/features/workspaces/repositories"
	"databasus-backend/internal/util/logger"
)

var (
	getWorkspaceRepository = sync.OnceValue(func() *workspaces_repositories.WorkspaceRepository {
		return &workspaces_repositories.WorkspaceRepository{}
	})
	getMembershipRepository = sync.OnceValue(func() *workspaces_repositories.MembershipRepository {
		return &workspaces_repositories.MembershipRepository{}
	})
)

var getWorkspaceService = sync.OnceValue(func() *WorkspaceService {
	return &WorkspaceService{
		getWorkspaceRepository(),
		getMembershipRepository(),
		users_services.GetUserService(),
		audit_logs.GetAuditLogService(),
		users_services.GetSettingsService(),
		[]workspaces_interfaces.WorkspaceDeletionListener{},
	}
})

var getMembershipService = sync.OnceValue(func() *MembershipService {
	return &MembershipService{
		getMembershipRepository(),
		getWorkspaceRepository(),
		users_services.GetUserService(),
		audit_logs.GetAuditLogService(),
		getWorkspaceService(),
		users_services.GetSettingsService(),
		email.GetInstanceSender(),
		logger.GetLogger(),
	}
})

func GetWorkspaceService() *WorkspaceService {
	return getWorkspaceService()
}

func GetMembershipService() *MembershipService {
	return getMembershipService()
}
