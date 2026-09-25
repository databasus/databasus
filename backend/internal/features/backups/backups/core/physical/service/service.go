package physical_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	"databasus-backend/internal/storage"
	util_encryption "databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/walmath"
)

var failedPhysicalBackupStatuses = []physical_enums.PhysicalBackupStatus{
	physical_enums.PhysicalBackupStatusError,
	physical_enums.PhysicalBackupStatusChainBroken,
}

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

// PhysicalBackupService owns the transactional, multi-table operations over the
// physical catalog that the cleaner needs: chain-cascade deletion (file before
// row, leaves before parents), usage accounting, and orphan-WAL deletion. It is
// the single seam through which the cleaner mutates catalog rows — the cleaner
// never issues raw DELETEs.
type PhysicalBackupService struct {
	fullBackupRepository *physical_repositories.PhysicalFullBackupRepository
	walSegmentRepository *physical_repositories.PhysicalWalSegmentRepository
	storageService       *storages.StorageService
	fileStore            *storage_files.Store
	fieldEncryptor       util_encryption.FieldEncryptor
	logger               *slog.Logger
}

// GetLatestCompletedFullBackup returns the most recent completed FULL backup for
// a database, or nil when the database has none. Telemetry uses its raw/compressed
// sizes as the physical database's size of record.
func (s *PhysicalBackupService) GetLatestCompletedFullBackup(
	databaseID uuid.UUID,
) (*physical_models.PhysicalFullBackup, error) {
	completedBackups, err := s.fullBackupRepository.FindCompletedNewestFirstByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	if len(completedBackups) == 0 {
		return nil, nil
	}

	return completedBackups[0], nil
}

// DeleteFull cascades one chain's deletion bounded by walByteBudgetMB. The
// order is WAL (oldest LSN first) → INCRs (leaves first) → orphaned history →
// the FULL itself, each object deleted from storage before its row. Idempotent:
// a missing FULL is a no-op (a peer already deleted it); a budget-capped call
// returns ChainFullyDeleted=false and the caller resumes next tick.
func (s *PhysicalBackupService) DeleteFull(
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
func (s *PhysicalBackupService) DeleteFullCompletely(
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
func (s *PhysicalBackupService) DeleteChainDependentsKeepFull(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
	walByteBudgetMB float64,
) (DeletedSummary, error) {
	return s.cascadeDelete(ctx, rootFullBackupID, walByteBudgetMB, true)
}

// DeleteWalSegmentsInSpan deletes orphan WAL — segments with no anchoring FULL —
// for one (database, timeline) over an LSN span, budget-bounded. There is no
// FULL to lock, so each batch is SELECT ... FOR UPDATE on the WAL rows directly.
func (s *PhysicalBackupService) DeleteWalSegmentsInSpan(
	ctx context.Context,
	databaseID uuid.UUID,
	timelineID int,
	span chain_view.LSNRange,
	walByteBudgetMB float64,
) (int, float64, error) {
	var deletedRows int
	var deletedMB float64

	txErr := storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, mb, _, err := s.deleteWalInSpanBudgeted(ctx, tx, databaseID, timelineID, span, walByteBudgetMB, true)
		deletedRows = rows
		deletedMB = mb

		return err
	})
	if txErr != nil {
		return 0, 0, txErr
	}

	return deletedRows, deletedMB, nil
}

