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

type StreamerClaim struct {
	DatabaseID       uuid.UUID
	ClaimID          uuid.UUID
	StalenessSeconds int
	Now              time.Time
}

func (r *WalStreamerRepository) ClaimIfClaimable(ctx context.Context, claim StreamerClaim) (bool, error) {
	databaseID := claim.DatabaseID
	now := claim.Now
	stalenessSeconds := claim.StalenessSeconds

	if databaseID == uuid.Nil {
		return false, errors.New("database ID is required")
	}

	var claimedIDs []uuid.UUID

	err := database.Conn(ctx, r.db).
		Raw(`
			INSERT INTO backups_physical_wal_streamers (database_id, started_at, last_heartbeat_at, status, claim_id)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT (database_id) DO UPDATE
			    SET claim_id=EXCLUDED.claim_id, started_at        = EXCLUDED.started_at,
			        last_heartbeat_at = EXCLUDED.last_heartbeat_at,
			        status            = EXCLUDED.status
			    WHERE backups_physical_wal_streamers.last_heartbeat_at < ?
			       OR backups_physical_wal_streamers.status = ?
			RETURNING database_id
		`,
			databaseID,
			now, now,
			WalStreamerStatusRunning, claim.ClaimID,
			now.Add(-time.Duration(stalenessSeconds)*time.Second),
			WalStreamerStatusFailed,
		).
		Scan(&claimedIDs).Error
	if err != nil {
		return false, err
	}

	return len(claimedIDs) == 1, nil
}

func (r *WalStreamerRepository) Heartbeat(ctx context.Context, claimID uuid.UUID, now time.Time) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("claim_id = ?", claimID).
		Update("last_heartbeat_at", now).Error
}

// A crash loop keeps the failing_since of its first failure, so the time a
// streamer has been failing survives every reclaim that fails again.
func (r *WalStreamerRepository) MarkFailed(ctx context.Context, claimID uuid.UUID, now time.Time) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("claim_id = ?", claimID).
		Updates(map[string]any{
			"status":        WalStreamerStatusFailed,
			"failing_since": gorm.Expr("COALESCE(failing_since, ?)", now),
		}).Error
}

// A deliberate stop (shutdown, streaming switched off) releases the row for
// reclaim like a failure does, but it is not one, so it leaves failing_since
// alone and a deploy does not read as a streamer that has been failing.
func (r *WalStreamerRepository) MarkStopped(ctx context.Context, claimID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("claim_id = ?", claimID).
		Update("status", WalStreamerStatusFailed).Error
}

func (r *WalStreamerRepository) ClearFailingSince(ctx context.Context, claimID uuid.UUID) error {
	return database.Conn(ctx, r.db).
		Model(&WalStreamer{}).
		Where("claim_id = ?", claimID).
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
