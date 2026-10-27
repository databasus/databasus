package physical_repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

type PhysicalInFlightBackupRepository struct{}

// ClaimSpec is the input to Claim: the cross-table single-in-flight reservation
// for one database. Bundling the fields (three of them UUIDs) into a struct keeps
// call sites unambiguous.
type ClaimSpec struct {
	DatabaseID uuid.UUID
	BackupType physical_enums.PhysicalBackupType
	BackupID   uuid.UUID
}

// Claim atomically reserves the cross-table single-in-flight slot for a database.
// Returns true on success, false on conflict (someone else already holds the
// slot). The caller passes its transaction handle so the claim commits together
// with the typed-table INSERT — that's the whole point of the slot existing in DB
// rather than in memory.
func (r *PhysicalInFlightBackupRepository) Claim(db *gorm.DB, spec ClaimSpec) (bool, error) {
	if spec.DatabaseID == uuid.Nil || spec.BackupID == uuid.Nil {
		return false, errors.New("database ID and backup ID are required")
	}

	result := db.Exec(
		`INSERT INTO backups_physical_in_flight (database_id, backup_type, backup_id, claimed_at)
		 VALUES (?, ?, ?, NOW())
		 ON CONFLICT (database_id) DO NOTHING`,
		spec.DatabaseID, spec.BackupType, spec.BackupID,
	)
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected > 0, nil
}

// FindAll returns every in-flight claim. Restart recovery reads them to fail and
// release each orphaned IN_PROGRESS backup whose goroutine died with the process.
func (r *PhysicalInFlightBackupRepository) FindAll() ([]*physical_models.PhysicalInFlightBackup, error) {
	var claims []*physical_models.PhysicalInFlightBackup

	if err := storage.GetDb().Find(&claims).Error; err != nil {
		return nil, err
	}

	return claims, nil
}

// Release deletes whatever in-flight claim a database currently holds,
// regardless of which backup owns it. Use it only for teardown — config
// disable, database removal, test cleanup — where the database is being stood
// down and any claim must go. A backuper finishing its own backup must use
// ReleaseOwned so it cannot delete a claim that a newer backup already took.
func (r *PhysicalInFlightBackupRepository) Release(databaseID uuid.UUID) error {
	return storage.
		GetDb().
		Delete(&physical_models.PhysicalInFlightBackup{}, "database_id = ?", databaseID).Error
}

// ReleaseOwned deletes the in-flight claim only while backupID still owns it.
// A backuper that finishes after the scheduler restarted and re-scheduled the
// database would otherwise delete the newer backup's claim (the claim is keyed
// by database_id), silently breaking the single-in-flight invariant; scoping
// the delete to backup_id turns that stale release into a no-op.
func (r *PhysicalInFlightBackupRepository) ReleaseOwned(databaseID, backupID uuid.UUID) error {
	return storage.
		GetDb().
		Delete(
			&physical_models.PhysicalInFlightBackup{},
			"database_id = ? AND backup_id = ?",
			databaseID,
			backupID,
		).Error
}

func (r *PhysicalInFlightBackupRepository) FindByDatabaseID(
	databaseID uuid.UUID,
) (*physical_models.PhysicalInFlightBackup, error) {
	var row physical_models.PhysicalInFlightBackup

	if err := storage.GetDb().Where("database_id = ?", databaseID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &row, nil
}
