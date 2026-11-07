package httpapi

import (
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
