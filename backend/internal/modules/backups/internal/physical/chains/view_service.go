package chains

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func (s *Service) FindLastExtendableChainByDatabase(ctx context.Context, databaseID uuid.UUID) (*ChainView, error) {
	fulls, err := s.backupRepository.FindCompletedFullNewestFirstByDatabase(ctx, databaseID)
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
	fulls, err := s.backupRepository.FindCompletedFullNewestFirstByDatabase(ctx, databaseID)
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

func (s *Service) getChainSpanForFull(ctx context.Context, full *catalog.Backup) (catalog.LSNRange, error) {
	if full.StartLSN == nil {
		return catalog.LSNRange{}, errors.New("chain_view: cannot compute span — root FULL has no start_lsn")
	}

	start := *full.StartLSN

	successor, isFound, err := s.backupRepository.FindNextCompletedFull(ctx, full)
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
// change this: a lossy chain stays extendable.
func isExtendable(newestView *ChainView) bool {
	return !slices.ContainsFunc(
		newestView.Incrementals,
		func(incrementalBackup *catalog.Backup) bool {
			return incrementalBackup.Status == catalog.BackupStatusChainBroken
		},
	)
}

func (s *Service) buildChainView(ctx context.Context, full *catalog.Backup,
) (*ChainView, error) {
	span, err := s.getChainSpanForFull(ctx, full)
	if err != nil {
		return nil, err
	}

	incrementalBackups, err := s.backupRepository.FindAllByRootFull(ctx, full.ID)
	if err != nil {
		return nil, err
	}

	return &ChainView{
		RootFull:     full,
		Incrementals: incrementalBackups,
		Span:         span,
	}, nil
}
