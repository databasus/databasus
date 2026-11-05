package download

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

const tokenColumns = "id, token, backup_id, user_id, expires_at, is_used, created_at"

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

func (r *tokenRepository) insert(ctx context.Context, downloadToken DownloadToken) error {
	err := database.Conn(ctx, r.db).Exec(`INSERT INTO backups_download_tokens
		(id, token, backup_id, user_id, expires_at, is_used, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		downloadToken.ID, downloadToken.Token, downloadToken.BackupID, downloadToken.UserID,
		downloadToken.ExpiresAt, downloadToken.IsUsed, downloadToken.CreatedAt,
	).Error
	if err != nil {
		return fmt.Errorf("insert the download token: %w", err)
	}

	return nil
}

// getUsable finds the unused, unexpired token issued for the backup.
func (r *tokenRepository) getUsable(
	ctx context.Context,
	token string,
	backupID uuid.UUID,
	now time.Time,
) (DownloadToken, bool, error) {
	var foundTokens []DownloadToken

	err := database.Conn(ctx, r.db).Raw("SELECT "+tokenColumns+` FROM backups_download_tokens
		WHERE token = ? AND backup_id = ? AND NOT is_used AND expires_at > ?`,
		token, backupID, now).
		Scan(&foundTokens).Error
	if err != nil {
		return DownloadToken{}, false, fmt.Errorf("get the download token: %w", err)
	}

	if len(foundTokens) == 0 {
		return DownloadToken{}, false, nil
	}

	return foundTokens[0], true, nil
}

// consume marks the token used unless another request did first, which it reports as false.
func (r *tokenRepository) consume(ctx context.Context, downloadToken DownloadToken, now time.Time) (bool, error) {
	result := database.Conn(ctx, r.db).Exec(`UPDATE backups_download_tokens SET is_used = TRUE
		WHERE token = ? AND backup_id = ? AND NOT is_used AND expires_at > ?`,
		downloadToken.Token, downloadToken.BackupID, now)
	if result.Error != nil {
		return false, fmt.Errorf("consume the download token: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r *tokenRepository) deleteExpired(ctx context.Context, now time.Time) (int64, error) {
	result := database.Conn(ctx, r.db).Exec("DELETE FROM backups_download_tokens WHERE expires_at <= ?", now)
	if result.Error != nil {
		return 0, fmt.Errorf("delete the expired download tokens: %w", result.Error)
	}

	return result.RowsAffected, nil
}
