// Package module implements the workspaces.Module contract on top of the module's services.
package module

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/access"
	workspaceshttpapi "databasus-backend/internal/modules/workspaces/internal/httpapi"
	"databasus-backend/internal/modules/workspaces/internal/workspace"
)

type Deps struct {
	Authorizer *access.Authorizer
	Workspaces *workspace.Service
	Handler    *workspaceshttpapi.Handler
}

type Module struct {
	authorizer *access.Authorizer
	workspaces *workspace.Service
	handler    *workspaceshttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{authorizer: deps.Authorizer, workspaces: deps.Workspaces, handler: deps.Handler}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	workspaceshttpapi.RegisterHandlersWithOptions(
		routes,
		workspaceshttpapi.NewStrictHandlerWithOptions(m.handler, nil, workspaceshttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		workspaceshttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}

func (m *Module) Authorize(
	ctx context.Context,
	actor users.Actor,
	workspaceID uuid.UUID,
	action access.Action,
) error {
	return m.authorizer.Authorize(ctx, actor, workspaceID, action)
}

func (m *Module) GetWorkspaces(ctx context.Context, workspaceIDs []uuid.UUID) ([]workspace.Workspace, error) {
	return m.workspaces.GetWorkspaces(ctx, workspaceIDs)
}

func (m *Module) SubscribeToWorkspaceDeleting(handler workspace.DeletingHandler) {
	m.workspaces.SubscribeToDeleting(handler)
}
