package crypto

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

// LegacyMasterKeyRepository reads the table where releases before the key file kept the master key.
type LegacyMasterKeyRepository struct {
	db *gorm.DB
}

func NewLegacyMasterKeyRepository(db *gorm.DB) *LegacyMasterKeyRepository {
	return &LegacyMasterKeyRepository{db: db}
}

// SelectNonEmptyKeyForUpdate locks a stored key until the transaction of ctx ends.
func (r *LegacyMasterKeyRepository) SelectNonEmptyKeyForUpdate(ctx context.Context) (string, bool, error) {
	var secrets []string

	err := database.Conn(ctx, r.db).
		Raw("SELECT secret FROM lib_crypto_secret_keys WHERE secret <> '' LIMIT 1 FOR UPDATE").
		Scan(&secrets).Error
	if err != nil {
		return "", false, fmt.Errorf("select the legacy master key: %w", err)
	}

	if len(secrets) == 0 {
		return "", false, nil
	}

	return secrets[0], true, nil
}

func (r *LegacyMasterKeyRepository) DeleteKey(ctx context.Context, secret string) error {
	err := database.Conn(ctx, r.db).Exec("DELETE FROM lib_crypto_secret_keys WHERE secret = ?", secret).Error
	if err != nil {
		return fmt.Errorf("delete the legacy master key: %w", err)
	}

	return nil
}
