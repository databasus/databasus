package instancerecovery

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users/internal/settings"
)

const (
	usersTable    = "public.users"
	settingsTable = "public.users_settings"
)

type Repositories struct {
	state *stateRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{state: &stateRepository{db: db}}
}

type stateRepository struct {
	db *gorm.DB
}

func (r *stateRepository) hasAccounts(ctx context.Context) (bool, error) {
	isPresent, err := database.HasTable(ctx, r.db, usersTable)
	if err != nil || !isPresent {
		return false, err
	}

	var hasAccounts bool

	err = database.Conn(ctx, r.db).Raw("SELECT EXISTS (SELECT 1 FROM users)").Scan(&hasAccounts).Error
	if err != nil {
		return false, fmt.Errorf("check for accounts: %w", err)
	}

	return hasAccounts, nil
}

// listSettings reads every row of the instance settings; a new instance has none until the first
// read creates one.
func (r *stateRepository) listSettings(ctx context.Context) ([]settings.Settings, error) {
	isPresent, err := database.HasTable(ctx, r.db, settingsTable)
	if err != nil || !isPresent {
		return nil, err
	}

	var rows []settings.Settings

	err = database.Conn(ctx, r.db).Raw(`SELECT id, is_allow_external_registrations, is_allow_member_invitations,
			is_member_allowed_to_create_workspaces, is_two_factor_auth_required
		FROM users_settings`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("read the instance settings: %w", err)
	}

	return rows, nil
}

// endSessionsAndCodes rotates every password version, which ends every session and pending sign-in
// of the lost instance, and drops the codes it had issued.
func (r *stateRepository) endSessionsAndCodes(ctx context.Context) error {
	for _, statement := range []string{
		"UPDATE users SET password_version = gen_random_uuid()",
		"DELETE FROM users_pending_sign_ins",
		"DELETE FROM users_password_reset_codes",
		"DELETE FROM users_identity_codes",
	} {
		if err := database.Conn(ctx, r.db).Exec(statement).Error; err != nil {
			return fmt.Errorf("end the recovered sessions: %w", err)
		}
	}

	return nil
}