func (s *PhysicalBackupService) DeleteOrphanWalSegmentsInSpan(
	ctx context.Context,
	spec DeleteOrphanWalSegmentsSpec,
) (int, float64, error) {
	var deletedRows int
	var deletedMB float64

	txErr := storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := physical_repositories.AcquireBackupAndOrphanCleanupLock(tx, spec.DatabaseID); err != nil {
			return err
		}

		hasInFlightFull, err := physical_repositories.HasInFlightFullBackup(tx, spec.DatabaseID)
		if err != nil {
			return err
		}
		if hasInFlightFull {
			return nil
		}

		hasAnchoringFull, err := physical_repositories.HasCompletedFullBackupCoveringWal(
			tx,
			spec.DatabaseID,
			spec.TimelineID,
			spec.Span.End,
		)
		if err != nil {
			return err
		}
		if hasAnchoringFull {
			return nil
		}

		rows, mb, _, err := s.deleteWalInSpanBudgeted(
			ctx,
			tx,
			spec.DatabaseID,
			spec.TimelineID,
			spec.Span,
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
// RESTRICT FK on parent_incremental_backup_id is never violated. The chain's
// FULL and WAL are deliberately left intact — an incremental delete only sheds
// the sub-tree rooted at the target. Idempotent: a missing incremental is a
// no-op (a peer already deleted it).
func (s *PhysicalBackupService) DeleteIncrementalCascade(
	ctx context.Context,
	incrementalID uuid.UUID,
) (DeletedSummary, error) {
	summary := DeletedSummary{}

	txErr := storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target physical_models.PhysicalIncrementalBackup

		findErr := tx.Where("id = ?", incrementalID).First(&target).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			s.logger.DebugContext(ctx, "incremental already deleted by peer", "incremental_backup_id", incrementalID)

			return nil
		}
		if findErr != nil {
			return findErr
		}

		// Lock the anchor FULL FOR UPDATE to serialize with the cleaner and the
		// WAL uploader, mirroring the full-chain cascade. The FULL may already be
		// gone in rare races; that is harmless here since we touch only INCR rows.
		var full physical_models.PhysicalFullBackup

		lockErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", target.RootFullBackupID).
			First(&full).Error
		if lockErr != nil && !errors.Is(lockErr, gorm.ErrRecordNotFound) {
			return lockErr
		}

		var chain []*physical_models.PhysicalIncrementalBackup

		if err := tx.
			Where("root_full_backup_id = ?", target.RootFullBackupID).
			Find(&chain).Error; err != nil {
			return err
		}

		subtree := collectIncrementalSubtree(chain, incrementalID)

		count, mb, err := s.deleteIncrementalSet(ctx, tx, reverseTopoOrderIncrementals(subtree))
		summary.Incrementals = count
		summary.BytesDeletedMB = mb

		return err
	})
	if txErr != nil {
		return DeletedSummary{}, txErr
	}

	return summary, nil
}

// DeleteWalSegment removes one WAL segment and every later WAL segment on its
// timeline up to (not including) the next FULL/INCR anchor LSN — the span
// [segment.start_lsn, nextAnchor). Deleting a WAL segment makes every PITR
// target after it unreachable until the next backup re-anchors replay, so the
// now-dead tail goes with it. Earlier WAL and all backups stay. Idempotent.
func (s *PhysicalBackupService) DeleteWalSegment(
	ctx context.Context,
	walSegmentID uuid.UUID,
) (int, float64, error) {
	segment, err := s.walSegmentRepository.FindByID(walSegmentID)
	if err != nil {
		return 0, 0, err
	}
	if segment == nil {
		return 0, 0, nil
	}

	end, err := s.findNextAnchorLSNAfter(ctx, segment.DatabaseID, segment.TimelineID, segment.StartLSN)
	if err != nil {
		return 0, 0, err
	}

	span := chain_view.LSNRange{Start: segment.StartLSN, End: end}

	return s.DeleteWalSegmentsInSpan(ctx, segment.DatabaseID, segment.TimelineID, span, unboundedWalBudgetMB)
}

// GetDependentsSummary counts a chain's dependents and total size without
// deleting anything. Idempotent and read-only.
func (s *PhysicalBackupService) GetDependentsSummary(rootFullBackupID uuid.UUID) (DependentsSummary, error) {
	targetFull, err := s.fullBackupRepository.FindByID(rootFullBackupID)
	if err != nil {
		return DependentsSummary{}, err
	}
	if targetFull == nil {
		return DependentsSummary{}, ErrFullNotFound
	}

	var segments []*physical_models.PhysicalWalSegment
	var incrementals []*physical_models.PhysicalIncrementalBackup
	var historyFiles []*physical_models.PhysicalWalHistoryFile
	var lockedFull physical_models.PhysicalFullBackup

	transactionErr := storage.GetDb().Transaction(func(tx *gorm.DB) error {
		if err := physical_repositories.AcquireBackupAndOrphanCleanupLock(tx, targetFull.DatabaseID); err != nil {
			return err
		}

		lockErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", rootFullBackupID).
			First(&lockedFull).Error
		if errors.Is(lockErr, gorm.ErrRecordNotFound) {
			return ErrFullNotFound
		}
		if lockErr != nil {
			return lockErr
		}

		walPlan, err := s.getFullRemovalWalPlan(tx, &lockedFull)
		if err != nil {
			return err
		}
		if walPlan.hasDeletableWal {
			if err := selectWalSegmentsFullyContainedInSpan(
				tx,
				lockedFull.DatabaseID,
				lockedFull.TimelineID,
				walPlan.span,
			).Find(&segments).Error; err != nil {
				return err
			}
		}

		if err := tx.
			Where("root_full_backup_id = ?", rootFullBackupID).
			Find(&incrementals).Error; err != nil {
			return err
		}

		if walPlan.hasInFlightFull {
			return nil
		}

		hasSurvivingFull, err := hasSurvivingCompletedFullOnTimeline(tx, &lockedFull)
		if err != nil {
			return err
		}
		if hasSurvivingFull {
			return nil
		}

		return tx.
			Where("database_id = ? AND timeline_id = ?", lockedFull.DatabaseID, lockedFull.TimelineID).
			Find(&historyFiles).Error
	})
	if transactionErr != nil {
		return DependentsSummary{}, transactionErr
	}

	summary := DependentsSummary{
		RootFullBackupID: rootFullBackupID,
		WalSegments:      len(segments),
		Incrementals:     len(incrementals),
	}

	if lockedFull.BackupSizeMb != nil {
		summary.TotalSizeMB += *lockedFull.BackupSizeMb
	}

	for _, segment := range segments {
		summary.TotalSizeMB += segment.CompressedSizeMb
	}

	for _, incremental := range incrementals {
		if incremental.BackupSizeMb != nil {
			summary.TotalSizeMB += *incremental.BackupSizeMb
		}
	}

	for _, historyFile := range historyFiles {
		summary.HistoryFiles++
		summary.TotalSizeMB += historyFile.CompressedSizeMb
	}

	return summary, nil
}

