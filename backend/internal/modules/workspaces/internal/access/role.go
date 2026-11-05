// Package access holds the workspace roles, what each role allows and Authorize, the one check of
// an actor's access to a workspace.
package access

import "slices"

// Role is the role of a member in a workspace. RoleAdmin is a workspace admin; an instance admin
// is a user with the global role users.RoleAdmin.
type Role string

const (
	RoleOwner  Role = "WORKSPACE_OWNER"
	RoleAdmin  Role = "WORKSPACE_ADMIN"
	RoleMember Role = "WORKSPACE_MEMBER"
	RoleViewer Role = "WORKSPACE_VIEWER"
)

//nolint:gochecknoglobals // The fixed role/action matrix is shared by authorization checks to avoid repeated allocation; it is never mutated.
var actionsAllowedByRole = map[Role][]Action{
	RoleOwner: {
		ActionView,
		ActionEditContent,
		ActionUpdateWorkspace,
		ActionManageMembers,
		ActionManageAdmins,
		ActionTransferOwnership,
		ActionDeleteWorkspace,
	},
	RoleAdmin: {
		ActionView,
		ActionEditContent,
		ActionUpdateWorkspace,
		ActionManageMembers,
	},
	RoleMember: {
		ActionView,
		ActionEditContent,
	},
	RoleViewer: {
		ActionView,
	},
}

func (r Role) IsValid() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

// IsAssignable reports whether a request may set the role; the owner role changes hands only
// through an ownership transfer.
func (r Role) IsAssignable() bool {
	return r.IsValid() && r != RoleOwner
}

func (r Role) IsAllowed(action Action) bool {
	return slices.Contains(actionsAllowedByRole[r], action)
}
