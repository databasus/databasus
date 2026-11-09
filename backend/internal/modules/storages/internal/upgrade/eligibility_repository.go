package upgrade

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
)

func CheckEligibility(ctx context.Context, db *gorm.DB) error {
	var hasStoragesTable bool
	err := database.Conn(ctx, db).Raw("SELECT to_regclass('storages') IS NOT NULL").Scan(&hasStoragesTable).Error
	if err != nil {
		return fmt.Errorf("inspect storage schema before upgrading: %w", err)
	}

	if !hasStoragesTable {
		return nil
	}

	var hasGoogleDrive bool
	err = database.Conn(ctx, db).Raw(
		"SELECT EXISTS (SELECT 1 FROM storages WHERE type = 'GOOGLE_DRIVE')",
	).Scan(&hasGoogleDrive).Error
	if err != nil {
		return fmt.Errorf("inspect storages before upgrading: %w", err)
	}

	if hasGoogleDrive {
		return ErrGoogleDriveRetired
	}

	return nil
}
