// Package storages owns the places backups are kept: the storages of workspaces, their provider
// settings, their connection tests and every write, read and deletion of a stored file. Other
// modules reach it only through this file.
package storages

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/storages/internal/files"
	storageshttpapi "databasus-backend/internal/modules/storages/internal/httpapi"
	"databasus-backend/internal/modules/storages/internal/module"
	"databasus-backend/internal/modules/storages/internal/providers"
	"databasus-backend/internal/modules/storages/internal/providers/azureblob"
	"databasus-backend/internal/modules/storages/internal/providers/ftp"
	"databasus-backend/internal/modules/storages/internal/providers/googledrive"
	"databasus-backend/internal/modules/storages/internal/providers/local"
	"databasus-backend/internal/modules/storages/internal/providers/nas"
	"databasus-backend/internal/modules/storages/internal/providers/rclone"
	"databasus-backend/internal/modules/storages/internal/providers/s3"
	"databasus-backend/internal/modules/storages/internal/providers/sftp"
	"databasus-backend/internal/modules/storages/internal/storage"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type (
	// Type is where a storage keeps its files. It is fixed when the storage is created.
	Type = storage.Type
	// Storage is a place for backup files as other modules see it, without its settings.
	Storage = storage.Storage
	// StorageTransfer names the storage to move and the workspace it moves to.
	StorageTransfer = storage.Transfer
	// StorageTransferring is published inside the transfer's transaction, before the storage moves.
	StorageTransferring = storage.Transferring
	// StorageTransferringHandler runs in the transfer's transaction; an error refuses the transfer.
	StorageTransferringHandler = storage.TransferringHandler
	// StoredFileReference names a file in a storage.
	StoredFileReference = files.StoredFileReference
	// WriteReceipt is the permission to publish one uploaded file in a catalog transaction.
	WriteReceipt = files.WriteReceipt
	// StorageReader opens files of one storage.
	StorageReader = files.StorageReader
	// BackgroundWrite is an upload running beside the producer that feeds it through a pipe.
	BackgroundWrite = files.BackgroundWrite
	// BackgroundWriteSpec names the file, the pipe the upload reads and the callback of a failure.
	BackgroundWriteSpec = files.BackgroundWriteSpec
	// LocalFolders are the folders a local storage writes through.
	LocalFolders = providers.LocalFolders
	// FileTimings are the deadlines of the file lifecycle.
	FileTimings         = files.Timings
	DiscardedFiles      = files.DiscardedFiles
	FileDeletionRequest = files.FileDeletionRequest
	CleanupScope        = files.CleanupScope
	CleanupSummary      = files.CleanupSummary
)

const (
	TypeLocal       = providers.TypeLocal
	TypeS3          = providers.TypeS3
	TypeGoogleDrive = providers.TypeGoogleDrive
	TypeNAS         = providers.TypeNAS
	TypeAzureBlob   = providers.TypeAzureBlob
	TypeFTP         = providers.TypeFTP
	TypeSFTP        = providers.TypeSFTP
	TypeRclone      = providers.TypeRclone
)

// ProductionFileTimings is the policy of a running instance; tests pass shorter timings.
func ProductionFileTimings() FileTimings {
	return FileTimings{
		CommitWindow:   15 * time.Minute,
		AttemptLease:   2 * time.Minute,
		RetryBaseDelay: 30 * time.Second,
		RetryMaxDelay:  time.Hour,
		WorkerTick:     time.Minute,
		ClaimBatchSize: 50,
	}
}

// LocalStorageProbe is what the local storage probe needs; it runs without Deps.
type LocalStorageProbe struct {
	Folders LocalFolders
	Logger  *slog.Logger
}

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// WriteFile records the obligation to delete the file before the first byte leaves.
	WriteFile(ctx context.Context, reference StoredFileReference, file io.Reader) (WriteReceipt, error)
	StartBackgroundWrite(ctx context.Context, spec BackgroundWriteSpec) BackgroundWrite
	// ConfirmFileWrites and RequestFileDeletions refuse to run outside a transaction in ctx.
	ConfirmFileWrites(ctx context.Context, receipts []WriteReceipt) error
	RequestFileDeletions(ctx context.Context, references []StoredFileReference) error
	RequestDiscardedFileDeletions(ctx context.Context, request DiscardedFiles) error
	RequestScopedFileDeletions(ctx context.Context, request FileDeletionRequest) error
	GetCleanupSummaries(ctx context.Context, scopes []CleanupScope) (map[CleanupScope]CleanupSummary, error)
	GetPendingFileReferences(ctx context.Context, prefix string) ([]StoredFileReference, error)
	GetStorageReader(ctx context.Context, storageID uuid.UUID) (StorageReader, error)

	// GetStorages leaves out the IDs that match no storage.
	GetStorages(ctx context.Context, storageIDs []uuid.UUID) ([]Storage, error)
	// GetStorageBlockingTransfer reads the storage in the caller's transaction and makes a transfer of
	// it wait until that transaction ends, so the workspace the caller checked stays the storage's.
	GetStorageBlockingTransfer(ctx context.Context, storageID uuid.UUID) (Storage, bool, error)
	// GetStorageTypes returns the type of every storage of the instance, for telemetry.
	GetStorageTypes(ctx context.Context) ([]Type, error)
	TransferStorage(ctx context.Context, actor users.Actor, transfer StorageTransfer) error
	OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error
	// SubscribeToStorageTransferring is called by internal/app while it builds the application. The
	// handlers run in the order they subscribed.
	SubscribeToStorageTransferring(handler StorageTransferringHandler)
	GetJobs() []background.Job
}

