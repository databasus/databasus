// Package module implements the backups.Module contract on top of the module's services.
package module

import (
	"context"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/download"
	backupshttpapi "databasus-backend/internal/modules/backups/internal/httpapi"
	"databasus-backend/internal/modules/backups/internal/logical"
	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/backups/internal/restore"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type Deps struct {
	Physical       *physical.Service
	Configs        *config.Service
	LogicalBackups *logical.Service
	Downloads      *download.Service
	Restores       *restore.Service
	UserStreams    *download.UserStreams
	Runs           *background.Group
	Handler        *backupshttpapi.Handler
}

type Module struct {
	physical       *physical.Service
	configs        *config.Service
	logicalBackups *logical.Service
	downloads      *download.Service
	restores       *restore.Service
	userStreams    *download.UserStreams
	runs           *background.Group
	handler        *backupshttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{
		physical:       deps.Physical,
		configs:        deps.Configs,
		logicalBackups: deps.LogicalBackups,
		downloads:      deps.Downloads,
		restores:       deps.Restores,
		userStreams:    deps.UserStreams,
		runs:           deps.Runs,
		handler:        deps.Handler,
	}
}

func (m *Module) GetLogicalBackups(ctx context.Context, backupIDs []uuid.UUID) ([]logical.LogicalBackup, error) {
	return m.logicalBackups.GetBackups(ctx, backupIDs)
}

func (m *Module) GetLatestCompletedLogicalBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) (logical.LogicalBackup, bool, error) {
	return m.logicalBackups.GetLatestCompletedBackup(ctx, databaseID)
}

func (m *Module) OpenLogicalBackupFile(ctx context.Context, backupID uuid.UUID) (io.ReadCloser, error) {
	return m.logicalBackups.OpenBackupFileByID(ctx, backupID)
}

func (m *Module) SetLogicalBackupVerificationStatus(
	ctx context.Context,
	backupID uuid.UUID,
	status logical.VerificationStatus,
) error {
	return m.logicalBackups.SetVerificationStatus(ctx, backupID, status)
}

func (m *Module) SubscribeToLogicalBackupCompleted(handler logical.BackupCompletedHandler) {
	m.logicalBackups.SubscribeToBackupCompleted(handler)
}

func (m *Module) SubscribeToLogicalBackupsDeleting(handler logical.BackupsDeletingHandler) {
	m.logicalBackups.SubscribeToBackupsDeleting(handler)
}

// Start fails the restores and the backups the previous run left in progress.
func (m *Module) Start(ctx context.Context) error {
	err := m.restores.Start(ctx)
	if err != nil {
		return err
	}

	if err := m.logicalBackups.Start(ctx); err != nil {
		return err
	}
	return m.physical.Start(ctx)
}

func (m *Module) GetJobs() []background.Job {
	return append([]background.Job{
		m.logicalBackups.GetSchedulerJob(),
		m.logicalBackups.GetRetentionJob(),
		m.downloads.GetTokenCleanupJob(),
	}, m.physical.GetJobs()...)
}

func (m *Module) WaitForRuns(ctx context.Context) error {
	return m.runs.Wait(ctx)
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
	if err := m.configs.OnDatabaseBackupTypeChanged(ctx, event); err != nil {
		return err
	}
	return m.physical.OnDatabaseBackupTypeChanged(ctx, event)
}

// OnDatabaseDeleting removes the logical backups first, so their refusal of a running backup or
// restore comes before anything else of the database goes.
func (m *Module) OnDatabaseDeleting(ctx context.Context, event databases.DatabaseDeleting) error {
	if err := m.physical.OnDatabaseDeleting(ctx, event); err != nil {
		return err
	}
	err := m.logicalBackups.OnDatabaseDeleting(ctx, event)
	if err != nil {
		return err
	}

	return m.configs.OnDatabaseDeleting(ctx, event)
}

func (m *Module) GetLatestCompletedLogicalBackups(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]logical.LogicalBackup, error) {
	return m.logicalBackups.GetLatestCompletedBackups(ctx, ids)
}
