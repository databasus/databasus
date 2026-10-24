package physical_repositories

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	"databasus-backend/internal/storage"
)

type PhysicalFullBackupRepository struct{}

func (r *PhysicalFullBackupRepository) Save(fullBackup *physical_models.PhysicalFullBackup) error {
	if fullBackup.DatabaseID == uuid.Nil || fullBackup.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	db := storage.GetDb()

	if fullBackup.ID == uuid.Nil {
		fullBackup.ID = uuid.New()
		if fullBackup.CreatedAt.IsZero() {
			fullBackup.CreatedAt = time.Now().UTC()
		}

		return db.Create(fullBackup).Error
	}

	return db.Save(fullBackup).Error
}

func (r *PhysicalFullBackupRepository) FindByID(id uuid.UUID) (*physical_models.PhysicalFullBackup, error) {
	var backup physical_models.PhysicalFullBackup

	if err := storage.GetDb().Where("id = ?", id).First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

func (r *PhysicalFullBackupRepository) FindCompletedNewestFirstByDatabase(
	databaseID uuid.UUID,
) ([]*physical_models.PhysicalFullBackup, error) {
	var backups []*physical_models.PhysicalFullBackup

	if err := storage.
		GetDb().
		Where("database_id = ? AND status = ?", databaseID, physical_enums.PhysicalBackupStatusCompleted).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *PhysicalFullBackupRepository) FindAllForGFS(
	databaseID uuid.UUID,
) ([]*physical_models.PhysicalFullBackup, error) {
	var backups []*physical_models.PhysicalFullBackup

	if err := storage.
		GetDb().
		Where("database_id = ? AND status = ?", databaseID, physical_enums.PhysicalBackupStatusCompleted).
		Order("created_at ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *PhysicalFullBackupRepository) UpdateStatus(
	id uuid.UUID,
	status physical_enums.PhysicalBackupStatus,
	errorReason *physical_enums.PhysicalBackupErrorReason,
) error {
	updates := map[string]any{"status": status, "error_reason": errorReason}

	if status == physical_enums.PhysicalBackupStatusCompleted {
		updates["completed_at"] = new(time.Now().UTC())
	}

	return storage.
		GetDb().
		Model(&physical_models.PhysicalFullBackup{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateProgress writes the in-flight size and duration for a still-running
// backup. It touches only those two columns and is gated on IN_PROGRESS — the
// same supersession guard as saveTerminalResultIfInProgress — so a late progress
// write can never resurrect a row a restart-recovery sweep already finalized.
func (r *PhysicalFullBackupRepository) UpdateProgress(id uuid.UUID, sizeMb float64, durationMs int64) error {
	return storage.
		GetDb().
		Model(&physical_models.PhysicalFullBackup{}).
		Where("id = ? AND status = ?", id, physical_enums.PhysicalBackupStatusInProgress).
		Updates(map[string]any{"backup_size_mb": sizeMb, "backup_duration_ms": durationMs}).Error
}

func (r *PhysicalFullBackupRepository) DeleteByID(id uuid.UUID) error {
	return storage.GetDb().Delete(&physical_models.PhysicalFullBackup{}, "id = ?", id).Error
}

func (r *PhysicalFullBackupRepository) FindNewestAnyStatusByDatabase(
	databaseID uuid.UUID,
	limit int,
) ([]*physical_models.PhysicalFullBackup, error) {
	var backups []*physical_models.PhysicalFullBackup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *PhysicalFullBackupRepository) FindLastFullAnyStatusByDatabase(
	databaseID uuid.UUID,
) (*physical_models.PhysicalFullBackup, error) {
	backups, err := r.FindNewestAnyStatusByDatabase(databaseID, 1)
	if err != nil {
		return nil, err
	}

	if len(backups) == 0 {
		return nil, nil
	}

	return backups[0], nil
}

func (r *PhysicalFullBackupRepository) FindAllByDatabaseOutsideStorage(
	databaseID uuid.UUID,
	storageID uuid.UUID,
) ([]*physical_models.PhysicalFullBackup, error) {
	var backups []*physical_models.PhysicalFullBackup

	if err := storage.
		GetDb().
		Where("database_id = ? AND storage_id <> ?", databaseID, storageID).
		Order("created_at ASC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *PhysicalFullBackupRepository) FindAllInProgress() ([]*physical_models.PhysicalFullBackup, error) {
	var backups []*physical_models.PhysicalFullBackup

	if err := storage.
		GetDb().
		Where("status = ?", physical_enums.PhysicalBackupStatusInProgress).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}
