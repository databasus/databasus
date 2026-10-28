package chains

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func (s *Service) FindLastExtendableChainByDatabase(ctx context.Context, databaseID uuid.UUID) (*ChainView, error) {
	fulls, err := s.fullBackupRepository.FindCompletedNewestFirstByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if len(fulls) == 0 {
		return nil, nil
	}

	newestView, err := s.buildChainView(ctx, fulls[0])
	if err != nil {
		return nil, err
	}

	if !isExtendable(newestView) {
		return nil, nil
	}

	return newestView, nil
}

func (s *Service) FindNonExtendableChainsByDatabase(ctx context.Context, databaseID uuid.UUID) ([]*ChainView, error) {
	fulls, err := s.fullBackupRepository.FindCompletedNewestFirstByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	views := make([]*ChainView, 0, len(fulls))

	for i, full := range fulls {
		view, err := s.buildChainView(ctx, full)
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

func (s *Service) GetChainSpan(ctx context.Context, rootFullBackupID uuid.UUID) (catalog.LSNRange, error) {
	full, _, err := s.fullBackupRepository.FindByID(ctx, rootFullBackupID)
	if err != nil {
		return catalog.LSNRange{}, err
	}
	if full == nil {
		return catalog.LSNRange{}, fmt.Errorf("chain_view: full backup not found: %s", rootFullBackupID)
	}

	return s.getChainSpanForFull(ctx, full)
}

func (s *Service) FindWalGapsInChain(ctx context.Context, rootFullBackupID uuid.UUID) ([]catalog.LSNRange, error) {
	full, _, err := s.fullBackupRepository.FindByID(ctx, rootFullBackupID)
	if err != nil {
		return nil, err
	}
	if full == nil {
		return nil, fmt.Errorf("chain_view: full backup not found: %s", rootFullBackupID)
	}

	span, err := s.GetChainSpan(ctx, rootFullBackupID)
	if err != nil {
		return nil, err
	}

	segments, err := s.walSegmentRepository.FindByChainSpan(ctx, full.DatabaseID, full.TimelineID, span.Start, span.End)
	if err != nil {
		return nil, err
	}

	return findGapsBetweenSegments(segments), nil
}

// findGapsBetweenSegments lists the holes between consecutive segments ordered
// by start_lsn; nil for fewer than two segments.
func findGapsBetweenSegments(segments []*catalog.WalSegment) []catalog.LSNRange {
	if len(segments) < 2 {
		return nil
	}

	gaps := make([]catalog.LSNRange, 0)

	for i := 1; i < len(segments); i++ {
		prev := segments[i-1]
		curr := segments[i]

		if curr.StartLSN > prev.EndLSN {
			gaps = append(gaps, catalog.LSNRange{
				Start: prev.EndLSN,
				End:   curr.StartLSN,
			})
		}
	}

	return gaps
}

func (s *Service) GetChainEndTimestamp(ctx context.Context, view *ChainView) (time.Time, error) {
	latest := time.Time{}
	if view.RootFull.CompletedAt != nil {
		latest = *view.RootFull.CompletedAt
	}

	for _, incrementalBackup := range view.Incrementals {
		if incrementalBackup.CompletedAt != nil && incrementalBackup.CompletedAt.After(latest) {
			latest = *incrementalBackup.CompletedAt
		}
	}

	latestWalReceivedAt, _, err := s.walSegmentRepository.GetLatestReceivedAtInChainSpan(
		ctx,
		view.RootFull.DatabaseID,
		view.RootFull.TimelineID,
		view.Span.Start,
		view.Span.End,
	)
	if err != nil {
		return time.Time{}, err
	}

	if latestWalReceivedAt != nil && latestWalReceivedAt.After(latest) {
		latest = *latestWalReceivedAt
	}

	return latest, nil
}

func (s *Service) FindWalOrphansByDatabase(ctx context.Context, databaseID uuid.UUID) ([]WalOrphanRef, error) {
	walOrphans, err := s.walSegmentRepository.FindOrphans(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	refs := make([]WalOrphanRef, 0, len(walOrphans))
	for _, walOrphan := range walOrphans {
		refs = append(refs, WalOrphanRef{WalSegment: walOrphan})
	}

	return refs, nil
}

func (s *Service) getChainSpanForFull(ctx context.Context, full *catalog.FullBackup) (catalog.LSNRange, error) {
	if full.StartLSN == nil {
		return catalog.LSNRange{}, errors.New("chain_view: cannot compute span — root FULL has no start_lsn")
	}

	start := *full.StartLSN

	successor, isFound, err := s.fullBackupRepository.FindNextCompleted(ctx, full)
	if err != nil {
		return catalog.LSNRange{}, err
	}
	if !isFound {
		return catalog.LSNRange{Start: start, End: catalog.LSNMax}, nil
	}

	end := catalog.LSNMax
	if successor.StartLSN != nil {
		end = *successor.StartLSN
	}

	return catalog.LSNRange{Start: start, End: end}, nil
}

// Any newer COMPLETED FULL closes a chain, so only the newest one can still be
// extended, and only until one of its INCRs breaks the chain. WAL gaps never
// change this: a lossy chain stays extendable, and FindWalGapsInChain surfaces
// the unreachable PITR windows.
func isExtendable(newestView *ChainView) bool {
	return !slices.ContainsFunc(
		newestView.Incrementals,
		func(incrementalBackup *catalog.IncrementalBackup) bool {
			return incrementalBackup.Status == catalog.BackupStatusChainBroken
		},
	)
}

func (s *Service) buildChainView(ctx context.Context, full *catalog.FullBackup,
) (*ChainView, error) {
	span, err := s.getChainSpanForFull(ctx, full)
	if err != nil {
		return nil, err
	}

	incrementalBackups, err := s.incrementalBackupRepository.FindAllByRootFull(ctx, full.ID)
	if err != nil {
		return nil, err
	}

	return &ChainView{
		RootFull:     full,
		Incrementals: incrementalBackups,
		Span:         span,
	}, nil
}
