package physical_repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

type PhysicalWalStreamerRepository struct{}

// Idempotent — supervisor reclaiming a previously-failed streamer is a
// normal flow, so an existing row bumps the heartbeat and flips status back
// to RUNNING instead of returning a conflict.
func (r *PhysicalWalStreamerRepository) Claim(databaseID uuid.UUID) error {
	if databaseID == uuid.Nil {
		return errors.New("database ID is required")
	}

	return storage.
		GetDb().
		Exec(`
			INSERT INTO backups_physical_wal_streamers (database_id, started_at, last_heartbeat_at, status)
			VALUES (?, NOW(), NOW(), ?)
			ON CONFLICT (database_id) DO UPDATE
			    SET started_at        = NOW(),
			        last_heartbeat_at = NOW(),
			        status            = EXCLUDED.status
		`, databaseID, physical_enums.PhysicalWalStreamerStatusRunning).Error
}

// ClaimIfClaimable atomically claims the streamer for databaseID only when it is
// claimable: no row yet, the row is FAILED, or its heartbeat is older than
// stalenessSeconds (the previous owner died). Returns true when this caller won
// the claim. The conditional ON CONFLICT ... WHERE is the claim lock — a fresh,
// RUNNING, recently-heartbeated row with a live owner yields no RETURNING row,
// so the claim fails without disturbing the incumbent.
func (r *PhysicalWalStreamerRepository) ClaimIfClaimable(
	databaseID uuid.UUID,
	stalenessSeconds int,
) (bool, error) {
	if databaseID == uuid.Nil {
		return false, errors.New("database ID is required")
	}

	var claimedIDs []uuid.UUID

	err := storage.
		GetDb().
		Raw(`
			INSERT INTO backups_physical_wal_streamers (database_id, started_at, last_heartbeat_at, status)
			VALUES (?, NOW(), NOW(), ?)
			ON CONFLICT (database_id) DO UPDATE
			    SET started_at        = NOW(),
			        last_heartbeat_at = NOW(),
			        status            = EXCLUDED.status
			    WHERE backups_physical_wal_streamers.last_heartbeat_at < NOW() - make_interval(secs => ?)
			       OR backups_physical_wal_streamers.status = ?
			RETURNING database_id
		`,
			databaseID,
			physical_enums.PhysicalWalStreamerStatusRunning,
			float64(stalenessSeconds),
			physical_enums.PhysicalWalStreamerStatusFailed,
		).
		Scan(&claimedIDs).Error
	if err != nil {
		return false, err
	}

	return len(claimedIDs) == 1, nil
}

// Heartbeat bumps last_heartbeat_at using the source-of-truth PG clock (NOW()),
// NOT a Go-side time.Now(): the staleness comparisons in ClaimIfClaimable /
// MarkStaleRunningFailed are also PG-side, so writing the heartbeat with the
// streaming process's local clock would let clock skew falsely reclaim a live
// streamer or keep a dead one alive.
func (r *PhysicalWalStreamerRepository) Heartbeat(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Model(&physical_models.PhysicalWalStreamer{}).
		Where("database_id = ?", databaseID).
		Update("last_heartbeat_at", gorm.Expr("NOW()")).Error
}

// A crash loop keeps the failing_since of its first failure, so the time a
// streamer has been failing survives every reclaim that fails again.
func (r *PhysicalWalStreamerRepository) MarkFailed(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Model(&physical_models.PhysicalWalStreamer{}).
		Where("database_id = ?", databaseID).
		Updates(map[string]any{
			"status":        physical_enums.PhysicalWalStreamerStatusFailed,
			"failing_since": gorm.Expr("COALESCE(failing_since, NOW())"),
		}).Error
}

// A deliberate stop (shutdown, streaming switched off) releases the row for
// reclaim like a failure does, but it is not one, so it leaves failing_since
// alone and a deploy does not read as a streamer that has been failing.
func (r *PhysicalWalStreamerRepository) MarkStopped(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Model(&physical_models.PhysicalWalStreamer{}).
		Where("database_id = ?", databaseID).
		Update("status", physical_enums.PhysicalWalStreamerStatusFailed).Error
}

func (r *PhysicalWalStreamerRepository) ClearFailingSince(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Model(&physical_models.PhysicalWalStreamer{}).
		Where("database_id = ?", databaseID).
		Where("failing_since IS NOT NULL").
		Update("failing_since", nil).Error
}

func (r *PhysicalWalStreamerRepository) MarkStaleRunningFailed(stalenessSeconds int) (int64, error) {
	result := storage.
		GetDb().
		Model(&physical_models.PhysicalWalStreamer{}).
		Where("status = ?", physical_enums.PhysicalWalStreamerStatusRunning).
		Where("last_heartbeat_at < NOW() - make_interval(secs => ?)", float64(stalenessSeconds)).
		Updates(map[string]any{
			"status":        physical_enums.PhysicalWalStreamerStatusFailed,
			"failing_since": gorm.Expr("COALESCE(failing_since, NOW())"),
		})

	return result.RowsAffected, result.Error
}

func (r *PhysicalWalStreamerRepository) FindByDatabaseID(
	databaseID uuid.UUID,
) (*physical_models.PhysicalWalStreamer, error) {
	var row physical_models.PhysicalWalStreamer

	if err := storage.GetDb().Where("database_id = ?", databaseID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &row, nil
}

func (r *PhysicalWalStreamerRepository) DeleteByDatabaseID(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Delete(&physical_models.PhysicalWalStreamer{}, "database_id = ?", databaseID).Error
}
