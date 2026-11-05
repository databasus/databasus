package httpapi

import (
	"context"

	"databasus-backend/internal/modules/databases/internal/connection"
	"databasus-backend/internal/modules/databases/internal/limiteduser"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
	"databasus-backend/internal/modules/users"
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

func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(ctx)
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
