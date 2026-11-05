package httpapi

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/configsave"
	"databasus-backend/internal/modules/backups/internal/download"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/backups/internal/restore"
	"databasus-backend/internal/modules/backups/internal/transfer"
	"databasus-backend/internal/modules/users"
)

var _ StrictServerInterface = (*Handler)(nil)

type Deps struct {
	ConfigSaves     *configsave.Service
	PhysicalBackups *physical.Service
	Configs         *config.Service
	Transfers       *transfer.Service
	LogicalBackups  *logical.Service
	Downloads       *download.Service
	Restores        *restore.Service
}

type Handler struct {
	configSaves     *configsave.Service
	physicalBackups *physical.Service
	configs         *config.Service
	transfers       *transfer.Service
	logicalBackups  *logical.Service
	downloads       *download.Service
	restores        *restore.Service
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		configSaves:     deps.ConfigSaves,
		physicalBackups: deps.PhysicalBackups,
		configs:         deps.Configs,
		transfers:       deps.Transfers,
		logicalBackups:  deps.LogicalBackups,
		downloads:       deps.Downloads,
		restores:        deps.Restores,
	}
}

func getActor(ctx context.Context) (users.Actor, error) {
	actor, isFound := users.GetActor(ctx)
	if !isFound {
		return users.Actor{}, errNoActor
	}

	return actor, nil
}