// GetTotalUsageMBByDatabase sums every COMPLETED FULL + COMPLETED INCR backup
// size and every WAL segment's compressed size for one database. Failed /
// canceled rows (NULL backup_size_mb) are excluded by the status filter.
func (s *PhysicalBackupService) GetTotalUsageMBByDatabase(databaseID uuid.UUID) (float64, error) {
	var totalMB float64

	err := storage.GetDb().Raw(`
		SELECT
		    COALESCE((SELECT SUM(backup_size_mb) FROM physical_full_backups
		              WHERE database_id = ? AND status = ?), 0)
		  + COALESCE((SELECT SUM(backup_size_mb) FROM physical_incremental_backups
		              WHERE database_id = ? AND status = ?), 0)
		  + COALESCE((SELECT SUM(compressed_size_mb) FROM physical_wal_segments
		              WHERE database_id = ?), 0)
	`,
		databaseID, physical_enums.PhysicalBackupStatusCompleted,
		databaseID, physical_enums.PhysicalBackupStatusCompleted,
		databaseID,
	).Scan(&totalMB).Error

	return totalMB, err
}

// GetLastBackupTimesByDatabaseIDs returns, per database, the newest successful
// physical backup time across FULL, INCREMENTAL and committed WAL - the value the
// UI shows as "last backup". FULL/INCR count only when COMPLETED; WAL counts once
// committed (file_name written). Databases with no qualifying backup are absent
// from the map. An empty input skips the query and returns an empty map.
func (s *PhysicalBackupService) GetLastBackupTimesByDatabaseIDs(
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]time.Time, error) {
	lastBackupTimes := make(map[uuid.UUID]time.Time, len(databaseIDs))

	if len(databaseIDs) == 0 {
		return lastBackupTimes, nil
	}

	type lastBackupRow struct {
		DatabaseID     uuid.UUID
		LastBackupTime time.Time
	}

	var rows []lastBackupRow

	err := storage.GetDb().Raw(`
		SELECT database_id, MAX(ts) AS last_backup_time
		FROM (
			SELECT database_id, MAX(created_at) AS ts FROM physical_full_backups
			    WHERE database_id IN ? AND status = ? GROUP BY database_id
			UNION ALL
			SELECT database_id, MAX(created_at) FROM physical_incremental_backups
			    WHERE database_id IN ? AND status = ? GROUP BY database_id
			UNION ALL
			SELECT database_id, MAX(received_at) FROM physical_wal_segments
			    WHERE database_id IN ? AND file_name IS NOT NULL GROUP BY database_id
		) merged
		GROUP BY database_id
	`,
		databaseIDs, physical_enums.PhysicalBackupStatusCompleted,
		databaseIDs, physical_enums.PhysicalBackupStatusCompleted,
		databaseIDs,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		lastBackupTimes[row.DatabaseID] = row.LastBackupTime
	}

	return lastBackupTimes, nil
}

