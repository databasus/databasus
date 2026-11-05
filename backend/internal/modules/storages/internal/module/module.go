// Package module implements the storages.Module contract on top of the module's services.
package module

import (
	"context"
	"io"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/storages/internal/files"
	storageshttpapi "databasus-backend/internal/modules/storages/internal/httpapi"
	"databasus-backend/internal/modules/storages/internal/storage"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Storages    *storage.Service
	Transfers   *storage.TransferService
	Files       *files.Store
	DeletionJob *files.DeletionJob
	Handler     *storageshttpapi.Handler
}

type Module struct {
	storages    *storage.Service
	transfers   *storage.TransferService
	files       *files.Store
	deletionJob *files.DeletionJob
	handler     *storageshttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{
		storages:    deps.Storages,
		transfers:   deps.Transfers,
		files:       deps.Files,
		deletionJob: deps.DeletionJob,
		handler:     deps.Handler,
	}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	storageshttpapi.RegisterHandlersWithOptions(
		routes,
		storageshttpapi.NewStrictHandlerWithOptions(m.handler, nil, storageshttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		storageshttpapi.GinServerOptions{
			BaseURL:      "",
			Middlewares:  []storageshttpapi.MiddlewareFunc{storageshttpapi.NewRequestValidator(hooks)},
			ErrorHandler: hooks.HandleParameterError,
		},
	)
}

func (m *Module) WriteFile(
	ctx context.Context,
	reference files.StoredFileReference,
	file io.Reader,
) (files.WriteReceipt, error) {
	return m.files.WriteFile(ctx, reference, file)
}

func (m *Module) StartBackgroundWrite(ctx context.Context, spec files.BackgroundWriteSpec) files.BackgroundWrite {
	return m.files.StartBackgroundWrite(ctx, spec)
}

func (m *Module) ConfirmFileWrites(ctx context.Context, receipts []files.WriteReceipt) error {
	return m.files.ConfirmFileWrites(ctx, receipts)
}

func (m *Module) RequestFileDeletions(ctx context.Context, references []files.StoredFileReference) error {
	return m.files.RequestFileDeletions(ctx, references)
}

func (m *Module) GetStorageReader(ctx context.Context, storageID uuid.UUID) (files.StorageReader, error) {
	return m.files.GetStorageReader(ctx, storageID)
}

func (m *Module) GetStorages(ctx context.Context, storageIDs []uuid.UUID) ([]storage.Storage, error) {
	return m.storages.GetStorages(ctx, storageIDs)
}

func (m *Module) GetStorageBlockingTransfer(ctx context.Context, storageID uuid.UUID) (storage.Storage, bool, error) {
	return m.storages.GetStorageBlockingTransfer(ctx, storageID)
}

func (m *Module) GetStorageTypes(ctx context.Context) ([]storage.Type, error) {
	return m.storages.GetStorageTypes(ctx)
}

func (m *Module) TransferStorage(ctx context.Context, actor users.Actor, transfer storage.Transfer) error {
	return m.transfers.TransferStorage(ctx, actor, transfer)
}

func (m *Module) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	return m.storages.OnWorkspaceDeleting(ctx, event)
}

func (m *Module) SubscribeToStorageTransferring(handler storage.TransferringHandler) {
	m.transfers.SubscribeToTransferring(handler)
}

func (m *Module) GetJobs() []background.Job {
	return []background.Job{m.deletionJob.GetJob()}
}

func (m *Module) RequestScopedFileDeletions(ctx context.Context, request files.FileDeletionRequest) error {
	return m.files.RequestScopedFileDeletions(ctx, request)
}

func (m *Module) GetCleanupSummaries(
	ctx context.Context,
	scopes []files.CleanupScope,
) (map[files.CleanupScope]files.CleanupSummary, error) {
	return m.files.GetCleanupSummaries(ctx, scopes)
}

func (m *Module) GetPendingFileReferences(ctx context.Context, prefix string) ([]files.StoredFileReference, error) {
	return m.files.GetPendingFileReferences(ctx, prefix)
}

func (m *Module) RequestDiscardedFileDeletions(ctx context.Context, request files.DiscardedFiles) error {
	return m.files.RequestDiscardedFileDeletions(ctx, request)
}
