package chains

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

// RestoreSet is the minimal, ordered set of artifacts needed to reconstruct a
// cluster to a point in time: one FULL, the COMPLETED INCRs up to the target,
// the contiguous WAL between the last backup's stop_lsn and the target, and the
// timeline .history files. It is the input to the restore stream writer; both
// the user download and (later) restore verification consume the same shape.
type RestoreSet struct {
	RootFull     *catalog.Backup
	Incrementals []*catalog.Backup
	WalSegments  []*catalog.WalSegment
	HistoryFiles []*catalog.WalHistoryFile

	// LastIncludedStopLSN is the stop_lsn of the last backup in the set (the last
	// INCR, or the FULL when there are no INCRs). pg_combinebackup output is
	// consistent at this LSN, so WAL replay starts here — WAL below it is already
	// folded into the artifacts and is NOT shipped.
	LastIncludedStopLSN walmath.LSN

	// MaxReplayableLSN is the furthest LSN the shipped WAL can replay to (the end
	// of the contiguous committed run, capped by the first gap).
	MaxReplayableLSN walmath.LSN

	// TargetTimelineID is the newest timeline the shipped WAL reaches, which recovery must follow.
	// It is nil when the set ships no WAL.
	TargetTimelineID *int
}

// ResolveRestoreSet selects the artifacts to restore databaseID to targetTime
// (nil ⇒ latest available point). It refuses with a typed error when the target
// is unreachable: ErrTargetBeforeEarliest (need an older chain),
// WalGapBeforeTargetError (a WAL gap stops replay short of the target), or
// ErrNoChainForRestore (no FULL at all). It performs no storage I/O — only the
// catalog is read.
func (s *Service) ResolveRestoreSet(ctx context.Context, databaseID uuid.UUID,
	targetTime *time.Time,
) (*RestoreSet, error) {
	reach, err := s.resolveRestoreReach(ctx, databaseID, targetTime)
	if err != nil {
		return nil, err
	}

	// A target later than the furthest point we can actually replay to means a
	// WAL gap (or simply missing WAL) sits before it. Refuse loudly; the caller
	// surfaces reachableLSN as the latest restorable point.
	if targetTime != nil && targetTime.After(reach.reachableTime) {
		return nil, WalGapBeforeTargetError{LatestRestorableLSN: reach.reachableLSN}
	}

	return &RestoreSet{
		RootFull:            reach.chain.RootFull,
		Incrementals:        reach.incrementals,
		WalSegments:         reach.walRun,
		HistoryFiles:        reach.historyFiles,
		LastIncludedStopLSN: reach.lastStopLSN,
		MaxReplayableLSN:    reach.reachableLSN,
		TargetTimelineID:    reach.targetTimelineID,
	}, nil
}

type restoreReach struct {
	chain         *ChainView
	incrementals  []*catalog.Backup
	lastStopLSN   walmath.LSN
	walSegments   []*catalog.WalSegment
	historyFiles  []*catalog.WalHistoryFile
	walRun        []*catalog.WalSegment
	reachableLSN  walmath.LSN
	reachableTime time.Time

	targetTimelineID *int
}

