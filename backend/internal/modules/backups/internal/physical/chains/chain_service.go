package chains

import (
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

type (
	Deps struct {
		Repository  *catalog.Repository
		Backups     *catalog.BackupRepository
		WalSegments *catalog.WalSegmentRepository
		WalHistory  *catalog.WalHistoryRepository

		Storages   storages.Module
		Transactor *database.Transactor
	}
	Service struct {
		repository           *catalog.Repository
		backupRepository     *catalog.BackupRepository
		walSegmentRepository *catalog.WalSegmentRepository
		walHistoryRepository *catalog.WalHistoryRepository

		storages   storages.Module
		transactor *database.Transactor
	}
)

func New(deps Deps) *Service {
	return &Service{
		repository:           deps.Repository,
		backupRepository:     deps.Backups,
		walSegmentRepository: deps.WalSegments,
		walHistoryRepository: deps.WalHistory,

		storages:   deps.Storages,
		transactor: deps.Transactor,
	}
}
