package settings

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
	settings *settingsRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{settings: &settingsRepository{db: db}}
}

type settingsRepository struct {
	db *gorm.DB
}

// getOrCreate reads the settings row and creates it with the defaults on an instance that has none
// yet.
func (r *settingsRepository) getOrCreate(ctx context.Context) (Settings, error) {
	var settings Settings

	err := database.Conn(ctx, r.db).Order("id").First(&settings).Error
	if err == nil {
		return settings, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Settings{}, fmt.Errorf("read the instance settings: %w", err)
	}

	settings = Settings{
		ID:                                uuid.New(),
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
		IsTwoFactorAuthRequired:           false,
	}

	err = database.Conn(ctx, r.db).Create(&settings).Error
	if err != nil {
		return Settings{}, fmt.Errorf("create the instance settings: %w", err)
	}

	return settings, nil
}

func (r *settingsRepository) updateSwitches(ctx context.Context, settingsID uuid.UUID, switches Switches) error {
	err := database.Conn(ctx, r.db).Exec(
		`UPDATE users_settings
		SET is_allow_external_registrations = ?,
			is_allow_member_invitations = ?,
			is_member_allowed_to_create_workspaces = ?,
			is_two_factor_auth_required = ?
		WHERE id = ?`,
		switches.IsAllowExternalRegistrations,
		switches.IsAllowMemberInvitations,
		switches.IsMemberAllowedToCreateWorkspaces,
		switches.IsTwoFactorAuthRequired,
		settingsID,
	).Error
	if err != nil {
		return fmt.Errorf("update the instance settings: %w", err)
	}

	return nil
}

func (r *settingsRepository) updateIsTwoFactorAuthRequired(
	ctx context.Context,
	settingsID uuid.UUID,
	isRequired bool,
) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE users_settings SET is_two_factor_auth_required = ? WHERE id = ?", isRequired, settingsID,
	).Error
	if err != nil {
		return fmt.Errorf("update the second factor setting: %w", err)
	}

	return nil
}
