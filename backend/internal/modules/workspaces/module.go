// Package workspaces owns workspaces, their members, the workspace roles, invitations to a
// workspace, ownership transfer and the audit log views, which show the names of workspaces. Other
// modules reach it only through this file.
package workspaces

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/access"
	"databasus-backend/internal/modules/workspaces/internal/auditlog"
	workspaceshttpapi "databasus-backend/internal/modules/workspaces/internal/httpapi"
	"databasus-backend/internal/modules/workspaces/internal/members"
	"databasus-backend/internal/modules/workspaces/internal/module"
	"databasus-backend/internal/modules/workspaces/internal/workspace"
)

type (
	// Role is the role of a member in a workspace. RoleAdmin is a workspace admin; an instance admin
	// is a user with the global role users.RoleAdmin.
	Role = access.Role
	// Action is what an actor is about to do in a workspace. ActionEditContent covers the databases,
	// storages and notifiers of the workspace and everything attached to them.
	Action = access.Action
	// Workspace is a workspace as other modules see it.
	Workspace = workspace.Workspace
	// WorkspaceDeleting is published inside the deletion's transaction, before the workspace row
	// goes.
	WorkspaceDeleting = workspace.Deleting
	// WorkspaceDeletingHandler runs in the deletion's transaction; an error rolls the deletion back.
	WorkspaceDeletingHandler = workspace.DeletingHandler
)

const (
	RoleOwner  = access.RoleOwner
	RoleAdmin  = access.RoleAdmin
	RoleMember = access.RoleMember
	RoleViewer = access.RoleViewer

	ActionView              = access.ActionView
	ActionEditContent       = access.ActionEditContent
	ActionUpdateWorkspace   = access.ActionUpdateWorkspace
	ActionManageMembers     = access.ActionManageMembers
	ActionManageAdmins      = access.ActionManageAdmins
	ActionTransferOwnership = access.ActionTransferOwnership
	ActionDeleteWorkspace   = access.ActionDeleteWorkspace
)

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// Authorize answers not_found when the actor is not a member of the workspace or the workspace
	// does not exist, and permission_denied when the actor's role does not allow the action. An
	// instance admin passes every check.
	Authorize(ctx context.Context, actor users.Actor, workspaceID uuid.UUID, action Action) error
	// GetWorkspaces leaves out the IDs that match no workspace.
	GetWorkspaces(ctx context.Context, workspaceIDs []uuid.UUID) ([]Workspace, error)
	// SubscribeToWorkspaceDeleting is called by internal/app while it builds the application. The
	// handlers run in the order they subscribed.
	SubscribeToWorkspaceDeleting(handler WorkspaceDeletingHandler)
}

type Deps struct {
	DB         *gorm.DB
	Transactor *database.Transactor
	Clock      clock.Source
	Logger     *slog.Logger
	Mailer     mailer.Sender
	AuditLog   *audit.Log
	Users      users.Module
	// PublicURL is the address the invitation email links to, empty when the operator set none.
	PublicURL string
}

func New(deps Deps) Module {
	authorizer := access.NewAuthorizer(access.NewRepositories(deps.DB))

	workspaceService := workspace.NewService(workspace.Deps{
		Repositories: workspace.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Authorizer:   authorizer,
	})

	memberService := members.NewService(members.Deps{
		Repositories: members.NewRepositories(deps.DB),
		Transactor:   deps.Transactor,
		Clock:        deps.Clock,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Authorizer:   authorizer,
		Workspaces:   workspaceService,
		Mailer:       deps.Mailer,
		Logger:       deps.Logger,
		PublicURL:    deps.PublicURL,
	})

	auditLogService := auditlog.NewService(auditlog.Deps{
		AuditLog:   deps.AuditLog,
		Authorizer: authorizer,
		Workspaces: workspaceService,
	})

	return module.New(module.Deps{
		Authorizer: authorizer,
		Workspaces: workspaceService,
		Handler: workspaceshttpapi.NewHandler(workspaceshttpapi.Deps{
			Workspaces: workspaceService,
			Members:    memberService,
			AuditLog:   auditLogService,
		}),
	})
}