// backupsUnionSubquery merges the FULL, incremental and committed-WAL tables into
// one result set with a common shape, projecting synthetic type ('FULL' /
// 'INCREMENTAL' / 'WAL') and, for WAL, a synthetic 'COMPLETED' status. The three
// database_id placeholders bind first; an outer WHERE (the list filter) and
// pagination wrap it. WAL ordering uses received_at as the row's created_at.
const backupsUnionSubquery = `
	FROM (
		SELECT id, 'FULL' AS type, status, timeline_id,
		       COALESCE(start_lsn::text, '') AS start_lsn,
		       COALESCE(stop_lsn::text, '') AS stop_lsn,
		       NULL::uuid AS root_full_backup_id,
		       NULL::uuid AS parent_incremental_backup_id,
		       NULL::text AS wal_filename,
		       COALESCE(backup_size_mb, 0) AS size_mb,
		       fail_message,
		       created_at, completed_at
		FROM physical_full_backups
		WHERE database_id = ?
		UNION ALL
		SELECT id, 'INCREMENTAL', status, timeline_id,
		       COALESCE(start_lsn::text, ''), COALESCE(stop_lsn::text, ''),
		       root_full_backup_id, parent_incremental_backup_id, NULL::text,
		       COALESCE(backup_size_mb, 0),
		       fail_message,
		       created_at, completed_at
		FROM physical_incremental_backups
		WHERE database_id = ?
		UNION ALL
		SELECT id, 'WAL', 'COMPLETED', timeline_id,
		       start_lsn::text, end_lsn::text,
		       NULL::uuid, NULL::uuid, wal_filename,
		       compressed_size_mb,
		       NULL::text,
		       received_at, NULL::timestamptz
		FROM physical_wal_segments
		WHERE database_id = ? AND file_name IS NOT NULL
	) backups`

// ListBackups returns one page of the database's backups — FULLs, incrementals
// and committed WAL segments merged into a single list newest-first by creation
// time (WAL ordered by received_at), optionally narrowed by filter. It UNIONs the
// three tables and paginates in SQL so a database with many WAL segments does not
// load its whole catalog per request. id breaks ties for a stable page boundary.
func (s *PhysicalBackupService) ListBackups(
	databaseID uuid.UUID,
	filter BackupListFilter,
	limit, offset int,
) ([]PhysicalBackupListRow, error) {
	var rows []PhysicalBackupListRow

	filterClause, filterArgs := filter.buildClause()

	args := []any{databaseID, databaseID, databaseID}
	args = append(args, filterArgs...)
	args = append(args, limit, offset)

	query := `
		SELECT id, type, status, timeline_id, start_lsn, stop_lsn,
		       root_full_backup_id, parent_incremental_backup_id, wal_filename,
		       size_mb, fail_message, created_at, completed_at` +
		backupsUnionSubquery + filterClause + `
		ORDER BY created_at DESC, id DESC
		LIMIT ? OFFSET ?`

	err := storage.GetDb().Raw(query, args...).Scan(&rows).Error

	return rows, err
}

// CountBackups returns the number of backups (every type) for a database that
// match the same filter ListBackups uses — the denominator for pagination.
func (s *PhysicalBackupService) CountBackups(
	databaseID uuid.UUID,
	filter BackupListFilter,
) (int64, error) {
	var total int64

	filterClause, filterArgs := filter.buildClause()

	args := []any{databaseID, databaseID, databaseID}
	args = append(args, filterArgs...)

	query := `SELECT COUNT(*)` + backupsUnionSubquery + filterClause

	err := storage.GetDb().Raw(query, args...).Scan(&total).Error

	return total, err
}

// WAL segments add to the size but not to the count: one streaming database
// uploads thousands of them, and the count is meant to read as FULL and INCR backups.
func (s *PhysicalBackupService) GetBackupTotalsByDatabaseIDs(
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]DatabasePhysicalBackupTotals, error) {
	totalsByDatabaseID := make(map[uuid.UUID]DatabasePhysicalBackupTotals, len(databaseIDs))

	if len(databaseIDs) == 0 {
		return totalsByDatabaseID, nil
	}

	var totals []DatabasePhysicalBackupTotals

	err := storage.GetDb().Raw(`
		SELECT database_id,
		       SUM(backups_count) AS backups_count,
		       SUM(completed_backups_count) AS completed_backups_count,
		       SUM(failed_backups_count) AS failed_backups_count,
		       SUM(completed_backup_size_mb) AS completed_backup_size_mb,
		       SUM(wal_size_mb) AS wal_size_mb
		FROM (
			SELECT database_id,
			       COUNT(*) AS backups_count,
			       COUNT(*) FILTER (WHERE status = ?) AS completed_backups_count,
			       COUNT(*) FILTER (WHERE status IN ?) AS failed_backups_count,
			       COALESCE(SUM(backup_size_mb) FILTER (WHERE status = ?), 0) AS completed_backup_size_mb,
			       0 AS wal_size_mb
			FROM physical_full_backups
			WHERE database_id IN ?
			GROUP BY database_id
			UNION ALL
			SELECT database_id,
			       COUNT(*),
			       COUNT(*) FILTER (WHERE status = ?),
			       COUNT(*) FILTER (WHERE status IN ?),
			       COALESCE(SUM(backup_size_mb) FILTER (WHERE status = ?), 0),
			       0
			FROM physical_incremental_backups
			WHERE database_id IN ?
			GROUP BY database_id
			UNION ALL
			SELECT database_id, 0, 0, 0, 0, COALESCE(SUM(compressed_size_mb), 0)
			FROM physical_wal_segments
			WHERE database_id IN ?
			GROUP BY database_id
		) merged
		GROUP BY database_id
	`,
		physical_enums.PhysicalBackupStatusCompleted, failedPhysicalBackupStatuses,
		physical_enums.PhysicalBackupStatusCompleted, databaseIDs,
		physical_enums.PhysicalBackupStatusCompleted, failedPhysicalBackupStatuses,
		physical_enums.PhysicalBackupStatusCompleted, databaseIDs,
		databaseIDs,
	).Scan(&totals).Error
	if err != nil {
		return nil, fmt.Errorf("sum physical backups by database: %w", err)
	}

	for _, databaseTotals := range totals {
		totalsByDatabaseID[databaseTotals.DatabaseID] = databaseTotals
	}

	return totalsByDatabaseID, nil
}

