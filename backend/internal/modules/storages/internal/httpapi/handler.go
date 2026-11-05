package httpapi

import (
	"context"

	"databasus-backend/internal/modules/storages/internal/storage"
	"databasus-backend/internal/modules/users"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	Storages        *storage.Service
	Transfers       *storage.TransferService
	ConnectionTests *storage.ConnectionTestService
}

type Handler struct {
	storages        *storage.Service
	transfers       *storage.TransferService
	connectionTests *storage.ConnectionTestService
}

func NewHandler(deps Deps) *Handler {
	return &Handler{storages: deps.Storages, transfers: deps.Transfers, connectionTests: deps.ConnectionTests}
}

func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(ctx)
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
