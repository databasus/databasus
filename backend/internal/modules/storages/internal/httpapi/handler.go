package httpapi

import (
	"databasus-backend/internal/modules/storages/internal/storage"
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
