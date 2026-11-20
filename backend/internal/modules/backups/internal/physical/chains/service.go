package chains

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

const (
	// Sidecar suffix mirrors the executor's metadata convention; every FULL /
	// INCR / WAL / history artifact has a sibling "<name>.metadata".
	metadataSuffix = ".metadata"

	// One DELETE transaction never removes more than this many WAL rows per
	// batch, so the anchor-FULL lock is not held across an unbounded delete.
	maxWalDeleteBatchRows = 50

	// unboundedWalBudgetMB is the budget a user-initiated delete passes so the
	// WAL byte cap never trips: the user expects the target gone now, not
	// drained over many cleaner ticks. Far above any real catalog's WAL volume.
	unboundedWalBudgetMB = 1e18
)

// Service owns the transactional, multi-table operations over the
// physical catalog that the cleaner needs: chain-cascade deletion (file before
// row, leaves before parents), usage accounting, and orphan-WAL deletion. It is
// the single seam through which the cleaner mutates catalog rows — the cleaner
// never issues raw DELETEs.
// DeleteFull cascades one chain's deletion bounded by walByteBudgetMB. The
// order is WAL (oldest LSN first) → INCRs (leaves first) → orphaned history →
// the FULL itself, each object deleted from storage before its row. Idempotent:
// a missing FULL is a no-op (a peer already deleted it); a budget-capped call
// returns ChainFullyDeleted=false and the caller resumes next tick.
func (s *Service) DeleteFull(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
	walByteBudgetMB float64,
) (DeletedSummary, error) {
	return s.cascadeDelete(ctx, rootFullBackupID, walByteBudgetMB, false)
}

// DeleteFullCompletely removes a whole chain in one user-initiated call, looping
// DeleteFull with an unbounded budget until the FULL row is gone. The cleaner
// uses the budgeted DeleteFull to spread work across ticks; the API delete path
// uses this so the user sees the chain fully removed rather than partially.
func (s *Service) DeleteFullCompletely(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
) (DeletedSummary, error) {
	total := DeletedSummary{RootFullBackupID: rootFullBackupID}

	for {
		summary, err := s.DeleteFull(ctx, rootFullBackupID, unboundedWalBudgetMB)
		if err != nil {
			return total, err
		}

		total.WalSegments += summary.WalSegments
		total.Incrementals += summary.Incrementals
		total.HistoryFiles += summary.HistoryFiles
		total.BytesDeletedMB += summary.BytesDeletedMB
		total.ChainFullyDeleted = summary.ChainFullyDeleted

		if summary.ChainFullyDeleted {
			return total, nil
		}

		// Nothing left to delete and the FULL is still not gone means a peer
		// already removed it (idempotent no-op) — stop rather than spin.
		if summary.WalSegments == 0 && summary.Incrementals == 0 {
			return total, nil
		}
	}
}

// DeleteChainDependentsKeepFull strips a chain's INCRs and WAL but leaves the
// FULL row (and its history) in place — the FULL_BACKUPS LAST_N policy keeps the
// base FULL while shedding everything downstream of it.
func (s *Service) DeleteChainDependentsKeepFull(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
	walByteBudgetMB float64,
) (DeletedSummary, error) {
	return s.cascadeDelete(ctx, rootFullBackupID, walByteBudgetMB, true)
}

func (s *Service) DeleteOrphanWalSegmentsInSpan(
	ctx context.Context,
	spec DeleteOrphanWalSegmentsSpec,
) (int, float64, error) {
	var deletedRows int
	var deletedMB float64

	txErr := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.backupRepository.AcquireBackupAndOrphanCleanupLock(ctx, spec.DatabaseID); err != nil {
			return err
		}

		hasRunningFull, err := s.backupRepository.HasRunningFullBackup(ctx, spec.DatabaseID)
		if err != nil {
			return err
		}
		if hasRunningFull {
			return nil
		}

		chainCatalog, err := s.loadChainCatalog(ctx, spec.DatabaseID)
		if err != nil {
			return err
		}
		if !newWalOwnership(chainCatalog, uuid.Nil).isUnreferenced(spec.TimelineID, spec.Span) {
			return nil
		}

		rows, mb, _, err := s.deleteWalInSpansBudgeted(
			ctx,
			spec.DatabaseID,
			[]catalog.TimelineSpan{{TimelineID: spec.TimelineID, Span: spec.Span}},
			spec.WalByteBudgetMB,
			true,
		)
		deletedRows = rows
		deletedMB = mb

		return err
	})
	if txErr != nil {
		return 0, 0, txErr
	}

	return deletedRows, deletedMB, nil
}

