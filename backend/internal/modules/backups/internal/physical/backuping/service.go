package backuping

import (
	"log/slog"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	ExecutionClock     clock.Source
	FullBackups        *catalog.FullBackupRepository
	IncrementalBackups *catalog.IncrementalBackupRepository
	InFlightBackups    *catalog.InFlightBackupRepository
	HistoryFiles       *catalog.WalHistoryRepository
	Repository         *catalog.Repository
	Configs            *config.Service
	Chains             *chains.Service
	Cancellations      *background.CancellationRegistry
	Runs               *background.Group
	Storages           storages.Module
	Databases          databases.Module
	Workspaces         workspaces.Module
	Notifiers          notifiers.Module
	Logger             *slog.Logger
	Clock              clock.Source
	Transactor         *database.Transactor
	MasterKey          *crypto.MasterKey
	ClientToolsDir     string
}

type Service struct {
	Backuper  *Backuper
	Scheduler *Scheduler
	Canceller *Canceller
}

func New(deps Deps) *Service {
	backuper := &Backuper{
		databases: deps.Databases, workspaceService: deps.Workspaces,
		fullRepo: deps.FullBackups, incrRepo: deps.IncrementalBackups,
		inFlightRepo: deps.InFlightBackups, historyRepo: deps.HistoryFiles,
		configs: deps.Configs, storages: deps.Storages, notificationSender: deps.Notifiers,
		taskCancellationRegistry: deps.Cancellations, masterKey: deps.MasterKey, logger: deps.Logger,
		fullExecutor: NewFullExecutor(), incrExecutor: NewIncrementalExecutor(),
		clock: deps.Clock, executionClock: deps.ExecutionClock, transactor: deps.Transactor,
		clientTools: pgshared.ClientTools{Root: deps.ClientToolsDir}, runs: deps.Runs,
	}
	scheduler := &Scheduler{
		fullRepo: deps.FullBackups, incrRepo: deps.IncrementalBackups, inFlightRepo: deps.InFlightBackups,
		repository: deps.Repository, configs: deps.Configs, chainViewService: deps.Chains,
		taskCancellationRegistry: deps.Cancellations, storages: deps.Storages, backuper: backuper,
		logger: deps.Logger, clock: deps.Clock, transactor: deps.Transactor,
	}
	return &Service{
		Backuper:  backuper,
		Scheduler: scheduler,
		Canceller: &Canceller{
			inFlightRepo:             deps.InFlightBackups,
			taskCancellationRegistry: deps.Cancellations,
			logger:                   deps.Logger,
		},
	}
}
