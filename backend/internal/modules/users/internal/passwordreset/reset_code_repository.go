package passwordreset

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// service never holds it.
type Repositories struct {
	resetCodes *resetCodeRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{resetCodes: &resetCodeRepository{db: db}}
}

type resetCodeRepository struct {
	db *gorm.DB
}

func (r *resetCodeRepository) insert(ctx context.Context, resetCode ResetCode) error {
	err := database.Conn(ctx, r.db).Create(&resetCode).Error
	if err != nil {
		return fmt.Errorf("insert the reset code: %w", err)
	}

	return nil
}

func (r *resetCodeRepository) countCreatedSince(ctx context.Context, userID uuid.UUID, since time.Time) (int64, error) {
	var codeCount int64

	err := database.Conn(ctx, r.db).Model(&ResetCode{}).
		Where("user_id = ? AND created_at > ?", userID, since).
		Count(&codeCount).Error
	if err != nil {
		return 0, fmt.Errorf("count the reset codes of the user: %w", err)
	}

	return codeCount, nil
}

// getNewest reads the code issued to the user last, whatever its state, so an older code never
// takes its place.
func (r *resetCodeRepository) getNewest(ctx context.Context, userID uuid.UUID) (ResetCode, bool, error) {
	var resetCode ResetCode

	err := database.Conn(ctx, r.db).Where("user_id = ?", userID).Order("created_at DESC").First(&resetCode).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ResetCode{}, false, nil
	}

	if err != nil {
		return ResetCode{}, false, fmt.Errorf("get the newest reset code: %w", err)
	}

	return resetCode, true, nil
}

// claimAttempt takes one of the code's attempts before it is compared, and reports false once the
// code is used, expired or out of attempts. Claiming first keeps a burst of simultaneous guesses at
// five, because the database hands the attempts out one at a time, and it commits at once, so a
// refused request keeps the count. It returns the attempts used so far.
func (r *resetCodeRepository) claimAttempt(
	ctx context.Context,
	resetCodeID uuid.UUID,
	now time.Time,
) (int, bool, error) {
	var claimedResetCode ResetCode

	result := database.Conn(ctx, r.db).Model(&claimedResetCode).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "failed_attempt_count"}}}).
		Where(
			"id = ? AND is_used = ? AND failed_attempt_count < ? AND expires_at > ?",
			resetCodeID, false, maxCodeAttempts, now,
		).
		UpdateColumn("failed_attempt_count", gorm.Expr("failed_attempt_count + 1"))
	if result.Error != nil {
		return 0, false, fmt.Errorf("claim an attempt of the reset code: %w", result.Error)
	}

	return claimedResetCode.FailedAttemptCount, result.RowsAffected == 1, nil
}

// spend reports false when another request already used the code or it expired, so one code cannot
// complete two resets.
func (r *resetCodeRepository) spend(ctx context.Context, resetCodeID uuid.UUID, now time.Time) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE users_password_reset_codes SET is_used = TRUE WHERE id = ? AND is_used = FALSE AND expires_at > ?",
		resetCodeID, now,
	)
	if result.Error != nil {
		return false, fmt.Errorf("spend the reset code: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

func (r *resetCodeRepository) deleteCreatedBefore(ctx context.Context, before time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_password_reset_codes WHERE created_at < ?", before)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the old reset codes: %w", result.Error)
	}

	return result.RowsAffected, nil
}
