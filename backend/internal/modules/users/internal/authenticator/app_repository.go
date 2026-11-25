package authenticator

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// service never holds it.
type Repositories struct {
	apps          *appRepository
	setups        *setupRepository
	recoveryCodes *recoveryCodeRepository
	failures      *failureRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{
		apps:          &appRepository{db: db},
		setups:        &setupRepository{db: db},
		recoveryCodes: &recoveryCodeRepository{db: db},
		failures:      &failureRepository{db: db},
	}
}

type appRepository struct {
	db *gorm.DB
}

// lockUser holds every second-factor check and change of the user for the rest of the transaction,
// so two requests with one code cannot both succeed and no more wrong codes are checked than the
// limit allows. Callers that also lock the user's pending sign-ins take that lock first.
func (r *appRepository) lockUser(ctx context.Context, userID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "users_second_factor:"+userID.String()).Error
	if err != nil {
		return fmt.Errorf("lock the second factor of the user: %w", err)
	}

	return nil
}

func (r *appRepository) getByUserID(ctx context.Context, userID uuid.UUID) (AuthenticatorApp, bool, error) {
	var app AuthenticatorApp

	err := database.Conn(ctx, r.db).Where("user_id = ?", userID).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AuthenticatorApp{}, false, nil
	}

	if err != nil {
		return AuthenticatorApp{}, false, fmt.Errorf("get the authenticator app of the user: %w", err)
	}

	return app, true, nil
}

func (r *appRepository) listByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]AuthenticatorApp, error) {
	var apps []AuthenticatorApp

	err := database.Conn(ctx, r.db).Where("user_id IN ?", userIDs).Find(&apps).Error
	if err != nil {
		return nil, fmt.Errorf("list the authenticator apps of the users: %w", err)
	}

	return apps, nil
}

func (r *appRepository) insert(ctx context.Context, app AuthenticatorApp) error {
	err := database.Conn(ctx, r.db).Create(&app).Error
	if err != nil {
		return fmt.Errorf("insert the authenticator app: %w", err)
	}

	return nil
}

// advanceLastUsedStep records the period of an accepted code only when it is later than the last
// one, and reports false when a code of that period or a later one was accepted first.
func (r *appRepository) advanceLastUsedStep(ctx context.Context, appID uuid.UUID, step int64) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE users_authenticator_apps SET last_used_step = ? WHERE id = ? AND last_used_step < ?",
		step, appID, step,
	)
	if result.Error != nil {
		return false, fmt.Errorf("record the used step of the authenticator app: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

// deleteByUserID removes the app; its recovery codes and the pending sign-ins that wait for its
// code go with it through the foreign keys. It reports false when the user had no app.
func (r *appRepository) deleteByUserID(ctx context.Context, userID uuid.UUID) (bool, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_authenticator_apps WHERE user_id = ?", userID)
	if result.Error != nil {
		return false, fmt.Errorf("delete the authenticator app of the user: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}
