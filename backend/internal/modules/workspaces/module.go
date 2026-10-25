// Package workspaces owns workspaces, their members and the workspace roles. Other modules reach it
// only through this file.
package workspaces

// Role is the role of a member in a workspace. RoleAdmin is a workspace admin; an instance admin
// is a user with the global role users.RoleAdmin.
type Role string

const (
	RoleOwner  Role = "WORKSPACE_OWNER"
	RoleAdmin  Role = "WORKSPACE_ADMIN"
	RoleMember Role = "WORKSPACE_MEMBER"
	RoleViewer Role = "WORKSPACE_VIEWER"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

// Action is what an actor is about to do in a workspace. ActionEditContent covers the databases,
// storages and notifiers of the workspace and everything attached to them.
type Action string

const (
	ActionView              Action = "VIEW"
	ActionEditContent       Action = "EDIT_CONTENT"
	ActionUpdateWorkspace   Action = "UPDATE_WORKSPACE"
	ActionManageMembers     Action = "MANAGE_MEMBERS"
	ActionManageAdmins      Action = "MANAGE_ADMINS"
	ActionTransferOwnership Action = "TRANSFER_OWNERSHIP"
	ActionDeleteWorkspace   Action = "DELETE_WORKSPACE"
)
