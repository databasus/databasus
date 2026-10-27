package physical_repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
)

func AcquireBackupAndOrphanCleanupLock(tx *gorm.DB, databaseID uuid.UUID) error {
	return tx.Exec(
		`SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text, 0))`,
		databaseID,
	).Error
}

func HasInFlightFullBackup(tx *gorm.DB, databaseID uuid.UUID) (bool, error) {
	var claim physical_models.PhysicalInFlightBackup

	err := tx.
		Where("database_id = ? AND backup_type = ?", databaseID, physical_enums.PhysicalBackupTypeFull).
		First(&claim).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func HasCompletedFullBackupCoveringWal(
	tx *gorm.DB,
	databaseID uuid.UUID,
	timelineID int,
	walEndLSN walmath.LSN,
) (bool, error) {
	var fullCount int64

	err := tx.Model(&physical_models.PhysicalFullBackup{}).
		Where(
			"database_id = ? AND timeline_id = ? AND status = ? AND start_lsn IS NOT NULL AND start_lsn < ?::pg_lsn",
			databaseID,
			timelineID,
			physical_enums.PhysicalBackupStatusCompleted,
			walEndLSN.String(),
		).
		Count(&fullCount).Error
	if err != nil {
		return false, err
	}

	return fullCount > 0, nil
}
