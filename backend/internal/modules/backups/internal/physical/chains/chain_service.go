package chains

import (
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

type (
	Deps struct {
		Repository         *catalog.Repository
		FullBackups        *catalog.FullBackupRepository
		IncrementalBackups *catalog.IncrementalBackupRepository
		InFlightBackups    *catalog.InFlightBackupRepository
		WalSegments        *catalog.WalSegmentRepository
		WalHistory         *catalog.WalHistoryRepository

		Storages   storages.Module
		Transactor *database.Transactor
	}
	Service struct {
		repository                  *catalog.Repository
		fullBackupRepository        *catalog.FullBackupRepository
		incrementalBackupRepository *catalog.IncrementalBackupRepository
		inFlightBackupRepository    *catalog.InFlightBackupRepository
		walSegmentRepository        *catalog.WalSegmentRepository
		walHistoryRepository        *catalog.WalHistoryRepository

		storages   storages.Module
		transactor *database.Transactor
	}
)

func New(deps Deps) *Service {
	return &Service{
		repository:                  deps.Repository,
		fullBackupRepository:        deps.FullBackups,
		incrementalBackupRepository: deps.IncrementalBackups,
		inFlightBackupRepository:    deps.InFlightBackups,
		walSegmentRepository:        deps.WalSegments,
		walHistoryRepository:        deps.WalHistory,

		storages:   deps.Storages,
		transactor: deps.Transactor,
	}
}
