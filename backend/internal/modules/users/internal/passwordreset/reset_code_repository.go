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

// getNewestUsable reads the newest code that is unused and unexpired at now, locking it for the
// rest of the transaction, so one code cannot complete two resets.
func (r *resetCodeRepository) getNewestUsable(
	ctx context.Context,
	userID uuid.UUID,
	now time.Time,
) (ResetCode, bool, error) {
	var resetCode ResetCode

	err := database.Conn(ctx, r.db).
		Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate, Table: clause.Table{Name: ""}, Options: ""}).
		Where("user_id = ? AND is_used = FALSE AND expires_at > ?", userID, now).
		Order("created_at DESC").
		First(&resetCode).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ResetCode{}, false, nil
	}

	if err != nil {
		return ResetCode{}, false, fmt.Errorf("get the usable reset code: %w", err)
	}

	return resetCode, true, nil
}

func (r *resetCodeRepository) markUsed(ctx context.Context, resetCodeID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE users_password_reset_codes SET is_used = TRUE WHERE id = ?",
		resetCodeID).Error
	if err != nil {
		return fmt.Errorf("mark the reset code used: %w", err)
	}

	return nil
}

func (r *resetCodeRepository) deleteCreatedBefore(ctx context.Context, before time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_password_reset_codes WHERE created_at < ?", before)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the old reset codes: %w", result.Error)
	}

	return result.RowsAffected, nil
}
