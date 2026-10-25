package access

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
