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
	chainCatalog, err := s.loadChainCatalog(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if len(chainCatalog.Fulls) == 0 {
		return nil, nil
	}

	newestView, err := s.buildChainView(ctx, chainCatalog, chainCatalog.Fulls[0])
	if err != nil {
		return nil, err
	}

	if !isExtendable(newestView) {
		return nil, nil
	}

	return newestView, nil
}

func (s *Service) FindNonExtendableChainsByDatabase(ctx context.Context, databaseID uuid.UUID) ([]*ChainView, error) {
	chainCatalog, err := s.loadChainCatalog(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	views := make([]*ChainView, 0, len(chainCatalog.Fulls))

	for i, full := range chainCatalog.Fulls {
		view, err := s.buildChainView(ctx, chainCatalog, full)
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

	latestWalReceivedAt, _, err := s.walSegmentRepository.GetLatestReceivedAtOnTimelineLine(
		ctx,
		view.RootFull.DatabaseID,
		view.GetOwnedSpans(),
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
	candidates, err := s.walSegmentRepository.FindOrphanCandidates(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return []WalOrphanRef{}, nil
	}

	chainCatalog, err := s.loadChainCatalog(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	ownership := newWalOwnership(chainCatalog, uuid.Nil)
	refs := make([]WalOrphanRef, 0, len(candidates))
	for _, candidate := range candidates {
		if ownership.isUnreferenced(
			candidate.TimelineID,
			catalog.LSNRange{Start: candidate.StartLSN, End: candidate.EndLSN},
		) {
			refs = append(refs, WalOrphanRef{WalSegment: candidate})
		}
	}

	return refs, nil
}

// chainCatalog holds what every chain view of a database is built from, read once per lookup: the
// completed FULLs newest first and the timeline history rows that give their lines.
type chainCatalog struct {
	Fulls     []*catalog.Backup
	Histories []*catalog.WalHistoryFile
}

func (s *Service) loadChainCatalog(ctx context.Context, databaseID uuid.UUID) (chainCatalog, error) {
	fulls, err := s.backupRepository.FindCompletedFullNewestFirstByDatabase(ctx, databaseID)
	if err != nil {
		return chainCatalog{}, err
	}

	histories, err := s.walHistoryRepository.FindAllByDatabase(ctx, databaseID)
	if err != nil {
		return chainCatalog{}, err
	}

	return chainCatalog{Fulls: fulls, Histories: histories}, nil
}

// getChainSpan bounds a chain by the next completed FULL on its line, which may lie on a later
// timeline of the line. LSNs grow along a line, so the nearest such FULL is the one with the
// lowest start above the chain's own.
func getChainSpan(chainCatalog chainCatalog, full *catalog.Backup, line TimelineLine) (catalog.LSNRange, error) {
	if full.StartLSN == nil {
		return catalog.LSNRange{}, errors.New("chain_view: cannot compute span — root FULL has no start_lsn")
	}

	span := catalog.LSNRange{Start: *full.StartLSN, End: catalog.LSNMax}
	for _, successor := range chainCatalog.Fulls {
		if successor.ID == full.ID || successor.StartLSN == nil || *successor.StartLSN <= span.Start {
			continue
		}

		if line.getRangeIndexContaining(successor.TimelineID, *successor.StartLSN) >= 0 {
			span.End = min(span.End, *successor.StartLSN)
		}
	}

	return span, nil
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

func (s *Service) buildChainView(ctx context.Context, chainCatalog chainCatalog, full *catalog.Backup,
) (*ChainView, error) {
	line := BuildTimelineLine(chainCatalog.Histories, full)

	span, err := getChainSpan(chainCatalog, full, line)
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
		Line:         line,
	}, nil
}
