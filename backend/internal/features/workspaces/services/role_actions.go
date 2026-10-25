package workspaces_services

import (
	"slices"

	"databasus-backend/internal/modules/workspaces"
)

var actionsAllowedByRole = map[workspaces.Role][]workspaces.Action{
	workspaces.RoleOwner: {
		workspaces.ActionView,
		workspaces.ActionEditContent,
		workspaces.ActionUpdateWorkspace,
		workspaces.ActionManageMembers,
		workspaces.ActionManageAdmins,
		workspaces.ActionTransferOwnership,
		workspaces.ActionDeleteWorkspace,
	},
	workspaces.RoleAdmin: {
		workspaces.ActionView,
		workspaces.ActionEditContent,
		workspaces.ActionUpdateWorkspace,
		workspaces.ActionManageMembers,
	},
	workspaces.RoleMember: {
		workspaces.ActionView,
		workspaces.ActionEditContent,
	},
	workspaces.RoleViewer: {
		workspaces.ActionView,
	},
}

func isRoleAllowed(role workspaces.Role, action workspaces.Action) bool {
	return slices.Contains(actionsAllowedByRole[role], action)
}