type Deps struct {
	DB             *gorm.DB
	Transactor     *database.Transactor
	Logger         *slog.Logger
	AuditLog       *audit.Log
	FieldEncryptor *crypto.FieldEncryptor
	Users          users.Module
	Workspaces     workspaces.Module
	Clock          clock.Source
	LocalFolders   LocalFolders
	FileTimings    FileTimings
}

func New(deps Deps) Module {
	// One group runs the uploads of background writes and the NAS chunk writes, so every goroutine of
	// the module turns a panic into an error.
	writes := background.New(background.Deps{Logger: deps.Logger})

	registry := providers.NewRegistry(
		local.New(local.Deps{Repositories: local.NewRepositories(deps.DB), Folders: deps.LocalFolders}),
		s3.New(s3.Deps{
			Repositories:   s3.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		googledrive.New(googledrive.Deps{
			Repositories:   googledrive.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			Logger:         deps.Logger,
		}),
		nas.New(nas.Deps{
			Repositories:   nas.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			ChunkWrites:    writes,
		}),
		azureblob.New(azureblob.Deps{
			Repositories:   azureblob.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		ftp.New(ftp.Deps{
			Repositories:   ftp.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		sftp.New(sftp.Deps{
			Repositories:   sftp.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
		rclone.New(rclone.Deps{
			Repositories:   rclone.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
		}),
	)

	repositories := storage.NewRepositories(deps.DB)
	settingsReader := storage.NewSettingsReader(storage.SettingsReaderDeps{
		Repositories: repositories,
		Registry:     registry,
	})

	fileDeps := files.Deps{
		Repository:      files.NewPendingDeletionRepository(deps.DB),
		WriteProtection: files.NewWriteProtectionRepository(deps.DB),
		Transactor:      deps.Transactor,
		Locator:         files.NewLocator(files.LocatorDeps{Registry: registry, Settings: settingsReader}),
		Logger:          deps.Logger,
		Clock:           deps.Clock,
		Timings:         deps.FileTimings,
		Uploads:         writes,
	}
	fileStore := files.NewStore(fileDeps)
	deletionJob := files.NewDeletionJob(fileStore, fileDeps)

	storageService := storage.NewService(storage.Deps{
		Repositories:  repositories,
		Settings:      settingsReader,
		Registry:      registry,
		FileDeletions: fileStore,
		Transactor:    deps.Transactor,
		Logger:        deps.Logger,
		AuditLog:      deps.AuditLog,
		Users:         deps.Users,
		Workspaces:    deps.Workspaces,
	})

	transferService := storage.NewTransferService(storage.TransferServiceDeps{
		Storages:     storageService,
		Repositories: repositories,
		Transactor:   deps.Transactor,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Workspaces:   deps.Workspaces,
	})

	return module.New(module.Deps{
		Storages:    storageService,
		Transfers:   transferService,
		Files:       fileStore,
		DeletionJob: deletionJob,
		Handler: storageshttpapi.NewHandler(storageshttpapi.Deps{
			Storages:  storageService,
			Transfers: transferService,
			ConnectionTests: storage.NewConnectionTestService(storage.ConnectionTestServiceDeps{
				Storages:   storageService,
				Registry:   registry,
				Workspaces: deps.Workspaces,
			}),
		}),
	})
}

// ProbeLocalStorage saves and deletes a file through local storage. Container startup runs it before
// the internal database exists, so it needs no Deps.
func ProbeLocalStorage(ctx context.Context, probe LocalStorageProbe) error {
	return local.Probe(ctx, local.ProbeSpec{Folders: probe.Folders, Logger: probe.Logger})
}