func (s *PhysicalBackupService) GetInstallationBackupTotals() (PhysicalBackupTotals, error) {
	var totals PhysicalBackupTotals

	err := storage.GetDb().Raw(`
		SELECT
		    (SELECT COUNT(*) FROM physical_full_backups)
		  + (SELECT COUNT(*) FROM physical_incremental_backups) AS backups_count,
		    COALESCE((SELECT SUM(backup_size_mb) FROM physical_full_backups WHERE status = ?), 0)
		  + COALESCE((SELECT SUM(backup_size_mb) FROM physical_incremental_backups WHERE status = ?), 0)
		    AS completed_backup_size_mb,
		    COALESCE((SELECT SUM(compressed_size_mb) FROM physical_wal_segments), 0) AS wal_size_mb
	`,
		physical_enums.PhysicalBackupStatusCompleted,
		physical_enums.PhysicalBackupStatusCompleted,
	).Scan(&totals).Error
	if err != nil {
		return PhysicalBackupTotals{}, fmt.Errorf("sum physical backups of the installation: %w", err)
	}

	return totals, nil
}

// findNextAnchorLSNAfter returns the smallest start_lsn among the database's
// FULL and INCR backups on the timeline that sits strictly after afterLSN — the
// boundary a WAL delete stops at. LSNMax when no later anchor exists (the tail
// of the timeline).
func (s *PhysicalBackupService) findNextAnchorLSNAfter(
	ctx context.Context,
	databaseID uuid.UUID,
	timelineID int,
	afterLSN walmath.LSN,
) (walmath.LSN, error) {
	row := storage.GetDb().WithContext(ctx).Raw(`
		SELECT MIN(start_lsn)::text FROM (
			SELECT start_lsn FROM physical_full_backups
			WHERE database_id = ? AND timeline_id = ? AND start_lsn IS NOT NULL AND start_lsn > ?::pg_lsn
			UNION ALL
			SELECT start_lsn FROM physical_incremental_backups
			WHERE database_id = ? AND timeline_id = ? AND start_lsn IS NOT NULL AND start_lsn > ?::pg_lsn
		) anchors
	`,
		databaseID, timelineID, afterLSN.String(),
		databaseID, timelineID, afterLSN.String(),
	).Row()

	var anchorText sql.NullString

	if err := row.Scan(&anchorText); err != nil {
		return 0, err
	}

	if !anchorText.Valid {
		return chain_view.LSNMax, nil
	}

	return walmath.ParseLSN(anchorText.String)
}

