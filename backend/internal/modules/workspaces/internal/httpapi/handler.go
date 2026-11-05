package httpapi

import (
	"context"

	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces/internal/auditlog"
	"databasus-backend/internal/modules/workspaces/internal/members"
	"databasus-backend/internal/modules/workspaces/internal/workspace"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Workspaces *workspace.Service
	Members    *members.Service
	AuditLog   *auditlog.Service
}

type Handler struct {
	workspaces *workspace.Service
	members    *members.Service
	auditLog   *auditlog.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{workspaces: deps.Workspaces, members: deps.Members, auditLog: deps.AuditLog}
}

func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(ctx)
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