// DeleteIncrementalCascade removes one incremental and every incremental that
// descends from it (its children, their children, …), leaves first so the
// RESTRICT FK on parent_backup_id is never violated. The chain's
// FULL and WAL are deliberately left intact — an incremental delete only sheds
// the sub-tree rooted at the target. Idempotent: a missing incremental is a
// no-op (a peer already deleted it).
func (s *Service) DeleteIncrementalCascade(
	ctx context.Context,
	incrementalID uuid.UUID,
) (DeletedSummary, error) {
	summary := DeletedSummary{}

	txErr := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		target, isFound, err := s.backupRepository.FindByID(ctx, incrementalID)
		if err != nil {
			return err
		}
		if !isFound {
			return nil
		}

		if _, _, err := s.backupRepository.FindByIDForUpdate(ctx, *target.RootFullBackupID); err != nil {
			return err
		}

		chain, err := s.backupRepository.FindAllByRootFull(ctx, *target.RootFullBackupID)
		if err != nil {
			return err
		}

		subtree := collectIncrementalSubtree(chain, incrementalID)

		count, mb, err := s.deleteIncrementalSet(ctx, reverseTopoOrderIncrementals(subtree))
		summary.Incrementals = count
		summary.BytesDeletedMB = mb

		return err
	})
	if txErr != nil {
		return DeletedSummary{}, txErr
	}

	return summary, nil
}

func (s *Service) ListBackups(ctx context.Context, query catalog.BackupListQuery) ([]catalog.BackupListRow, error) {
	return s.repository.ListBackups(ctx, query)
}

func (s *Service) cascadeDelete(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
	walByteBudgetMB float64,
	keepFull bool,
) (DeletedSummary, error) {
	summary := DeletedSummary{RootFullBackupID: rootFullBackupID}

	targetFull, _, err := s.backupRepository.FindByID(ctx, rootFullBackupID)
	if err != nil {
		return DeletedSummary{}, err
	}
	if targetFull == nil {
		return summary, nil
	}

	txErr := s.transactor.Transaction(ctx, func(ctx context.Context) error {
		if err := s.backupRepository.AcquireBackupAndOrphanCleanupLock(ctx,
			targetFull.DatabaseID,
		); err != nil {
			return err
		}

		lockedFull, isFound, err := s.backupRepository.FindByIDForUpdate(ctx, rootFullBackupID)
		if err != nil {
			return err
		}
		if !isFound {
			return nil
		}

		walPlan, err := s.getWalRemovalPlan(ctx, lockedFull, keepFull)
		if err != nil {
			return err
		}

		if len(walPlan.spans) > 0 {
			walRows, walBytes, budgetHit, err := s.deleteWalInSpansBudgeted(
				ctx,
				lockedFull.DatabaseID,
				walPlan.spans,
				walByteBudgetMB,
				false,
			)
			summary.WalSegments = walRows
			summary.BytesDeletedMB = walBytes
			if err != nil {
				return err
			}

			if budgetHit {
				// WAL budget exhausted before the chain emptied — commit the partial
				// deletion; the next tick re-queries the now-oldest rows.
				return nil
			}
		}

		incrementalCount, err := s.deleteIncrementals(ctx, rootFullBackupID)
		summary.Incrementals = incrementalCount
		if err != nil {
			return err
		}

		if keepFull {
			return nil
		}

		if !walPlan.hasRunningFull {
			historyCount, err := s.deleteOrphanedHistoryFiles(ctx, lockedFull)
			summary.HistoryFiles = historyCount
			if err != nil {
				return err
			}
		}

		if err := s.deleteFullArtifactAndRow(ctx, lockedFull); err != nil {
			return err
		}

		summary.ChainFullyDeleted = true

		return nil
	})
	if txErr != nil {
		return DeletedSummary{RootFullBackupID: rootFullBackupID}, txErr
	}

	return summary, nil
}

