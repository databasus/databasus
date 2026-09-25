package healthcheck_attempt

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/storage"
)

type HealthcheckAttemptRepository struct{}

func (r *HealthcheckAttemptRepository) FindByDatabaseIdOrderByCreatedAtDesc(
	databaseID uuid.UUID,
	afterDate time.Time,
) ([]*HealthcheckAttempt, error) {
	var attempts []*HealthcheckAttempt

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Where("created_at > ?", afterDate).
		Order("created_at DESC").
		Find(&attempts).Error; err != nil {
		return nil, err
	}

	return attempts, nil
}

func (r *HealthcheckAttemptRepository) FindLastByDatabaseID(
	databaseID uuid.UUID,
) (*HealthcheckAttempt, error) {
	var attempt HealthcheckAttempt

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		First(&attempt).Error; err != nil {
		return nil, err
	}

	return &attempt, nil
}

func (r *HealthcheckAttemptRepository) DeleteOlderThan(
	databaseID uuid.UUID,
	olderThan time.Time,
) error {
	return storage.
		GetDb().
		Where("database_id = ? AND created_at < ?", databaseID, olderThan).
		Delete(&HealthcheckAttempt{}).Error
}

func (r *HealthcheckAttemptRepository) Create(
	attempt *HealthcheckAttempt,
) error {
	if attempt.ID == uuid.Nil {
		attempt.ID = uuid.New()
	}

	if attempt.CreatedAt.IsZero() {
		attempt.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(attempt).Error
}

func (r *HealthcheckAttemptRepository) Insert(
	attempt *HealthcheckAttempt,
) error {
	return r.Create(attempt)
}

func (r *HealthcheckAttemptRepository) FindByDatabaseIDWithLimit(
	databaseID uuid.UUID,
	limit int,
) ([]*HealthcheckAttempt, error) {
	var attempts []*HealthcheckAttempt

	if err := storage.
		GetDb().
		Where("database_id = ?", databaseID).
		Order("created_at DESC").
		Limit(limit).
		Find(&attempts).Error; err != nil {
		return nil, err
	}

	return attempts, nil
}

func (r *HealthcheckAttemptRepository) CountByDatabaseID(
	databaseID uuid.UUID,
) (int64, error) {
	var count int64

	if err := storage.
		GetDb().
		Model(&HealthcheckAttempt{}).
		Where("database_id = ?", databaseID).
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *HealthcheckAttemptRepository) FindRecentByDatabaseIDs(
	databaseIDs []uuid.UUID,
	limitPerDatabase int,
) ([]*HealthcheckAttempt, error) {
	attempts := []*HealthcheckAttempt{}

	if len(databaseIDs) == 0 {
		return attempts, nil
	}

	if err := storage.GetDb().Raw(`
		SELECT id, database_id, status, created_at
		FROM (
			SELECT id, database_id, status, created_at,
			       ROW_NUMBER() OVER (PARTITION BY database_id ORDER BY created_at DESC) AS recency_rank
			FROM healthcheck_attempts
			WHERE database_id IN ?
		) ranked
		WHERE recency_rank <= ?
		ORDER BY database_id, created_at DESC
	`,
		databaseIDs,
		limitPerDatabase,
	).Scan(&attempts).Error; err != nil {
		return nil, err
	}

	return attempts, nil
}
