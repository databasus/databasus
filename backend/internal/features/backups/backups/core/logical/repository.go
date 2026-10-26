package backups_core_logical

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/storage"
)

type BackupRepository struct{}

func (r *BackupRepository) Save(backup *LogicalBackup) error {
	return r.SaveInTransaction(storage.GetDb(), backup)
}

// A stored file is kept only when the row naming it commits, so publication and the
// receipts it spends have to share one transaction.
func (r *BackupRepository) SaveInTransaction(tx *gorm.DB, backup *LogicalBackup) error {
	if backup.DatabaseID == uuid.Nil || backup.StorageID == uuid.Nil {
		return errors.New("database ID and storage ID are required")
	}

	isNew := backup.ID == uuid.Nil
	if isNew {
		backup.ID = uuid.New()

		return tx.Create(backup).Error
	}

	return tx.Save(backup).Error
}

func (r *BackupRepository) UpdateRestoreVerificationStatus(
	backupID uuid.UUID,
	status RestoreVerificationStatus,
) error {
	if backupID == uuid.Nil {
		return errors.New("backup ID is required")
	}

	return storage.
		GetDb().
		Model(&LogicalBackup{}).
		Where("id = ?", backupID).
		Update("restore_verification_status", status).
		Error
}

func (r *BackupRepository) ExistsCompletedSince(
	databaseID uuid.UUID,
	since time.Time,
) (bool, error) {
	var count int64

	if err := storage.
		GetDb().
		Model(&LogicalBackup{}).
		Where("database_id = ? AND status = ? AND created_at >= ?",
			databaseID, BackupStatusCompleted, since).
		Limit(1).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *BackupRepository) FindByDatabaseID(databaseID uuid.UUID) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByDatabaseIDWithLimit(
	databaseID uuid.UUID,
	limit int,
) ([]*LogicalBackup, error) {
	if limit <= 0 {
		return nil, errors.New("limit must be greater than 0")
	}

	var backups []*LogicalBackup

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

func (r *BackupRepository) FindByStorageID(storageID uuid.UUID) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("storage_id = ?", storageID).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindLastByDatabaseID(databaseID uuid.UUID) (*LogicalBackup, error) {
	var backup LogicalBackup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		First(&backup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

func (r *BackupRepository) FindByID(id uuid.UUID) (*LogicalBackup, error) {
	var backup LogicalBackup

	if err := storage.
		GetDb().
		Where("id = ?", id).
		First(&backup).Error; err != nil {
		return nil, err
	}

	return &backup, nil
}

func (r *BackupRepository) FindByStatus(status BackupStatus) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("status = ?", status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByStorageIdAndStatus(
	storageID uuid.UUID,
	status BackupStatus,
) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("storage_id = ? AND status = ?", storageID, status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByDatabaseIdAndStatus(
	databaseID uuid.UUID,
	status BackupStatus,
) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("database_id = ? AND status = ?", databaseID, status).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) DeleteByID(id uuid.UUID) error {
	return r.DeleteByIDInTransaction(storage.GetDb(), id)
}

func (r *BackupRepository) DeleteByIDInTransaction(tx *gorm.DB, id uuid.UUID) error {
	return tx.Delete(&LogicalBackup{}, "id = ?", id).Error
}

func (r *BackupRepository) FindBackupsBeforeDate(
	databaseID uuid.UUID,
	date time.Time,
) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("database_id = ? AND created_at < ?", databaseID, date).
		Order("created_at DESC").
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindByDatabaseIDWithPagination(
	databaseID uuid.UUID,
	limit, offset int,
) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) CountByDatabaseID(databaseID uuid.UUID) (int64, error) {
	var count int64

	if err := storage.
		GetDb().
		Model(&LogicalBackup{}).
		Where("database_id = ?", databaseID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *BackupRepository) FindOldestByDatabaseExcludingInProgress(
	databaseID uuid.UUID,
	limit int,
) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	if err := storage.
		GetDb().
		Where("database_id = ? AND status != ?", databaseID, BackupStatusInProgress).
		Order("created_at ASC").
		Limit(limit).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) FindLatestCompleted(
	databaseID uuid.UUID,
) (*LogicalBackup, error) {
	var backup LogicalBackup

	err := storage.
		GetDb().
		Where("database_id = ? AND status = ?",
			databaseID, BackupStatusCompleted).
		Order("created_at DESC").
		First(&backup).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &backup, nil
}

func (r *BackupRepository) FindByDatabaseIDWithFiltersAndPagination(
	databaseID uuid.UUID,
	filters *BackupFilters,
	limit, offset int,
) ([]*LogicalBackup, error) {
	var backups []*LogicalBackup

	query := storage.
		GetDb().
		Where("database_id = ?", databaseID)

	if filters != nil {
		query = filters.applyToQuery(query)
	}

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&backups).Error; err != nil {
		return nil, err
	}

	return backups, nil
}

func (r *BackupRepository) CountByDatabaseIDWithFilters(
	databaseID uuid.UUID,
	filters *BackupFilters,
) (int64, error) {
	var count int64

	query := storage.
		GetDb().
		Model(&LogicalBackup{}).
		Where("database_id = ?", databaseID)

	if filters != nil {
		query = filters.applyToQuery(query)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (f *BackupFilters) applyToQuery(query *gorm.DB) *gorm.DB {
	if len(f.Statuses) > 0 {
		query = query.Where("status IN ?", f.Statuses)
	}

	if f.BeforeDate != nil {
		query = query.Where("created_at < ?", *f.BeforeDate)
	}

	return query
}