type walRemovalPlan struct {
	spans          []catalog.TimelineSpan
	hasRunningFull bool
}

// getWalRemovalPlan returns the WAL a chain's removal may delete: the chain's ownership span along
// its line. A running FULL may anchor any archived WAL, so nothing is deleted while one runs.
// Removing the FULL itself keeps its WAL when an older completed FULL's line still replays it; a
// FULL_BACKUPS cleanup that keeps the FULL sheds the owned WAL regardless.
func (s *Service) getWalRemovalPlan(ctx context.Context, full *catalog.Backup, keepFull bool) (walRemovalPlan, error) {
	hasRunningFull, err := s.backupRepository.HasRunningFullBackup(ctx, full.DatabaseID)
	if err != nil {
		return walRemovalPlan{}, err
	}
	if hasRunningFull {
		return walRemovalPlan{hasRunningFull: true}, nil
	}

	if full.StartLSN == nil {
		return walRemovalPlan{}, nil
	}

	chainCatalog, err := s.loadChainCatalog(ctx, full.DatabaseID)
	if err != nil {
		return walRemovalPlan{}, err
	}

	if !keepFull && hasPredecessorOnLine(chainCatalog, full) {
		return walRemovalPlan{}, nil
	}

	spans, err := getOwnedWalSpans(chainCatalog, full)
	if err != nil {
		return walRemovalPlan{}, err
	}

	return walRemovalPlan{spans: spans}, nil
}

// deleteWalInSpansBudgeted deletes the WAL fully inside each span in turn, sharing one budget.
func (s *Service) deleteWalInSpansBudgeted(
	ctx context.Context,
	databaseID uuid.UUID,
	spans []catalog.TimelineSpan,
	walByteBudgetMB float64,
	lockRows bool,
) (deletedRows int, deletedMB float64, budgetHit bool, err error) {
	for _, span := range spans {
		spanRows, spanMB, isBudgetHit, spanErr := s.deleteWalInSpanBudgeted(
			ctx,
			databaseID,
			span,
			walByteBudgetMB-deletedMB,
			lockRows,
		)
		deletedRows += spanRows
		deletedMB += spanMB
		if spanErr != nil || isBudgetHit {
			return deletedRows, deletedMB, isBudgetHit, spanErr
		}
	}

	return deletedRows, deletedMB, false, nil
}

// deleteWalInSpanBudgeted removes only WAL rows lying fully inside the span. The
// upper bound is the segment's end_lsn, not its start_lsn: a span ends at the
// next chain's start_lsn, which sits inside a segment, and that straddling
// segment carries the bytes the next chain replays from. Bounding on start_lsn
// would delete it along with the chain being pruned. Storage
// deletes are fail-closed: a transient DeleteFile error aborts the batch (the
// rows survive and retry) rather than orphaning an object. lockRows takes a
// row-level FOR UPDATE on each batch (orphan path, no anchor FULL to serialize
// on); when false the caller already holds the anchor FULL lock. Returns rows
// deleted, MB deleted, and whether the budget capped the run (more rows remain).
func (s *Service) deleteWalInSpanBudgeted(
	ctx context.Context,
	databaseID uuid.UUID,
	span catalog.TimelineSpan,
	walByteBudgetMB float64,
	lockRows bool,
) (deletedRows int, deletedMB float64, budgetHit bool, err error) {
	for {
		segments, findErr := s.walSegmentRepository.FindContained(
			ctx,
			catalog.ContainedWalQuery{
				DatabaseID: databaseID,
				TimelineID: span.TimelineID,
				Span:       span.Span,
				Limit:      maxWalDeleteBatchRows,
				ShouldLock: lockRows,
			},
		)
		if findErr != nil {
			return deletedRows, deletedMB, false, findErr
		}

		if len(segments) == 0 {
			return deletedRows, deletedMB, false, nil
		}

		for _, segment := range segments {
			if err = s.deleteWalSegment(ctx, segment); err != nil {
				return deletedRows, deletedMB, false, err
			}

			deletedRows++
			deletedMB += segment.CompressedSizeMb

			// Soft byte cap, checked per row: stop as soon as the budget is
			// reached so a sub-batch span cannot blow past it. Possibly-more
			// rows remain, so report budgetHit=true.
			if deletedMB >= walByteBudgetMB {
				return deletedRows, deletedMB, true, nil
			}
		}

		if len(segments) < maxWalDeleteBatchRows {
			return deletedRows, deletedMB, false, nil
		}
	}
}

