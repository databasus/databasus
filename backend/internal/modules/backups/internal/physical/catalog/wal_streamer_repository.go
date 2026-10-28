package catalog

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type WalStreamerRepository struct{ db *gorm.DB }

func NewWalStreamerRepository(db *gorm.DB) *WalStreamerRepository {
	return &WalStreamerRepository{db: db}
}

// A manager reclaiming a failed streamer is a
// normal flow, so an existing row bumps the heartbeat and flips status back
// to RUNNING instead of returning a conflict.
func (r *WalStreamerRepository) Claim(ctx context.Context, databaseID uuid.UUID, now time.Time) error {
	if databaseID == uuid.Nil {
		return errors.New("database ID is required")
	}

	return database.Conn(ctx, r.db).
		Exec(`
			INSERT INTO backups_physical_wal_streamers (database_id, started_at, last_heartbeat_at, status)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (database_id) DO UPDATE
			    SET started_at        = EXCLUDED.started_at,
			        last_heartbeat_at = EXCLUDED.last_heartbeat_at,
			        status            = EXCLUDED.status
		`, databaseID, now, now, WalStreamerStatusRunning).Error
}

// ClaimIfClaimable atomically claims the streamer for databaseID only when it is
// claimable: no row yet, the row is FAILED, or its heartbeat is older than
// stalenessSeconds (the previous owner died). Returns true when this caller won
// the claim. The conditional ON CONFLICT ... WHERE is the claim lock — a fresh,
// RUNNING, recently-heartbeated row with a live owner yields no RETURNING row,
// so the claim fails without disturbing the incumbent.
func (r *WalStreamerRepository) ClaimIfClaimable(ctx context.Context, databaseID uuid.UUID,
	stalenessSeconds int, now time.Time,
) (bool, error) {
	if databaseID == uuid.Nil {
		return false, errors.New("database ID is required")
	}

	var claimedIDs []uuid.UUID

	err := database.Conn(ctx, r.db).
		Raw(`
			INSERT INTO backups_physical_wal_streamers (database_id, started_at, last_heartbeat_at, status)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (database_id) DO UPDATE
			    SET started_at        = EXCLUDED.started_at,
			        last_heartbeat_at = EXCLUDED.last_heartbeat_at,
			        status            = EXCLUDED.status
			    WHERE backups_physical_wal_streamers.last_heartbeat_at < ?
			       OR backups_physical_wal_streamers.status = ?
			RETURNING database_id
		`,
			databaseID,
			now, now,
			WalStreamerStatusRunning,
			now.Add(-time.Duration(stalenessSeconds)*time.Second),
			WalStreamerStatusFailed,
		).
		Scan(&claimedIDs).Error
	if err != nil {
		return false, err
	}

	return len(claimedIDs) == 1, nil
}

func (r *WalStreamerRepository) Heartbeat(ctx context.Context, databaseID uuid.UUID, now time.Time) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("database_id = ?", databaseID).
		Update("last_heartbeat_at", now).Error
}

// A crash loop keeps the failing_since of its first failure, so the time a
// streamer has been failing survives every reclaim that fails again.
func (r *WalStreamerRepository) MarkFailed(ctx context.Context, databaseID uuid.UUID, now time.Time) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("database_id = ?", databaseID).
		Updates(map[string]any{
			"status":        WalStreamerStatusFailed,
			"failing_since": gorm.Expr("COALESCE(failing_since, ?)", now),
		}).Error
}

// A deliberate stop (shutdown, streaming switched off) releases the row for
// reclaim like a failure does, but it is not one, so it leaves failing_since
// alone and a deploy does not read as a streamer that has been failing.
func (r *WalStreamerRepository) MarkStopped(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("database_id = ?", databaseID).
		Update("status", WalStreamerStatusFailed).Error
}

func (r *WalStreamerRepository) ClearFailingSince(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("database_id = ?", databaseID).
		Where("failing_since IS NOT NULL").
		Update("failing_since", nil).Error
}

func (r *WalStreamerRepository) MarkStaleRunningFailed(
	ctx context.Context,
	stalenessSeconds int,
	now time.Time,
) (int64, error) {
	result := database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("status = ?", WalStreamerStatusRunning).
		Where("last_heartbeat_at < ?", now.Add(-time.Duration(stalenessSeconds)*time.Second)).
		Updates(map[string]any{
			"status":        WalStreamerStatusFailed,
			"failing_since": gorm.Expr("COALESCE(failing_since, ?)", now),
		})

	return result.RowsAffected, result.Error
}

func (r *WalStreamerRepository) FindByDatabaseID(ctx context.Context, databaseID uuid.UUID,
) (*WalStreamer, bool, error) {
	var row WalStreamer

	if err := database.Conn(ctx, r.db).Where("database_id = ?", databaseID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}

		return nil, false, err
	}

	return &row, true, nil
}

func (r *WalStreamerRepository) DeleteByDatabaseID(ctx context.Context, databaseID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Delete(&WalStreamer{}, "database_id = ?", databaseID).Error
}
