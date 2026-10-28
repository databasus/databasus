package chain_view

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/storage"
)

type ChainViewService struct {
	fullBackupRepository        *physical_repositories.PhysicalFullBackupRepository
	incrementalBackupRepository *physical_repositories.PhysicalIncrementalBackupRepository
	walSegmentRepository        *physical_repositories.PhysicalWalSegmentRepository
	walHistoryRepository        *physical_repositories.PhysicalWalHistoryRepository
}

func (s *ChainViewService) FindLastExtendableChainByDatabase(databaseID uuid.UUID) (*ChainView, error) {
	fulls, err := s.fullBackupRepository.FindCompletedNewestFirstByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	if len(fulls) == 0 {
		return nil, nil
	}

	newestView, err := s.buildChainView(fulls[0])
	if err != nil {
		return nil, err
	}

	if !isExtendable(newestView) {
		return nil, nil
	}

	return newestView, nil
}

func (s *ChainViewService) FindNonExtendableChainsByDatabase(databaseID uuid.UUID) ([]*ChainView, error) {
	fulls, err := s.fullBackupRepository.FindCompletedNewestFirstByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	views := make([]*ChainView, 0, len(fulls))

	for i, full := range fulls {
		view, err := s.buildChainView(full)
		if err != nil {
			return nil, err
		}

		if i == 0 && isExtendable(view) {
			continue
		}

		views = append(views, view)
	}

	return views, nil
}

func (s *ChainViewService) GetChainSpan(rootFullBackupID uuid.UUID) (LSNRange, error) {
	full, err := s.fullBackupRepository.FindByID(rootFullBackupID)
	if err != nil {
		return LSNRange{}, err
	}
	if full == nil {
		return LSNRange{}, fmt.Errorf("chain_view: full backup not found: %s", rootFullBackupID)
	}

	return s.getChainSpanForFull(full)
}

func (s *ChainViewService) FindWalGapsInChain(rootFullBackupID uuid.UUID) ([]LSNRange, error) {
	full, err := s.fullBackupRepository.FindByID(rootFullBackupID)
	if err != nil {
		return nil, err
	}
	if full == nil {
		return nil, fmt.Errorf("chain_view: full backup not found: %s", rootFullBackupID)
	}

	span, err := s.GetChainSpan(rootFullBackupID)
	if err != nil {
		return nil, err
	}

	segments, err := s.walSegmentRepository.FindByChainSpan(
		full.DatabaseID, full.TimelineID, span.Start, span.End,
	)
	if err != nil {
		return nil, err
	}

	return findGapsBetweenSegments(segments), nil
}

// findGapsBetweenSegments lists the holes between consecutive segments ordered
// by start_lsn; nil for fewer than two segments.
func findGapsBetweenSegments(segments []*physical_models.PhysicalWalSegment) []LSNRange {
	if len(segments) < 2 {
		return nil
	}

	gaps := make([]LSNRange, 0)

	for i := 1; i < len(segments); i++ {
		prev := segments[i-1]
		curr := segments[i]

		if curr.StartLSN > prev.EndLSN {
			gaps = append(gaps, LSNRange{
				Start: prev.EndLSN,
				End:   curr.StartLSN,
			})
		}
	}

	return gaps
}

func (s *ChainViewService) GetChainEndTimestamp(view *ChainView) (time.Time, error) {
	latest := time.Time{}
	if view.RootFull.CompletedAt != nil {
		latest = *view.RootFull.CompletedAt
	}

	for _, incrementalBackup := range view.Incrementals {
		if incrementalBackup.CompletedAt != nil && incrementalBackup.CompletedAt.After(latest) {
			latest = *incrementalBackup.CompletedAt
		}
	}

	latestWalReceivedAt, err := s.walSegmentRepository.GetLatestReceivedAtInChainSpan(
		view.RootFull.DatabaseID, view.RootFull.TimelineID, view.Span.Start, view.Span.End,
	)
	if err != nil {
		return time.Time{}, err
	}

	if latestWalReceivedAt != nil && latestWalReceivedAt.After(latest) {
		latest = *latestWalReceivedAt
	}

	return latest, nil
}

func (s *ChainViewService) FindWalOrphansByDatabase(databaseID uuid.UUID) ([]WalOrphanRef, error) {
	walOrphans, err := s.walSegmentRepository.FindOrphans(databaseID)
	if err != nil {
		return nil, err
	}

	refs := make([]WalOrphanRef, 0, len(walOrphans))
	for _, walOrphan := range walOrphans {
		refs = append(refs, WalOrphanRef{WalSegment: walOrphan})
	}

	return refs, nil
}

func (s *ChainViewService) getChainSpanForFull(full *physical_models.PhysicalFullBackup) (LSNRange, error) {
	if full.StartLSN == nil {
		return LSNRange{}, errors.New("chain_view: cannot compute span — root FULL has no start_lsn")
	}

	start := *full.StartLSN

	var successor physical_models.PhysicalFullBackup

	err := storage.
		GetDb().
		Where(
			"database_id = ? AND timeline_id = ? AND status = ? AND start_lsn > ?::pg_lsn",
			full.DatabaseID,
			full.TimelineID,
			physical_enums.PhysicalBackupStatusCompleted,
			start.String(),
		).
		Order("start_lsn ASC").
		First(&successor).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LSNRange{Start: start, End: LSNMax}, nil
		}

		return LSNRange{}, err
	}

	end := LSNMax
	if successor.StartLSN != nil {
		end = *successor.StartLSN
	}

	return LSNRange{Start: start, End: end}, nil
}

// Any newer COMPLETED FULL closes a chain, so only the newest one can still be
// extended, and only until one of its INCRs breaks the chain. WAL gaps never
// change this: a lossy chain stays extendable, and FindWalGapsInChain surfaces
// the unreachable PITR windows.
func isExtendable(newestView *ChainView) bool {
	return !slices.ContainsFunc(
		newestView.Incrementals,
		func(incrementalBackup *physical_models.PhysicalIncrementalBackup) bool {
			return incrementalBackup.Status == physical_enums.PhysicalBackupStatusChainBroken
		},
	)
}

func (s *ChainViewService) buildChainView(
	full *physical_models.PhysicalFullBackup,
) (*ChainView, error) {
	span, err := s.getChainSpanForFull(full)
	if err != nil {
		return nil, err
	}

	incrementalBackups, err := s.incrementalBackupRepository.FindAllByRootFull(full.ID)
	if err != nil {
		return nil, err
	}

	return &ChainView{
		RootFull:     full,
		Incrementals: incrementalBackups,
		Span:         span,
	}, nil
}