func (s *Service) deleteIncrementals(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
) (int, error) {
	incrementals, err := s.backupRepository.FindAllByRootFull(ctx, rootFullBackupID)
	if err != nil {
		return 0, err
	}

	deleted, _, err := s.deleteIncrementalSet(ctx, reverseTopoOrderIncrementals(incrementals))

	return deleted, err
}

// deleteIncrementalSet deletes the given incrementals in the order received
// (callers pass them leaves-first), each artifact before its row. Returns the
// number of rows removed and the MB freed.
func (s *Service) deleteIncrementalSet(
	ctx context.Context,
	ordered []*catalog.Backup,
) (int, float64, error) {
	deleted := 0
	var deletedMB float64

	for _, incremental := range ordered {
		if incremental.FileName != nil {
			if err := s.requestFileDeletions(ctx, incremental.StorageID,
				*incremental.FileName,
				*incremental.FileName+metadataSuffix,
				incremental.GetManifestFileName(),
			); err != nil {
				return deleted, deletedMB, err
			}
		}

		if err := s.backupRepository.DeleteByID(ctx, incremental.ID); err != nil {
			return deleted, deletedMB, err
		}

		deleted++

		if incremental.BackupSizeMb != nil {
			deletedMB += *incremental.BackupSizeMb
		}
	}

	return deleted, deletedMB, nil
}

// collectIncrementalSubtree returns the target incremental plus every descendant
// (children, their children, …) found within chain, walking the
// parent_backup_id links breadth-first. Returns nil when the target
// is not in the chain. Order is not significant — callers re-order leaves-first.
func collectIncrementalSubtree(
	chain []*catalog.Backup,
	targetID uuid.UUID,
) []*catalog.Backup {
	childrenByParent := make(map[uuid.UUID][]*catalog.Backup)
	byID := make(map[uuid.UUID]*catalog.Backup, len(chain))

	for _, incremental := range chain {
		byID[incremental.ID] = incremental

		if incremental.ParentBackupID != nil {
			parent := *incremental.ParentBackupID
			childrenByParent[parent] = append(childrenByParent[parent], incremental)
		}
	}

	target, ok := byID[targetID]
	if !ok {
		return nil
	}

	subtree := []*catalog.Backup{target}
	queue := []uuid.UUID{targetID}

	for len(queue) > 0 {
		parentID := queue[0]
		queue = queue[1:]

		for _, child := range childrenByParent[parentID] {
			subtree = append(subtree, child)
			queue = append(queue, child.ID)
		}
	}

	return subtree
}

// deleteOrphanedHistoryFiles drops the history of the FULL's timeline once no other completed FULL
// has that timeline on its line. WAL of the timeline that no remaining line holds goes with it:
// without its history such WAL would look like a promoted timeline whose history is still pending,
// which the orphan sweep never removes.
func (s *Service) deleteOrphanedHistoryFiles(
	ctx context.Context,
	full *catalog.Backup,
) (int, error) {
	chainCatalog, err := s.loadChainCatalog(ctx, full.DatabaseID)
	if err != nil {
		return 0, err
	}

	ownership := newWalOwnership(chainCatalog, full.ID)
	if ownership.hasTimelineOnAnyLine(full.TimelineID) {
		return 0, nil
	}

	historyFiles, err := s.walHistoryRepository.FindAllByDatabaseTimeline(ctx, full.DatabaseID, full.TimelineID)
	if err != nil {
		return 0, err
	}

	if len(historyFiles) > 0 {
		if err := s.deleteUnreferencedTimelineWal(ctx, full, ownership); err != nil {
			return 0, err
		}
	}

	deleted := 0

	for _, historyFile := range historyFiles {
		if err := s.requestFileDeletions(ctx, historyFile.StorageID,
			historyFile.FileName, historyFile.FileName+metadataSuffix,
		); err != nil {
			return deleted, err
		}

		if err := s.walHistoryRepository.DeleteByID(ctx, historyFile.ID); err != nil {
			return deleted, err
		}

		deleted++
	}

	return deleted, nil
}

