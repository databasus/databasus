package catalog

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

// Receive times can differ from LSN order under concurrent uploads. Display
// boundaries use time order, but each WAL range must remain contiguous in LSN
// order. Gap markers use the complete catalog so filters cannot invent gaps.
func backupListItemsQuery(filterClause string) string {
	return `
		WITH committed_wal AS (
			SELECT id, timeline_id, start_lsn, end_lsn, wal_filename,
			       compressed_size_mb, received_at,
			       COALESCE(
			           LAG(end_lsn) OVER (PARTITION BY timeline_id ORDER BY start_lsn) <> start_lsn,
			           FALSE
			       ) AS is_gap_start
			FROM backups_physical_wal_segments
			WHERE database_id = ? AND file_name IS NOT NULL
		),
		merged AS (
			SELECT id, kind AS type, status, timeline_id, start_lsn, stop_lsn,
                   root_full_backup_id,
                   NULLIF(parent_backup_id, root_full_backup_id) AS parent_incremental_backup_id,
                   NULL::text AS wal_filename,
                   COALESCE(backup_size_mb, 0) AS size_mb,
                   fail_message, created_at, completed_at,
                   FALSE AS is_gap_start
            FROM backups_physical
            WHERE database_id = ?
            UNION ALL
			SELECT id, 'WAL', 'COMPLETED', timeline_id, start_lsn, end_lsn,
			       NULL::uuid, NULL::uuid, wal_filename,
			       compressed_size_mb,
			       NULL::text, received_at, NULL::timestamptz,
			       is_gap_start
			FROM committed_wal
		),
		filtered AS (
			SELECT * FROM merged` + filterClause + `
		),
		ordered AS (
			SELECT *,
			       LAG(type) OVER time_order AS previous_type,
			       LAG(timeline_id) OVER time_order AS previous_timeline_id
			FROM filtered
			WINDOW time_order AS (ORDER BY created_at, id)
		),
		grouped AS (
			SELECT *,
			       SUM(
			           CASE
			               WHEN type <> 'WAL'
			                    OR previous_type IS DISTINCT FROM 'WAL'
			                    OR previous_timeline_id <> timeline_id
			               THEN 1 ELSE 0
			           END
			       ) OVER (ORDER BY created_at, id ROWS UNBOUNDED PRECEDING) AS group_id
			FROM ordered
		),
		wal_ordered AS (
			SELECT *, LAG(stop_lsn) OVER (PARTITION BY group_id ORDER BY start_lsn) AS previous_stop_lsn
			FROM grouped
			WHERE type = 'WAL'
		),
		wal_ranges AS (
			SELECT *, SUM(CASE WHEN previous_stop_lsn IS DISTINCT FROM start_lsn THEN 1 ELSE 0 END)
			       OVER (PARTITION BY group_id ORDER BY start_lsn ROWS UNBOUNDED PRECEDING) AS wal_range_id
			FROM wal_ordered
		),
		items AS (
			SELECT id, type, status, timeline_id,
			       COALESCE(start_lsn::text, '') AS start_lsn,
			       COALESCE(stop_lsn::text, '') AS stop_lsn,
			       root_full_backup_id, parent_incremental_backup_id,
			       NULL::text AS from_wal_filename,
			       NULL::text AS to_wal_filename,
			       NULL::int AS wal_segments_count,
			       FALSE AS has_gap_before,
			       size_mb, fail_message, created_at, completed_at,
			       created_at AS sort_time
			FROM grouped
			WHERE type <> 'WAL'
			UNION ALL
			SELECT (ARRAY_AGG(id ORDER BY start_lsn))[1], 'WAL', 'COMPLETED', MIN(timeline_id),
			       MIN(start_lsn)::text, MAX(stop_lsn)::text,
			       NULL::uuid, NULL::uuid,
			       (ARRAY_AGG(wal_filename ORDER BY start_lsn))[1],
			       (ARRAY_AGG(wal_filename ORDER BY start_lsn DESC))[1],
			       COUNT(*)::int,
			       (ARRAY_AGG(is_gap_start ORDER BY start_lsn))[1],
			       SUM(size_mb), NULL::text, MIN(created_at), MAX(created_at),
			       MAX(created_at)
			FROM wal_ranges
			GROUP BY group_id, wal_range_id
		)
		SELECT id, type, status, timeline_id, start_lsn, stop_lsn,
		       root_full_backup_id, parent_incremental_backup_id,
		       from_wal_filename, to_wal_filename, wal_segments_count, has_gap_before,
		       size_mb, fail_message, created_at, completed_at, sort_time
		FROM items`
}

// ListBackups returns up to limit list items for the database newest-first,
// optionally narrowed by filter and starting after the given position. It
// groups and pages in SQL so a database with many WAL segments never loads its
// whole catalog into the process. ID breaks sort-time ties for a stable page
// boundary.
type BackupListQuery struct {
	DatabaseID uuid.UUID
	Filter     BackupListFilter
	After      *BackupListPosition
	Limit      int
}

func (r *Repository) ListBackups(ctx context.Context, querySpec BackupListQuery) ([]BackupListRow, error) {
	databaseID, filter, after, limit := querySpec.DatabaseID, querySpec.Filter, querySpec.After, querySpec.Limit
	var rows []BackupListRow

	filterClause, filterArgs := filter.buildClause()

	args := []any{databaseID, databaseID}
	args = append(args, filterArgs...)

	query := backupListItemsQuery(filterClause)

	if after != nil {
		query += `
		WHERE (sort_time, id) < (?, ?)`
		args = append(args, after.SortTime, after.ID)
	}

	query += `
		ORDER BY sort_time DESC, id DESC
		LIMIT ?`
	args = append(args, limit)

	err := database.Conn(ctx, r.db).Raw(query, args...).Scan(&rows).Error

	return rows, err
}

// GetDatabasesWithUnresolvedMismatch answers for the given databases. A mismatch stays unresolved until a
// full backup completes after it.
func (r *Repository) GetDatabasesWithUnresolvedMismatch(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]bool, error) {
	unresolvedDatabases := make(map[uuid.UUID]bool)
	if len(databaseIDs) == 0 {
		return unresolvedDatabases, nil
	}

	var rows []struct{ DatabaseID uuid.UUID }

	err := database.Conn(ctx, r.db).Raw(`
		SELECT database_id
		FROM backups_physical
		WHERE database_id = ANY(?::uuid[])
		GROUP BY database_id
		HAVING COUNT(*) FILTER (WHERE error_reason = @mismatch) > 0
			AND (
				COUNT(*) FILTER (WHERE status = @completed AND kind = 'FULL') = 0
				OR MAX(completed_at) FILTER (WHERE status = @completed AND kind = 'FULL')
					<= MAX(COALESCE(completed_at, created_at)) FILTER (WHERE error_reason = @mismatch)
			)`,
		pq.Array(databaseIDs),
		sql.Named("mismatch", BackupErrorSystemIdentifierMismatch),
		sql.Named("completed", BackupStatusCompleted),
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		unresolvedDatabases[row.DatabaseID] = true
	}

	return unresolvedDatabases, nil
}
