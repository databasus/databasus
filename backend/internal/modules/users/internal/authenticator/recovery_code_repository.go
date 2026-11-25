package authenticator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type recoveryCodeRepository struct {
	db *gorm.DB
}

type unusedRecoveryCodeCount struct {
	AuthenticatorAppID uuid.UUID `gorm:"column:authenticator_app_id"`
	UnusedCount        int       `gorm:"column:unused_count"`
}

func (r *recoveryCodeRepository) countUnusedByAppIDs(
	ctx context.Context,
	appIDs []uuid.UUID,
) (map[uuid.UUID]int, error) {
	var counts []unusedRecoveryCodeCount

	err := database.Conn(ctx, r.db).Raw(
		`SELECT authenticator_app_id, count(*) AS unused_count
		FROM users_recovery_codes
		WHERE authenticator_app_id IN ? AND used_at IS NULL
		GROUP BY authenticator_app_id`,
		appIDs,
	).Scan(&counts).Error
	if err != nil {
		return nil, fmt.Errorf("count the unused recovery codes: %w", err)
	}

	countsByAppID := make(map[uuid.UUID]int, len(counts))
	for _, count := range counts {
		countsByAppID[count.AuthenticatorAppID] = count.UnusedCount
	}

	return countsByAppID, nil
}

func (r *recoveryCodeRepository) insertAll(ctx context.Context, recoveryCodes []RecoveryCode) error {
	err := database.Conn(ctx, r.db).Create(&recoveryCodes).Error
	if err != nil {
		return fmt.Errorf("insert the recovery codes: %w", err)
	}

	return nil
}

func (r *recoveryCodeRepository) spend(
	ctx context.Context,
	appID uuid.UUID,
	hashedCode string,
	now time.Time,
) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		`UPDATE users_recovery_codes SET used_at = ?
		WHERE authenticator_app_id = ? AND hashed_code = ? AND used_at IS NULL`,
		now, appID, hashedCode,
	)
	if result.Error != nil {
		return false, fmt.Errorf("spend the recovery code: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *recoveryCodeRepository) deleteByAppID(ctx context.Context, appID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM users_recovery_codes WHERE authenticator_app_id = ?", appID).Error
	if err != nil {
		return fmt.Errorf("delete the recovery codes: %w", err)
	}

	return nil
}