func (s *Service) deleteUnreferencedTimelineWal(
	ctx context.Context,
	full *catalog.Backup,
	ownership walOwnership,
) error {
	cursor := walmath.LSN(0)

	for {
		segments, err := s.walSegmentRepository.FindContained(ctx, catalog.ContainedWalQuery{
			DatabaseID: full.DatabaseID,
			TimelineID: full.TimelineID,
			Span:       catalog.LSNRange{Start: cursor, End: catalog.LSNMax},
			Limit:      maxWalDeleteBatchRows,
			ShouldLock: false,
		})
		if err != nil {
			return err
		}

		for _, segment := range segments {
			cursor = segment.StartLSN + 1
			if !ownership.isTimelineOnAnyLine(
				segment.TimelineID,
				catalog.LSNRange{Start: segment.StartLSN, End: segment.EndLSN},
			) {
				if err := s.deleteWalSegment(ctx, segment); err != nil {
					return err
				}
			}
		}

		if len(segments) < maxWalDeleteBatchRows {
			return nil
		}
	}
}

func (s *Service) deleteWalSegment(ctx context.Context, segment *catalog.WalSegment) error {
	if segment.FileName != nil {
		if err := s.requestFileDeletions(ctx, segment.StorageID,
			*segment.FileName, *segment.FileName+metadataSuffix,
		); err != nil {
			return err
		}
	}

	return s.walSegmentRepository.DeleteByID(ctx, segment.ID)
}

func (s *Service) deleteFullArtifactAndRow(
	ctx context.Context,
	full *catalog.Backup,
) error {
	if full.FileName != nil {
		if err := s.requestFileDeletions(ctx, full.StorageID,
			*full.FileName,
			*full.FileName+metadataSuffix,
			full.GetManifestFileName(),
		); err != nil {
			return err
		}
	}

	return s.backupRepository.DeleteByID(ctx, full.ID)
}

// requestFileDeletions records the obligation to remove these files in the transaction
// the context carries, the one that removes the rows naming them. Nothing reaches the provider
// here, so a rolled back cascade leaves every object intact and a storage outage
// does not decide whether a catalog row may go.
func (s *Service) requestFileDeletions(
	ctx context.Context,
	storageID uuid.UUID,
	fileNames ...string,
) error {
	references := make([]storages.StoredFileReference, 0, len(fileNames))

	for _, fileName := range fileNames {
		references = append(references,
			storages.StoredFileReference{StorageID: storageID, FileName: fileName})
	}

	if captured, isCapturing := ctx.Value(storageSwitchFilesKey{}).(*storageSwitchFiles); isCapturing {
		captured.references = append(captured.references, references...)
		return nil
	}
	return s.storages.RequestFileDeletions(ctx, references)
}

// reverseTopoOrderIncrementals orders incrementals leaves-first: an INCR is
// emitted before any INCR it names as parent, so deleting in this order respects
// the RESTRICT FK on parent_backup_id. Linear chains collapse to
// newest-first; the explicit walk also covers the rare branched case where two
// INCRs share one parent.
func reverseTopoOrderIncrementals(
	incrementals []*catalog.Backup,
) []*catalog.Backup {
	present := make(map[uuid.UUID]bool, len(incrementals))
	childCount := make(map[uuid.UUID]int, len(incrementals))

	for _, incremental := range incrementals {
		present[incremental.ID] = true
	}

	for _, incremental := range incrementals {
		if incremental.ParentBackupID != nil && present[*incremental.ParentBackupID] {
			childCount[*incremental.ParentBackupID]++
		}
	}

	ordered := make([]*catalog.Backup, 0, len(incrementals))
	remaining := incrementals

	for len(remaining) > 0 {
		var stillBlocked []*catalog.Backup
		progressed := false

		for _, incremental := range remaining {
			if childCount[incremental.ID] > 0 {
				stillBlocked = append(stillBlocked, incremental)

				continue
			}

			ordered = append(ordered, incremental)
			progressed = true

			if incremental.ParentBackupID != nil && present[*incremental.ParentBackupID] {
				childCount[*incremental.ParentBackupID]--
			}
		}

		if !progressed {
			// RESTRICT FKs make a cycle impossible; never loop forever anyway.
			ordered = append(ordered, stillBlocked...)

			break
		}

		remaining = stillBlocked
	}

	return ordered
}
