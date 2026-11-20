package tokens

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	tokens *tokenRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{tokens: &tokenRepository{db: db}}
}

type tokenRepository struct {
	db *gorm.DB
}

func (r *tokenRepository) insert(ctx context.Context, token Token) error {
	err := database.Conn(ctx, r.db).Create(&token).Error
	if err != nil {
		return fmt.Errorf("insert the download token: %w", err)
	}

	return nil
}

// consume marks the token used and returns it, unless it is unknown, used, expired or of another
// purpose, which it reports as false. One UPDATE decides, so two requests cannot both win.
func (r *tokenRepository) consume(
	ctx context.Context,
	tokenSha256 string,
	purpose Purpose,
	now time.Time,
) (Token, bool, error) {
	var consumedTokens []Token

	err := database.Conn(ctx, r.db).Raw(`UPDATE instancebackups_tokens SET consumed_at = ?
		WHERE token_sha256 = ? AND purpose = ? AND consumed_at IS NULL AND expires_at > ?
		RETURNING *`,
		now, tokenSha256, purpose, now,
	).Scan(&consumedTokens).Error
	if err != nil {
		return Token{}, false, fmt.Errorf("consume the download token: %w", err)
	}

	if len(consumedTokens) == 0 {
		return Token{}, false, nil
	}

	return consumedTokens[0], true, nil
}

func (r *tokenRepository) deleteExpiredBefore(ctx context.Context, threshold time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM instancebackups_tokens WHERE expires_at < ?", threshold)
	if result.Error != nil {
		return 0, fmt.Errorf("delete expired download tokens: %w", result.Error)
	}

	return result.RowsAffected, nil
}

// deleteAll revokes every token, as recovery does with credentials copied from the source.
func (r *tokenRepository) deleteAll(ctx context.Context) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM instancebackups_tokens").Error
	if err != nil {
		return fmt.Errorf("delete every download token: %w", err)
	}

	return nil
}
