// Package module implements the backups.Module contract on top of the module's services.
package module

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/backups/internal/config"
	backupshttpapi "databasus-backend/internal/modules/backups/internal/httpapi"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
)

type Deps struct {
	Configs *config.Service
	Handler *backupshttpapi.Handler
}

type Module struct {
	configs *config.Service
	handler *backupshttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{configs: deps.Configs, handler: deps.Handler}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	backupshttpapi.RegisterHandlersWithOptions(
		routes,
		backupshttpapi.NewStrictHandlerWithOptions(m.handler, nil, backupshttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		backupshttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}

func (m *Module) LockAndGetLogicalBackupConfig(
	ctx context.Context,
	databaseID uuid.UUID,
) (config.LogicalBackupConfig, error) {
	return m.configs.LockAndGetLogicalBackupConfig(ctx, databaseID)
}

func (m *Module) GetLogicalBackupConfig(ctx context.Context, databaseID uuid.UUID) (config.LogicalBackupConfig, error) {
	return m.configs.GetLogicalBackupConfig(ctx, databaseID)
}

func (m *Module) GetPhysicalBackupConfig(
	ctx context.Context,
	databaseID uuid.UUID,
) (config.PhysicalBackupConfig, error) {
	return m.configs.GetPhysicalBackupConfig(ctx, databaseID)
}

func (m *Module) ListEnabledLogicalBackupConfigs(ctx context.Context) ([]config.LogicalBackupConfig, error) {
	return m.configs.ListEnabledLogicalBackupConfigs(ctx)
}

func (m *Module) ListEnabledPhysicalBackupConfigs(ctx context.Context) ([]config.PhysicalBackupConfig, error) {
	return m.configs.ListEnabledPhysicalBackupConfigs(ctx)
}

func (m *Module) SaveBackupConfig(
	ctx context.Context,
	actor users.Actor,
	savedConfig config.AnyBackupConfig,
) (config.AnyBackupConfig, error) {
	return m.configs.SaveBackupConfig(ctx, actor, savedConfig)
}

func (m *Module) RequestFullBackup(ctx context.Context, databaseID uuid.UUID) error {
	return m.configs.RequestFullBackup(ctx, databaseID)
}

func (m *Module) ClearFullBackupRequest(ctx context.Context, databaseID uuid.UUID, requestedAt *time.Time) error {
	return m.configs.ClearFullBackupRequest(ctx, databaseID, requestedAt)
}

func (m *Module) RequestIncrementalBackup(ctx context.Context, databaseID uuid.UUID) error {
	return m.configs.RequestIncrementalBackup(ctx, databaseID)
}

func (m *Module) ClearIncrementalBackupRequest(
	ctx context.Context,
	databaseID uuid.UUID,
	requestedAt *time.Time,
) error {
	return m.configs.ClearIncrementalBackupRequest(ctx, databaseID, requestedAt)
}

func (m *Module) OnStorageTransferring(ctx context.Context, event storages.StorageTransferring) error {
	return m.configs.OnStorageTransferring(ctx, event)
}

func (m *Module) OnDatabaseCopied(ctx context.Context, event databases.DatabaseCopied) error {
	return m.configs.OnDatabaseCopied(ctx, event)
}

func (m *Module) OnDatabaseBackupTypeChanged(ctx context.Context, event databases.DatabaseBackupTypeChanged) error {
	return m.configs.OnDatabaseBackupTypeChanged(ctx, event)
}

func (m *Module) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	return m.configs.OnDatabaseDeleting(ctx, event)
}

func (m *Module) SubscribeToLogicalBackupConfigStorageChanging(
	handler config.LogicalBackupConfigStorageChangingHandler,
) {
	m.configs.SubscribeToLogicalBackupConfigStorageChanging(handler)
}

func (m *Module) SubscribeToPhysicalBackupConfigSaved(handler config.PhysicalBackupConfigSavedHandler) {
	m.configs.SubscribeToPhysicalBackupConfigSaved(handler)
}