func (s *PhysicalBackupService) cascadeDelete(
	ctx context.Context,
	rootFullBackupID uuid.UUID,
	walByteBudgetMB float64,
	keepFull bool,
) (DeletedSummary, error) {
	summary := DeletedSummary{RootFullBackupID: rootFullBackupID}

	targetFull, err := s.fullBackupRepository.FindByID(rootFullBackupID)
	if err != nil {
		return DeletedSummary{}, err
	}
	if targetFull == nil {
		return summary, nil
	}

	txErr := storage.GetDb().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := physical_repositories.AcquireBackupAndOrphanCleanupLock(
			tx,
			targetFull.DatabaseID,
		); err != nil {
			return err
		}

		var lockedFull physical_models.PhysicalFullBackup

		// FOR UPDATE serializes with concurrent cleaner instances and the WAL
		// uploader (which locks the same anchor FULL before its INSERT).
		lockErr := tx.
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", rootFullBackupID).
			First(&lockedFull).Error
		if errors.Is(lockErr, gorm.ErrRecordNotFound) {
			s.logger.DebugContext(ctx, "chain already deleted by peer", "root_full_backup_id", rootFullBackupID)

			return nil
		}
		if lockErr != nil {
			return lockErr
		}

		var span chain_view.LSNRange
		var hasDeletableWal bool
		var err error

		var hasInFlightFull bool
		if keepFull {
			hasInFlightFull, err = physical_repositories.HasInFlightFullBackup(tx, lockedFull.DatabaseID)
			if err != nil {
				return err
			}
			if !hasInFlightFull {
				span, err = s.getWalOwnershipSpan(tx, &lockedFull)
				hasDeletableWal = true
			}
		} else {
			walPlan, planErr := s.getFullRemovalWalPlan(tx, &lockedFull)
			if planErr != nil {
				return planErr
			}
			span = walPlan.span
			hasDeletableWal = walPlan.hasDeletableWal
			hasInFlightFull = walPlan.hasInFlightFull
		}
		if err != nil {
			return err
		}

		if hasDeletableWal {
			walRows, walBytes, budgetHit, err := s.deleteWalInSpanBudgeted(ctx,
				tx, lockedFull.DatabaseID, lockedFull.TimelineID, span, walByteBudgetMB, false,
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

		incrementalCount, err := s.deleteIncrementals(ctx, tx, rootFullBackupID)
		summary.Incrementals = incrementalCount
		if err != nil {
			return err
		}

		if keepFull {
			return nil
		}

		if !hasInFlightFull {
			historyCount, err := s.deleteOrphanedHistoryFiles(ctx, tx, &lockedFull)
			summary.HistoryFiles = historyCount
			if err != nil {
				return err
			}
		}

		if err := s.deleteFullArtifactAndRow(ctx, tx, &lockedFull); err != nil {
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

type fullRemovalWalPlan struct {
	span            chain_view.LSNRange
	hasDeletableWal bool
	hasInFlightFull bool
}

func (s *PhysicalBackupService) getFullRemovalWalPlan(
	database *gorm.DB,
	full *physical_models.PhysicalFullBackup,
) (fullRemovalWalPlan, error) {
	hasInFlightFull, err := physical_repositories.HasInFlightFullBackup(database, full.DatabaseID)
	if err != nil {
		return fullRemovalWalPlan{}, err
	}
	if hasInFlightFull {
		return fullRemovalWalPlan{hasInFlightFull: true}, nil
	}

	if full.StartLSN == nil {
		return fullRemovalWalPlan{}, errors.New("cannot determine deletable WAL without FULL start_lsn")
	}

	var predecessorCount int64
	if err := database.Model(&physical_models.PhysicalFullBackup{}).
		Where(
			"database_id = ? AND timeline_id = ? AND status = ? AND start_lsn IS NOT NULL AND start_lsn < ?::pg_lsn",
			full.DatabaseID,
			full.TimelineID,
			physical_enums.PhysicalBackupStatusCompleted,
			full.StartLSN.String(),
		).
		Count(&predecessorCount).Error; err != nil {
		return fullRemovalWalPlan{}, err
	}
	if predecessorCount > 0 {
		return fullRemovalWalPlan{}, nil
	}

	span, err := s.getWalOwnershipSpan(database, full)
	if err != nil {
		return fullRemovalWalPlan{}, err
	}

	return fullRemovalWalPlan{span: span, hasDeletableWal: true}, nil
}

func hasSurvivingCompletedFullOnTimeline(
	database *gorm.DB,
	full *physical_models.PhysicalFullBackup,
) (bool, error) {
	var survivingFullCount int64

	if err := database.
		Model(&physical_models.PhysicalFullBackup{}).
		Where(
			"database_id = ? AND timeline_id = ? AND status = ? AND id != ?",
			full.DatabaseID,
			full.TimelineID,
			physical_enums.PhysicalBackupStatusCompleted,
			full.ID,
		).
		Count(&survivingFullCount).Error; err != nil {
		return false, err
	}

	return survivingFullCount > 0, nil
}

func (s *PhysicalBackupService) getWalOwnershipSpan(
	database *gorm.DB,
	full *physical_models.PhysicalFullBackup,
) (chain_view.LSNRange, error) {
	if full.StartLSN == nil {
		return chain_view.LSNRange{}, errors.New("cannot determine WAL ownership without FULL start_lsn")
	}

	var successor physical_models.PhysicalFullBackup
	findSuccessorErr := database.
		Where(
			"database_id = ? AND timeline_id = ? AND status = ? AND start_lsn > ?::pg_lsn",
			full.DatabaseID,
			full.TimelineID,
			physical_enums.PhysicalBackupStatusCompleted,
			full.StartLSN.String(),
		).
		Order("start_lsn ASC").
		First(&successor).Error
	if findSuccessorErr != nil && !errors.Is(findSuccessorErr, gorm.ErrRecordNotFound) {
		return chain_view.LSNRange{}, findSuccessorErr
	}

	spanEnd := chain_view.LSNMax
	if findSuccessorErr == nil && successor.StartLSN != nil {
		spanEnd = *successor.StartLSN
	}

	return chain_view.LSNRange{Start: *full.StartLSN, End: spanEnd}, nil
}

func selectWalSegmentsFullyContainedInSpan(
	database *gorm.DB,
	databaseID uuid.UUID,
	timelineID int,
	span chain_view.LSNRange,
) *gorm.DB {
	return database.Where(
		"database_id = ? AND timeline_id = ? AND start_lsn >= ?::pg_lsn AND end_lsn <= ?::pg_lsn",
		databaseID,
		timelineID,
		span.Start.String(),
		span.End.String(),
	)
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
func (s *PhysicalBackupService) deleteWalInSpanBudgeted(
	ctx context.Context,
	tx *gorm.DB,
	databaseID uuid.UUID,
	timelineID int,
	span chain_view.LSNRange,
	walByteBudgetMB float64,
	lockRows bool,
) (deletedRows int, deletedMB float64, budgetHit bool, err error) {
	for {
		var segments []*physical_models.PhysicalWalSegment

		batchQuery := selectWalSegmentsFullyContainedInSpan(tx, databaseID, timelineID, span).
			Order("start_lsn ASC").
			Limit(maxWalDeleteBatchRows)

		if lockRows {
			batchQuery = batchQuery.Clauses(clause.Locking{Strength: "UPDATE"})
		}

		if err = batchQuery.Find(&segments).Error; err != nil {
			return deletedRows, deletedMB, false, err
		}

		if len(segments) == 0 {
			return deletedRows, deletedMB, false, nil
		}

		for _, segment := range segments {
			if segment.FileName != nil {
				if delErr := s.requestFileDeletions(ctx, tx, segment.StorageID,
					*segment.FileName, *segment.FileName+metadataSuffix,
				); delErr != nil {
					return deletedRows, deletedMB, false, delErr
				}
			}

			if err = tx.Delete(&physical_models.PhysicalWalSegment{}, "id = ?", segment.ID).Error; err != nil {
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

func (s *PhysicalBackupService) deleteIncrementals(
	ctx context.Context,
	tx *gorm.DB,
	rootFullBackupID uuid.UUID,
) (int, error) {
	var incrementals []*physical_models.PhysicalIncrementalBackup

	if err := tx.
		Where("root_full_backup_id = ?", rootFullBackupID).
		Find(&incrementals).Error; err != nil {
		return 0, err
	}

	deleted, _, err := s.deleteIncrementalSet(ctx, tx, reverseTopoOrderIncrementals(incrementals))

	return deleted, err
}

// deleteIncrementalSet deletes the given incrementals in the order received
// (callers pass them leaves-first), each artifact before its row. Returns the
// number of rows removed and the MB freed.
func (s *PhysicalBackupService) deleteIncrementalSet(
	ctx context.Context,
	tx *gorm.DB,
	ordered []*physical_models.PhysicalIncrementalBackup,
) (int, float64, error) {
	deleted := 0
	var deletedMB float64

	for _, incremental := range ordered {
		if incremental.FileName != nil {
			if err := s.requestFileDeletions(ctx, tx, incremental.StorageID,
				*incremental.FileName,
				*incremental.FileName+metadataSuffix,
				valueOrEmptyString(incremental.ManifestFileName),
			); err != nil {
				return deleted, deletedMB, err
			}
		}

		if err := tx.Delete(&physical_models.PhysicalIncrementalBackup{}, "id = ?", incremental.ID).Error; err != nil {
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
// parent_incremental_backup_id links breadth-first. Returns nil when the target
// is not in the chain. Order is not significant — callers re-order leaves-first.
func collectIncrementalSubtree(
	chain []*physical_models.PhysicalIncrementalBackup,
	targetID uuid.UUID,
) []*physical_models.PhysicalIncrementalBackup {
	childrenByParent := make(map[uuid.UUID][]*physical_models.PhysicalIncrementalBackup)
	byID := make(map[uuid.UUID]*physical_models.PhysicalIncrementalBackup, len(chain))

	for _, incremental := range chain {
		byID[incremental.ID] = incremental

		if incremental.ParentIncrementalBackupID != nil {
			parent := *incremental.ParentIncrementalBackupID
			childrenByParent[parent] = append(childrenByParent[parent], incremental)
		}
	}

	target, ok := byID[targetID]
	if !ok {
		return nil
	}

	subtree := []*physical_models.PhysicalIncrementalBackup{target}
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

// deleteOrphanedHistoryFiles drops the history files of the FULL's timeline only
// when no other COMPLETED FULL survives on that timeline — otherwise the history
// still anchors a living chain.
func (s *PhysicalBackupService) deleteOrphanedHistoryFiles(
	ctx context.Context,
	tx *gorm.DB,
	full *physical_models.PhysicalFullBackup,
) (int, error) {
	hasSurvivingFull, err := hasSurvivingCompletedFullOnTimeline(tx, full)
	if err != nil {
		return 0, err
	}

	if hasSurvivingFull {
		return 0, nil
	}

	var historyFiles []*physical_models.PhysicalWalHistoryFile

	if err := tx.
		Where("database_id = ? AND timeline_id = ?", full.DatabaseID, full.TimelineID).
		Find(&historyFiles).Error; err != nil {
		return 0, err
	}

	deleted := 0

	for _, historyFile := range historyFiles {
		if err := s.requestFileDeletions(ctx, tx, historyFile.StorageID,
			historyFile.FileName, historyFile.FileName+metadataSuffix,
		); err != nil {
			return deleted, err
		}

		if err := tx.Delete(&physical_models.PhysicalWalHistoryFile{}, "id = ?", historyFile.ID).Error; err != nil {
			return deleted, err
		}

		deleted++
	}

	return deleted, nil
}

func (s *PhysicalBackupService) deleteFullArtifactAndRow(
	ctx context.Context,
	tx *gorm.DB,
	full *physical_models.PhysicalFullBackup,
) error {
	if full.FileName != nil {
		if err := s.requestFileDeletions(ctx, tx, full.StorageID,
			*full.FileName,
			*full.FileName+metadataSuffix,
			valueOrEmptyString(full.ManifestFileName),
		); err != nil {
			return err
		}
	}

	return tx.Delete(&physical_models.PhysicalFullBackup{}, "id = ?", full.ID).Error
}

// requestFileDeletions records the obligation to remove these files in the same
// transaction that removes the rows naming them. Nothing reaches the provider
// here, so a rolled back cascade leaves every object intact and a storage outage
// does not decide whether a catalog row may go.
func (s *PhysicalBackupService) requestFileDeletions(
	ctx context.Context,
	tx *gorm.DB,
	storageID uuid.UUID,
	fileNames ...string,
) error {
	references := make([]storage_files.StoredFileReference, 0, len(fileNames))

	for _, fileName := range fileNames {
		references = append(references,
			storage_files.StoredFileReference{StorageID: storageID, FileName: fileName})
	}

	return s.fileStore.RequestFileDeletions(ctx, tx, references)
}

// reverseTopoOrderIncrementals orders incrementals leaves-first: an INCR is
// emitted before any INCR it names as parent, so deleting in this order respects
// the RESTRICT FK on parent_incremental_backup_id. Linear chains collapse to
// newest-first; the explicit walk also covers the rare branched case where two
// INCRs share one parent.
func reverseTopoOrderIncrementals(
	incrementals []*physical_models.PhysicalIncrementalBackup,
) []*physical_models.PhysicalIncrementalBackup {
	present := make(map[uuid.UUID]bool, len(incrementals))
	childCount := make(map[uuid.UUID]int, len(incrementals))

	for _, incremental := range incrementals {
		present[incremental.ID] = true
	}

	for _, incremental := range incrementals {
		if incremental.ParentIncrementalBackupID != nil && present[*incremental.ParentIncrementalBackupID] {
			childCount[*incremental.ParentIncrementalBackupID]++
		}
	}

	ordered := make([]*physical_models.PhysicalIncrementalBackup, 0, len(incrementals))
	remaining := incrementals

	for len(remaining) > 0 {
		var stillBlocked []*physical_models.PhysicalIncrementalBackup
		progressed := false

		for _, incremental := range remaining {
			if childCount[incremental.ID] > 0 {
				stillBlocked = append(stillBlocked, incremental)

				continue
			}

			ordered = append(ordered, incremental)
			progressed = true

			if incremental.ParentIncrementalBackupID != nil && present[*incremental.ParentIncrementalBackupID] {
				childCount[*incremental.ParentIncrementalBackupID]--
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

func valueOrEmptyString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