// resolveRestoreReach picks the chain and backups for targetTime (nil ⇒ latest)
// and walks the WAL after the last of them along the chain's timeline line up
// to the first gap.
func (s *Service) resolveRestoreReach(ctx context.Context, databaseID uuid.UUID,
	targetTime *time.Time,
) (*restoreReach, error) {
	candidates, err := s.orderedRestoreCandidates(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	chain, err := selectRestoreChain(candidates, targetTime)
	if err != nil {
		return nil, err
	}

	if chain.RootFull.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	incrementals, lastBackup := includedIncrementals(chain, targetTime)

	histories, err := s.walHistoryRepository.FindAllByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	restoreWalSegments := make([]*catalog.WalSegment, 0)
	if hasFirstTimelineHistory(chain, histories) {
		lineSegments, err := s.walSegmentRepository.FindByTimelineLine(
			ctx,
			databaseID,
			chain.Line.getSpans(catalog.LSNRange{Start: chain.Span.Start, End: catalog.LSNMax}),
		)
		if err != nil {
			return nil, err
		}

		restoreWalSegments = withoutForkSegmentEvidence(chain.Line, lineSegments)
	}

	walk := walkTimelineLine(timelineLineWalkSpec{
		Line:                 chain.Line,
		StorageID:            chain.RootFull.StorageID,
		Segments:             restoreWalSegments,
		LastStopLSN:          lastBackup.StopLSN,
		LastBackupTime:       lastBackup.CompletedAt,
		LastBackupTimelineID: lastBackup.TimelineID,
	})
	walRun, reachableLSN := walk.Run, walk.ReachableLSN
	if targetTime != nil && targetTime.Equal(lastBackup.CompletedAt) {
		hasConfirmingTransaction := slices.ContainsFunc(walRun, func(segment *catalog.WalSegment) bool {
			return segment.LastTransactionAt != nil && segment.LastTransactionAt.After(lastBackup.CompletedAt)
		})
		if !hasConfirmingTransaction {
			walRun = nil
			reachableLSN = lastBackup.StopLSN
		}
	}

	var historyFiles []*catalog.WalHistoryFile
	var targetTimelineID *int
	if len(walRun) > 0 {
		reachedTimelineID := walRun[len(walRun)-1].TimelineID
		targetTimelineID = &reachedTimelineID
		historyFiles = getLineHistoryFiles(chain, histories, walk.ReachedRangeIndex)
	}

	return &restoreReach{
		chain:            chain,
		incrementals:     incrementals,
		lastStopLSN:      lastBackup.StopLSN,
		walSegments:      restoreWalSegments,
		historyFiles:     historyFiles,
		walRun:           walRun,
		reachableLSN:     reachableLSN,
		reachableTime:    walk.ReachableTime,
		targetTimelineID: targetTimelineID,
	}, nil
}

// hasFirstTimelineHistory reports whether WAL of the chain's own timeline may ship: timeline 1 needs
// no history, and a later timeline needs its published history in the FULL's storage.
func hasFirstTimelineHistory(chain *ChainView, histories []*catalog.WalHistoryFile) bool {
	firstTimelineID := chain.Line.Ranges[0].TimelineID
	if firstTimelineID == 1 {
		return true
	}

	return findHistoryInStorage(histories, firstTimelineID, chain.RootFull.StorageID) != nil
}

// getLineHistoryFiles returns the history of every timeline above 1 on the line up to the reached
// range, in the FULL's storage. Recovery that targets the newest of them reads its ancestry from them.
func getLineHistoryFiles(
	chain *ChainView,
	histories []*catalog.WalHistoryFile,
	reachedRangeIndex int,
) []*catalog.WalHistoryFile {
	historyFiles := make([]*catalog.WalHistoryFile, 0, reachedRangeIndex+1)

	for _, lineRange := range chain.Line.Ranges[:reachedRangeIndex+1] {
		if lineRange.TimelineID == 1 {
			continue
		}

		if history := findHistoryInStorage(histories, lineRange.TimelineID, chain.RootFull.StorageID); history != nil {
			historyFiles = append(historyFiles, history)
		}
	}

	return historyFiles
}

func findHistoryInStorage(
	histories []*catalog.WalHistoryFile,
	timelineID int,
	storageID uuid.UUID,
) *catalog.WalHistoryFile {
	for _, history := range histories {
		if history.TimelineID == timelineID && history.StorageID == storageID {
			return history
		}
	}

	return nil
}

// ResolveRestoreSetForBackup selects the artifacts to restore to a SPECIFIC
// backup row rather than a point in time: a FULL restores just itself; an
// incremental restores its FULL plus the COMPLETED incrementals up to and
// including it (its chain ancestors). No WAL or history is shipped — a
// per-backup restore reconstructs the cluster to that backup's stop_lsn with
// pg_combinebackup alone, no replay. It performs no storage I/O. A FULL/INCR
// that is missing or not COMPLETED is reported as ErrNoChainForRestore.
func (s *Service) ResolveRestoreSetForBackup(ctx context.Context, databaseID, backupID uuid.UUID,
) (*RestoreSet, error) {
	backup, isFound, err := s.backupRepository.FindByID(ctx, backupID)
	if err != nil {
		return nil, err
	}
	if !isFound || backup.DatabaseID != databaseID {
		return nil, ErrNoChainForRestore
	}
	if backup.Kind == catalog.BackupKindFull {
		return restoreSetForFull(backup)
	}

	return s.restoreSetForIncremental(ctx, backup)
}

func restoreSetForFull(full *catalog.Backup) (*RestoreSet, error) {
	if full.Status != catalog.BackupStatusCompleted || full.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	return &RestoreSet{
		RootFull:            full,
		LastIncludedStopLSN: *full.StopLSN,
		MaxReplayableLSN:    *full.StopLSN,
	}, nil
}

func (s *Service) restoreSetForIncremental(ctx context.Context, target *catalog.Backup,
) (*RestoreSet, error) {
	if target.Status != catalog.BackupStatusCompleted ||
		target.StartLSN == nil || target.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	full, _, err := s.backupRepository.FindByID(ctx, *target.RootFullBackupID)
	if err != nil {
		return nil, err
	}
	if full == nil || full.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	chain, err := s.backupRepository.FindAllByRootFull(ctx, *target.RootFullBackupID)
	if err != nil {
		return nil, err
	}

	// Ancestors are the COMPLETED incrementals at or before the target's
	// start_lsn (the chain arrives ordered by start_lsn ASC), the target itself
	// included.
	included := make([]*catalog.Backup, 0, len(chain))
	lastStopLSN := *full.StopLSN

	for _, candidate := range chain {
		if candidate.Status != catalog.BackupStatusCompleted ||
			candidate.StartLSN == nil || candidate.StopLSN == nil {
			continue
		}

		if *candidate.StartLSN > *target.StartLSN {
			continue
		}

		included = append(included, candidate)
		lastStopLSN = *candidate.StopLSN
	}

	return &RestoreSet{
		RootFull:            full,
		Incrementals:        included,
		LastIncludedStopLSN: lastStopLSN,
		MaxReplayableLSN:    lastStopLSN,
	}, nil
}

// orderedRestoreCandidates returns every chain of the database newest→oldest.
// The single newest-extendable chain (if any) leads; FindNonExtendableChains
// already excludes it and preserves newest-first order for the rest.
func (s *Service) orderedRestoreCandidates(ctx context.Context, databaseID uuid.UUID) ([]*ChainView, error) {
	candidates := make([]*ChainView, 0)

	extendable, err := s.FindLastExtendableChainByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	if extendable != nil {
		candidates = append(candidates, extendable)
	}

	nonExtendable, err := s.FindNonExtendableChainsByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	return append(candidates, nonExtendable...), nil
}

// selectRestoreChain picks the chain to restore from: the newest one for a
// latest restore, or the newest chain whose FULL was completed at or before the
// target for a PITR restore. A FULL off the newest timeline line stays
// restorable by its identifier only.
func selectRestoreChain(candidates []*ChainView, targetTime *time.Time) (*ChainView, error) {
	candidates = slices.DeleteFunc(slices.Clone(candidates), func(candidate *ChainView) bool {
		return candidate.Line.IsOffLine
	})
	if len(candidates) == 0 {
		return nil, ErrNoChainForRestore
	}

	if targetTime == nil {
		return candidates[0], nil
	}

	for _, candidate := range candidates {
		completedAt := candidate.RootFull.CompletedAt
		if completedAt != nil && !completedAt.After(*targetTime) {
			return candidate, nil
		}
	}

	return nil, ErrTargetBeforeEarliest
}

// includedBackup is the newest backup a restore includes, where WAL replay starts.
type includedBackup struct {
	StopLSN     walmath.LSN
	CompletedAt time.Time
	TimelineID  int
}

// includedIncrementals keeps the COMPLETED INCRs (ordered oldest→newest) up to
// the target, and reports the newest backup kept (the FULL when none are kept).
// Incrementals arrive ordered by start_lsn ASC.
func includedIncrementals(
	chain *ChainView,
	targetTime *time.Time,
) ([]*catalog.Backup, includedBackup) {
	included := make([]*catalog.Backup, 0, len(chain.Incrementals))

	lastBackup := includedBackup{
		StopLSN:     *chain.RootFull.StopLSN,
		CompletedAt: derefTime(chain.RootFull.CompletedAt),
		TimelineID:  chain.RootFull.TimelineID,
	}

	for _, incremental := range chain.Incrementals {
		if incremental.Status != catalog.BackupStatusCompleted ||
			incremental.StopLSN == nil || incremental.CompletedAt == nil {
			continue
		}

		if targetTime != nil && incremental.CompletedAt.After(*targetTime) {
			continue
		}

		included = append(included, incremental)
		lastBackup = includedBackup{
			StopLSN:     *incremental.StopLSN,
			CompletedAt: *incremental.CompletedAt,
			TimelineID:  incremental.TimelineID,
		}
	}

	return included, lastBackup
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}
