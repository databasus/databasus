package walstreaming

import (
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
)

type Timings struct {
	ForcedRotationInterval    time.Duration
	ArchiveStalenessThreshold time.Duration
	LocalQueuePauseBytes      int64
}

type Deps struct {
	WalSegments    *catalog.WalSegmentRepository
	HistoryFiles   *catalog.WalHistoryRepository
	Streamers      *catalog.WalStreamerRepository
	Samples        *catalog.WalStreamSampleRepository
	Configs        *config.Service
	Cancellations  *background.CancellationRegistry
	Runs           *background.Group
	Storages       storages.Module
	Databases      databases.Module
	Notifiers      notifiers.Module
	Logger         *slog.Logger
	Clock          clock.Source
	Transactor     *database.Transactor
	MasterKey      *crypto.MasterKey
	ClientToolsDir string
	WalQueueDir    string
	Timings        Timings
}

func New(deps Deps) *Manager {
	return &Manager{
		databases:                deps.Databases,
		configs:                  deps.Configs,
		storages:                 deps.Storages,
		walSegmentRepo:           deps.WalSegments,
		historyRepo:              deps.HistoryFiles,
		walStreamerRepo:          deps.Streamers,
		walStreamSampleRepo:      deps.Samples,
		notificationSender:       deps.Notifiers,
		taskCancellationRegistry: deps.Cancellations,
		masterKey:                deps.MasterKey,
		logger:                   deps.Logger,
		clock:                    deps.Clock,
		transactor:               deps.Transactor,
		runs:                     deps.Runs,
		clientToolsDir:           deps.ClientToolsDir,
		walQueueDir:              deps.WalQueueDir,
		chainAlertMinInterval:    defaultChainAlertMinInterval,
		timings:                  deps.Timings,
		running:                  make(map[uuid.UUID]*runningStreamer),
		lastChainAlertAt:         make(chainAlertThrottle),
	}
}
