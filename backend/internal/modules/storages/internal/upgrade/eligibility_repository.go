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

	var blockers struct {
		HasGoogleDrive bool
		HasRclone      bool
	}
	err = database.Conn(ctx, db).Raw(
		`SELECT EXISTS (SELECT 1 FROM storages WHERE type = 'GOOGLE_DRIVE') AS has_google_drive,
 EXISTS (SELECT 1 FROM storages WHERE type = 'RCLONE') AS has_rclone`,
	).Scan(&blockers).Error
	if err != nil {
		return fmt.Errorf("inspect storages before upgrading: %w", err)
	}

	if blockers.HasGoogleDrive && blockers.HasRclone {
		return ErrGoogleDriveAndRcloneRetired
	}

	if blockers.HasRclone {
		return ErrRcloneRetired
	}

	if blockers.HasGoogleDrive {
		return ErrGoogleDriveRetired
	}

	return nil
}
