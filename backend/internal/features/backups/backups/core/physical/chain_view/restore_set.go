package chain_view

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	backups_physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

// RestoreSet is the minimal, ordered set of artifacts needed to reconstruct a
// cluster to a point in time: one FULL, the COMPLETED INCRs up to the target,
// the contiguous WAL between the last backup's stop_lsn and the target, and the
// timeline .history files. It is the input to the restore stream writer; both
// the user download and (later) restore verification consume the same shape.
type RestoreSet struct {
	RootFull     *physical_models.PhysicalFullBackup
	Incrementals []*physical_models.PhysicalIncrementalBackup
	WalSegments  []*physical_models.PhysicalWalSegment
	HistoryFiles []*physical_models.PhysicalWalHistoryFile

	// LastIncludedStopLSN is the stop_lsn of the last backup in the set (the last
	// INCR, or the FULL when there are no INCRs). pg_combinebackup output is
	// consistent at this LSN, so WAL replay starts here — WAL below it is already
	// folded into the artifacts and is NOT shipped.
	LastIncludedStopLSN walmath.LSN

	// MaxReplayableLSN is the furthest LSN the shipped WAL can replay to (the end
	// of the contiguous committed run, capped by the first gap).
	MaxReplayableLSN walmath.LSN
}

// ResolveRestoreSet selects the artifacts to restore databaseID to targetTime
// (nil ⇒ latest available point). It refuses with a typed error when the target
// is unreachable: ErrTargetBeforeEarliest (need an older chain),
// WalGapBeforeTargetError (a WAL gap stops replay short of the target), or
// ErrNoChainForRestore (no FULL at all). It performs no storage I/O — only the
// catalog is read.
func (s *ChainViewService) ResolveRestoreSet(
	databaseID uuid.UUID,
	targetTime *time.Time,
) (*RestoreSet, error) {
	reach, err := s.resolveRestoreReach(databaseID, targetTime)
	if err != nil {
		return nil, err
	}

	// A target later than the furthest point we can actually replay to means a
	// WAL gap (or simply missing WAL) sits before it. Refuse loudly; the caller
	// surfaces reachableLSN as the latest restorable point.
	if targetTime != nil && targetTime.After(reach.reachableTime) {
		return nil, WalGapBeforeTargetError{LatestRestorableLSN: reach.reachableLSN}
	}

	historyFiles, err := s.walHistoryRepository.FindAllByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	return &RestoreSet{
		RootFull:            reach.chain.RootFull,
		Incrementals:        reach.incrementals,
		WalSegments:         reach.walRun,
		HistoryFiles:        historyFiles,
		LastIncludedStopLSN: reach.lastStopLSN,
		MaxReplayableLSN:    reach.reachableLSN,
	}, nil
}

type restoreReach struct {
	chain         *ChainView
	incrementals  []*physical_models.PhysicalIncrementalBackup
	lastStopLSN   walmath.LSN
	walSegments   []*physical_models.PhysicalWalSegment
	walRun        []*physical_models.PhysicalWalSegment
	reachableLSN  walmath.LSN
	reachableTime time.Time
}

// resolveRestoreReach picks the chain and backups for targetTime (nil ⇒ latest)
// and walks the WAL after the last of them up to the first gap.
func (s *ChainViewService) resolveRestoreReach(
	databaseID uuid.UUID,
	targetTime *time.Time,
) (*restoreReach, error) {
	candidates, err := s.orderedRestoreCandidates(databaseID)
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

	incrementals, lastStopLSN, lastBackupTime := includedIncrementals(chain, targetTime)

	restoreWalSegments, err := s.walSegmentRepository.FindByChainSpan(
		databaseID,
		chain.RootFull.TimelineID,
		chain.Span.Start,
		LSNMax,
	)
	if err != nil {
		return nil, err
	}

	walRun, reachableLSN, reachableTime := contiguousWalRun(restoreWalSegments, lastStopLSN, lastBackupTime)

	return &restoreReach{
		chain:         chain,
		incrementals:  incrementals,
		lastStopLSN:   lastStopLSN,
		walSegments:   restoreWalSegments,
		walRun:        walRun,
		reachableLSN:  reachableLSN,
		reachableTime: reachableTime,
	}, nil
}

// ResolveRestoreSetForBackup selects the artifacts to restore to a SPECIFIC
// backup row rather than a point in time: a FULL restores just itself; an
// incremental restores its FULL plus the COMPLETED incrementals up to and
// including it (its chain ancestors). No WAL or history is shipped — a
// per-backup restore reconstructs the cluster to that backup's stop_lsn with
// pg_combinebackup alone, no replay. It performs no storage I/O. A FULL/INCR
// that is missing or not COMPLETED is reported as ErrNoChainForRestore.
func (s *ChainViewService) ResolveRestoreSetForBackup(
	databaseID, backupID uuid.UUID,
) (*RestoreSet, error) {
	full, err := s.fullBackupRepository.FindByID(backupID)
	if err != nil {
		return nil, err
	}

	if full != nil {
		if full.DatabaseID != databaseID {
			return nil, ErrNoChainForRestore
		}

		return restoreSetForFull(full)
	}

	incremental, err := s.incrementalBackupRepository.FindByID(backupID)
	if err != nil {
		return nil, err
	}

	if incremental != nil {
		if incremental.DatabaseID != databaseID {
			return nil, ErrNoChainForRestore
		}

		return s.restoreSetForIncremental(incremental)
	}

	return nil, ErrNoChainForRestore
}

