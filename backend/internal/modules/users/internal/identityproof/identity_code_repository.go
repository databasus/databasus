package identityproof

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	codes *identityCodeRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{codes: &identityCodeRepository{db: db}}
}

type identityCodeRepository struct {
	db *gorm.DB
}

func (r *identityCodeRepository) insert(ctx context.Context, code IdentityCode) error {
	err := database.Conn(ctx, r.db).Create(&code).Error
	if err != nil {
		return fmt.Errorf("insert the identity code: %w", err)
	}

	return nil
}

// getNewest returns the newest code of the user for the binding and session, used or not, so a
// replaced code never wins over the one that replaced it.
func (r *identityCodeRepository) getNewestForUpdate(
	ctx context.Context,
	userID uuid.UUID,
	sessionSha256 string,
	binding Binding,
) (IdentityCode, bool, error) {
	var codes []IdentityCode

	err := database.Conn(ctx, r.db).Raw(`SELECT * FROM users_identity_codes
		WHERE user_id = ? AND session_sha256 = ? AND purpose = ? AND subject = ?
		ORDER BY created_at DESC, id DESC LIMIT 1 FOR UPDATE`,
		userID, sessionSha256, binding.Purpose, binding.Subject,
	).Scan(&codes).Error
	if err != nil {
		return IdentityCode{}, false, fmt.Errorf("get the identity code: %w", err)
	}

	if len(codes) == 0 {
		return IdentityCode{}, false, nil
	}

	return codes[0], true, nil
}

func (r *identityCodeRepository) getNewestCreatedAt(ctx context.Context, userID uuid.UUID) (*time.Time, error) {
	var createdAt []time.Time

	err := database.Conn(ctx, r.db).
		Raw("SELECT created_at FROM users_identity_codes WHERE user_id = ? ORDER BY created_at DESC LIMIT 1", userID).
		Scan(&createdAt).Error
	if err != nil {
		return nil, fmt.Errorf("get the newest identity code time: %w", err)
	}

	if len(createdAt) == 0 {
		return nil, nil
	}

	return &createdAt[0], nil
}

func (r *identityCodeRepository) markUsed(ctx context.Context, codeID uuid.UUID) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE users_identity_codes SET is_used = TRUE WHERE id = ?", codeID).Error
	if err != nil {
		return fmt.Errorf("use the identity code: %w", err)
	}

	return nil
}

func (r *identityCodeRepository) incrementFailedAttemptCount(ctx context.Context, codeID uuid.UUID) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE users_identity_codes SET failed_attempt_count = failed_attempt_count + 1 WHERE id = ?", codeID).
		Error
	if err != nil {
		return fmt.Errorf("count a wrong identity code: %w", err)
	}

	return nil
}

func (r *identityCodeRepository) deleteExpiredBefore(ctx context.Context, threshold time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM users_identity_codes WHERE expires_at < ?", threshold)
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired identity codes: %w", result.Error)
	}

	return result.RowsAffected, nil
}
