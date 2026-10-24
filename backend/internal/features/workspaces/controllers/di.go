package workspaces_controllers

import (
	"sync"

	workspaces_services "databasus-backend/internal/features/workspaces/services"
)

var getWorkspaceController = sync.OnceValue(func() *WorkspaceController {
	return &WorkspaceController{
		workspaces_services.GetWorkspaceService(),
	}
})

var getMembershipController = sync.OnceValue(func() *MembershipController {
	return &MembershipController{
		workspaces_services.GetMembershipService(),
	}
})

func GetWorkspaceController() *WorkspaceController {
	return getWorkspaceController()
}

func GetMembershipController() *MembershipController {
	return getMembershipController()
}