func restoreSetForFull(full *physical_models.PhysicalFullBackup) (*RestoreSet, error) {
	if full.Status != backups_physical_enums.PhysicalBackupStatusCompleted || full.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	return &RestoreSet{
		RootFull:            full,
		LastIncludedStopLSN: *full.StopLSN,
		MaxReplayableLSN:    *full.StopLSN,
	}, nil
}

func (s *ChainViewService) restoreSetForIncremental(
	target *physical_models.PhysicalIncrementalBackup,
) (*RestoreSet, error) {
	if target.Status != backups_physical_enums.PhysicalBackupStatusCompleted ||
		target.StartLSN == nil || target.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	full, err := s.fullBackupRepository.FindByID(target.RootFullBackupID)
	if err != nil {
		return nil, err
	}
	if full == nil || full.StopLSN == nil {
		return nil, ErrNoChainForRestore
	}

	chain, err := s.incrementalBackupRepository.FindAllByRootFull(target.RootFullBackupID)
	if err != nil {
		return nil, err
	}

	// Ancestors are the COMPLETED incrementals at or before the target's
	// start_lsn (the chain arrives ordered by start_lsn ASC), the target itself
	// included.
	included := make([]*physical_models.PhysicalIncrementalBackup, 0, len(chain))
	lastStopLSN := *full.StopLSN

	for _, candidate := range chain {
		if candidate.Status != backups_physical_enums.PhysicalBackupStatusCompleted ||
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
func (s *ChainViewService) orderedRestoreCandidates(databaseID uuid.UUID) ([]*ChainView, error) {
	candidates := make([]*ChainView, 0)

	extendable, err := s.FindLastExtendableChainByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	if extendable != nil {
		candidates = append(candidates, extendable)
	}

	nonExtendable, err := s.FindNonExtendableChainsByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	return append(candidates, nonExtendable...), nil
}

// selectRestoreChain picks the chain to restore from: the newest one for a
// latest restore, or the newest chain whose FULL was completed at or before the
// target for a PITR restore.
func selectRestoreChain(candidates []*ChainView, targetTime *time.Time) (*ChainView, error) {
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

// includedIncrementals keeps the COMPLETED INCRs (ordered oldest→newest) up to
// the target, and reports the stop_lsn / completion time of the last one kept
// (the FULL's when none are kept). Incrementals arrive ordered by start_lsn ASC.
func includedIncrementals(
	chain *ChainView,
	targetTime *time.Time,
) ([]*physical_models.PhysicalIncrementalBackup, walmath.LSN, time.Time) {
	included := make([]*physical_models.PhysicalIncrementalBackup, 0, len(chain.Incrementals))

	lastStopLSN := *chain.RootFull.StopLSN
	lastBackupTime := derefTime(chain.RootFull.CompletedAt)

	for _, incremental := range chain.Incrementals {
		if incremental.Status != backups_physical_enums.PhysicalBackupStatusCompleted ||
			incremental.StopLSN == nil || incremental.CompletedAt == nil {
			continue
		}

		if targetTime != nil && incremental.CompletedAt.After(*targetTime) {
			continue
		}

		included = append(included, incremental)
		lastStopLSN = *incremental.StopLSN
		lastBackupTime = *incremental.CompletedAt
	}

	return included, lastStopLSN, lastBackupTime
}

// contiguousWalRun walks the chain's committed WAL segments forward from
// lastStopLSN, collecting an unbroken run until the first gap. Segments still
// uploading (file_name NULL) and segments already folded into the backups
// (end_lsn ≤ lastStopLSN) are skipped. Returns the run, the furthest LSN it
// reaches, and the timestamp of that furthest point.
func contiguousWalRun(
	segments []*physical_models.PhysicalWalSegment,
	lastStopLSN walmath.LSN,
	lastBackupTime time.Time,
) ([]*physical_models.PhysicalWalSegment, walmath.LSN, time.Time) {
	run := make([]*physical_models.PhysicalWalSegment, 0)

	cursor := lastStopLSN
	reachableTime := lastBackupTime

	for _, segment := range segments {
		if segment.FileName == nil || segment.EndLSN <= lastStopLSN {
			continue
		}

		// A segment that starts past the cursor leaves a hole the replay can't
		// cross — stop here, cursor is the furthest reachable LSN.
		if segment.StartLSN > cursor {
			break
		}

		run = append(run, segment)
		cursor = segment.EndLSN
		reachableTime = segment.ReceivedAt
	}

	return run, cursor, reachableTime
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}

	return *t
}
