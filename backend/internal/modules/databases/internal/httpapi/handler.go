package httpapi

import (
	"databasus-backend/internal/modules/databases/internal/connection"
	"databasus-backend/internal/modules/databases/internal/limiteduser"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Databases    *userdatabase.Service
	Connections  *connection.Service
	LimitedUsers *limiteduser.Service
}

type Handler struct {
	databases    *userdatabase.Service
	connections  *connection.Service
	limitedUsers *limiteduser.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{databases: deps.Databases, connections: deps.Connections, limitedUsers: deps.LimitedUsers}
}
