package authenticator

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

type setupRepository struct {
	db *gorm.DB
}

func (r *setupRepository) getByID(ctx context.Context, setupID uuid.UUID) (AuthenticatorAppSetup, bool, error) {
	var setup AuthenticatorAppSetup

	err := database.Conn(ctx, r.db).Where("id = ?", setupID).First(&setup).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AuthenticatorAppSetup{}, false, nil
	}

	if err != nil {
		return AuthenticatorAppSetup{}, false, fmt.Errorf("get the authenticator setup: %w", err)
	}

	return setup, true, nil
}

func (r *setupRepository) insert(ctx context.Context, setup AuthenticatorAppSetup) error {
	err := database.Conn(ctx, r.db).Create(&setup).Error
	if err != nil {
		return fmt.Errorf("insert the authenticator setup: %w", err)
	}

	return nil
}

func (r *setupRepository) incrementFailedAttemptCount(ctx context.Context, setupID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE users_authenticator_setups SET failed_attempt_count = failed_attempt_count + 1 WHERE id = ?",
		setupID,
	).Error
	if err != nil {
		return fmt.Errorf("count a wrong code of the authenticator setup: %w", err)
	}

	return nil
}

// deleteByUserID removes the user's unconfirmed setup together with the pending sign-in that waits
// for it.
func (r *setupRepository) deleteByUserID(ctx context.Context, userID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM users_authenticator_setups WHERE user_id = ?", userID).Error
	if err != nil {
		return fmt.Errorf("delete the authenticator setup of the user: %w", err)
	}

	return nil
}

func (r *setupRepository) deleteByID(ctx context.Context, setupID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM users_authenticator_setups WHERE id = ?", setupID).Error
	if err != nil {
		return fmt.Errorf("delete the authenticator setup: %w", err)
	}

	return nil
}

func (r *setupRepository) deleteExpiredBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_authenticator_setups WHERE expires_at < ?", cutoff)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the expired authenticator setups: %w", result.Error)
	}

	return result.RowsAffected, nil
}
