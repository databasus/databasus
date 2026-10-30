package signin

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
	pendingSignIns *pendingSignInRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{pendingSignIns: &pendingSignInRepository{db: db}}
}

type pendingSignInRepository struct {
	db *gorm.DB
}

// lockUser holds the user's pending sign-ins for the rest of the transaction, so two password
// steps that arrive together cannot both find no live pending sign-in and both mail a code.
func (r *pendingSignInRepository) lockUser(ctx context.Context, userID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", "users_pending_sign_ins:"+userID.String()).Error
	if err != nil {
		return fmt.Errorf("lock the pending sign-ins of the user: %w", err)
	}

	return nil
}

type livePendingSignInFilter struct {
	userID          uuid.UUID
	passwordVersion uuid.UUID
	now             time.Time
}

func (r *pendingSignInRepository) getLive(
	ctx context.Context,
	filter livePendingSignInFilter,
) (PendingSignIn, bool, error) {
	var pendingSignIn PendingSignIn

	err := database.Conn(ctx, r.db).
		Where(
			"user_id = ? AND password_version = ? AND is_used = ? AND failed_attempt_count < ? AND expires_at > ?",
			filter.userID, filter.passwordVersion, false, maxCodeAttempts, filter.now,
		).
		Order("created_at DESC").
		First(&pendingSignIn).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PendingSignIn{}, false, nil
	}

	if err != nil {
		return PendingSignIn{}, false, fmt.Errorf("get the live pending sign-in: %w", err)
	}

	return pendingSignIn, true, nil
}

func (r *pendingSignInRepository) getByID(ctx context.Context, pendingSignInID uuid.UUID) (PendingSignIn, bool, error) {
	var pendingSignIn PendingSignIn

	err := database.Conn(ctx, r.db).Where("id = ?", pendingSignInID).First(&pendingSignIn).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PendingSignIn{}, false, nil
	}

	if err != nil {
		return PendingSignIn{}, false, fmt.Errorf("get the pending sign-in: %w", err)
	}

	return pendingSignIn, true, nil
}

func (r *pendingSignInRepository) countCreatedSince(
	ctx context.Context,
	userID uuid.UUID,
	since time.Time,
) (int64, error) {
	var pendingSignInCount int64

	err := database.Conn(ctx, r.db).Model(&PendingSignIn{}).
		Where("user_id = ? AND created_at > ?", userID, since).
		Count(&pendingSignInCount).Error
	if err != nil {
		return 0, fmt.Errorf("count the pending sign-ins of the user: %w", err)
	}

	return pendingSignInCount, nil
}

func (r *pendingSignInRepository) insert(ctx context.Context, pendingSignIn PendingSignIn) error {
	err := database.Conn(ctx, r.db).Create(&pendingSignIn).Error
	if err != nil {
		return fmt.Errorf("insert the pending sign-in: %w", err)
	}

	return nil
}

func (r *pendingSignInRepository) markUsed(ctx context.Context, pendingSignInID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE users_pending_sign_ins SET is_used = TRUE WHERE id = ?", pendingSignInID).Error
	if err != nil {
		return fmt.Errorf("mark the pending sign-in used: %w", err)
	}

	return nil
}

// spend marks the pending sign-in used only while it is still usable, so two requests with the
// same correct code cannot both sign in. It reports false when another request spent it first.
func (r *pendingSignInRepository) spend(ctx context.Context, pendingSignInID uuid.UUID, now time.Time) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(
		"UPDATE users_pending_sign_ins SET is_used = TRUE WHERE id = ? AND is_used = FALSE AND expires_at > ?",
		pendingSignInID, now,
	)
	if result.Error != nil {
		return false, fmt.Errorf("spend the pending sign-in: %w", result.Error)
	}

	return result.RowsAffected == 1, nil
}

// claimAttempt takes one of the pending sign-in's attempts before the code is compared, and
// reports false once none is left. Claiming first keeps a burst of simultaneous guesses at five,
// because the database hands the attempts out one at a time. It returns the attempts used so far.
func (r *pendingSignInRepository) claimAttempt(
	ctx context.Context,
	pendingSignInID uuid.UUID,
	now time.Time,
) (int, bool, error) {
	var claimedPendingSignIn PendingSignIn

	result := database.Conn(ctx, r.db).Model(&claimedPendingSignIn).
		Clauses(clause.Returning{Columns: []clause.Column{{Name: "failed_attempt_count"}}}).
		Where(
			"id = ? AND is_used = ? AND failed_attempt_count < ? AND expires_at > ?",
			pendingSignInID, false, maxCodeAttempts, now,
		).
		UpdateColumn("failed_attempt_count", gorm.Expr("failed_attempt_count + 1"))
	if result.Error != nil {
		return 0, false, fmt.Errorf("claim an attempt of the pending sign-in: %w", result.Error)
	}

	return claimedPendingSignIn.FailedAttemptCount, result.RowsAffected == 1, nil
}

// deleteCreatedBefore goes by age rather than by expiry, because the hourly cap counts pending
// sign-ins whose codes stopped working long ago.
func (r *pendingSignInRepository) deleteCreatedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_pending_sign_ins WHERE created_at < ?", cutoff)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the old pending sign-ins: %w", result.Error)
	}

	return result.RowsAffected, nil
}
