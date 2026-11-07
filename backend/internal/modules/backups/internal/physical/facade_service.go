package physical

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/cache"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/auditing"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/download"
	"databasus-backend/internal/modules/backups/internal/physical/backuping"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/restorestream"
	"databasus-backend/internal/modules/backups/internal/physical/retention"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
	"databasus-backend/internal/modules/backups/internal/physical/walstreaming"
	"databasus-backend/internal/modules/backups/internal/usage"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Usage            *usage.Service
	ExecutionClock   clock.Source
	Repository       *catalog.Repository
	Backups          *catalog.BackupRepository
	WalSegments      *catalog.WalSegmentRepository
	WalHistory       *catalog.WalHistoryRepository
	WalStreamers     *catalog.WalStreamerRepository
	WalStreamSamples *catalog.WalStreamSampleRepository
	Configs          *config.Service
	Cancellations    *background.CancellationRegistry
	Runs             *background.Group
	UserStreams      *download.UserStreams
	Cache            cache.Store
	Databases        databases.Module
	Storages         storages.Module
	Workspaces       workspaces.Module
	Notifiers        notifiers.Module
	Auditing         auditing.Recorder
	Transactor       *database.Transactor
	Clock            clock.Source
	MasterKey        *crypto.MasterKey
	Logger           *slog.Logger
	ClientToolsDir   string
	WalQueueDir      string
	WalStreamTimings walstreaming.Timings
}

func New(deps Deps) *Service {
	lifecycle := catalog.NewBackupLifecycle(catalog.BackupLifecycleDeps{
		Configs: deps.Configs, Backups: deps.Backups, Databases: deps.Databases, Storages: deps.Storages,
		Transactor: deps.Transactor, Clock: deps.Clock,
	})
	chainsService := chains.New(
		chains.Deps{
			Repository:  deps.Repository,
			Backups:     deps.Backups,
			WalSegments: deps.WalSegments,
			WalHistory:  deps.WalHistory,
			Storages:    deps.Storages,
			Transactor:  deps.Transactor,
		},
	)
	backupsService := backuping.New(
		backuping.Deps{
			Lifecycle:      lifecycle,
			ExecutionClock: deps.ExecutionClock,
			Backups:        deps.Backups,
			HistoryFiles:   deps.WalHistory,
			Repository:     deps.Repository,
			Configs:        deps.Configs,
			Chains:         chainsService,
			Cancellations:  deps.Cancellations,
			Runs:           deps.Runs,
			Storages:       deps.Storages,
			Databases:      deps.Databases,
			Workspaces:     deps.Workspaces,
			Notifiers:      deps.Notifiers,
			Logger:         deps.Logger,
			Clock:          deps.Clock,
			Transactor:     deps.Transactor,
			MasterKey:      deps.MasterKey,
			ClientToolsDir: deps.ClientToolsDir,
		},
	)
	streamManager := walstreaming.New(
		walstreaming.Deps{
			WalSegments:    deps.WalSegments,
			HistoryFiles:   deps.WalHistory,
			Streamers:      deps.WalStreamers,
			Samples:        deps.WalStreamSamples,
			Configs:        deps.Configs,
			Cancellations:  deps.Cancellations,
			Runs:           deps.Runs,
			Storages:       deps.Storages,
			Databases:      deps.Databases,
			Notifiers:      deps.Notifiers,
			Logger:         deps.Logger,
			Clock:          deps.Clock,
			Transactor:     deps.Transactor,
			MasterKey:      deps.MasterKey,
			ClientToolsDir: deps.ClientToolsDir,
			WalQueueDir:    deps.WalQueueDir,
			Timings:        deps.WalStreamTimings,
		},
	)
	restoreService := restorestream.New(
		restorestream.Deps{
			Cache:    deps.Cache,
			Streams:  deps.UserStreams,
			Storages: deps.Storages,
			Logger:   deps.Logger,
			Clock:    deps.Clock,
		},
	)
	cleaner := retention.New(
		retention.Deps{
			Chains:           chainsService,
			Configs:          deps.Configs,
			Backups:          deps.Backups,
			WalSegments:      deps.WalSegments,
			WalStreamSamples: deps.WalStreamSamples,
			Logger:           deps.Logger,
			Clock:            deps.Clock,
		},
	)
	return &Service{
		usage:                deps.Usage,
		databases:            deps.Databases,
		storages:             deps.Storages,
		workspaces:           deps.Workspaces,
		configs:              deps.Configs,
		chains:               chainsService,
		backupRepository:     deps.Backups,
		walSegmentRepository: deps.WalSegments,
		canceller:            backupsService.Canceller,
		restoreStreamWriter:  restoreService.Writer,
		restoreTokenService:  restoreService,
		masterKey:            deps.MasterKey,
		auditing:             deps.Auditing,
		transactor:           deps.Transactor,
		clock:                deps.Clock,
		runs:                 deps.Runs,
		streams:              streamManager,
		health: streamhealth.New(
			streamhealth.Deps{
				WalSegments:      deps.WalSegments,
				WalStreamSamples: deps.WalStreamSamples,
				WalStreamers:     deps.WalStreamers,
			},
		),
		cancellations: deps.Cancellations,
		walStreamers:  deps.WalStreamers,
		logger:        deps.Logger,
		backuping:     backupsService,
		cleaner:       cleaner,
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.appCtx = ctx
	s.backuping.Backuper.Start(ctx)
	s.streams.Start(ctx)
	if err := s.backuping.Scheduler.FailBackupsLeftInProgress(ctx, s.logger); err != nil {
		return err
	}
	if err := s.streams.FailStaleStreamers(ctx); err != nil {
		return err
	}
	return backuping.DropOrphanBackupSlots(ctx, backuping.SlotCleanupSpec{Databases: s.databases, Logger: s.logger})
}

func (s *Service) GetJobs() []background.Job {
	return []background.Job{
		s.backuping.Scheduler.GetJob(),
		s.backuping.Scheduler.GetRequestJob(),
		s.cleaner.GetJob(),
		s.streams.GetJob(),
	}
}

func (s *Service) GetLatestCompletedFullBackups(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]catalog.Backup, error) {
	return s.backupRepository.GetLatestCompletedFullBackups(ctx, databaseIDs)
}
