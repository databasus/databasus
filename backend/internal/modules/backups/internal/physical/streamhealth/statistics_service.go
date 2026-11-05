package streamhealth

import "databasus-backend/internal/modules/backups/internal/physical/catalog"

type (
	Deps struct {
		WalSegments      *catalog.WalSegmentRepository
		WalStreamSamples *catalog.WalStreamSampleRepository
		WalStreamers     *catalog.WalStreamerRepository
	}
	Service struct {
		walSegmentRepository      *catalog.WalSegmentRepository
		walStreamSampleRepository *catalog.WalStreamSampleRepository
		walStreamerRepository     *catalog.WalStreamerRepository
	}
)

func New(deps Deps) *Service {
	return &Service{
		walSegmentRepository:      deps.WalSegments,
		walStreamSampleRepository: deps.WalStreamSamples,
		walStreamerRepository:     deps.WalStreamers,
	}
}
