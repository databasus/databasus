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
// read creates one. A target of a release before the requirement method still holds the boolean,
// which reads as the emailed code method, as the migration maps it.
func (r *stateRepository) listSettings(ctx context.Context) ([]settings.Settings, error) {
	isPresent, err := database.HasTable(ctx, r.db, settingsTable)
	if err != nil || !isPresent {
		return nil, err
	}

	hasRequirementColumn, err := r.hasSettingsColumn(ctx, "two_factor_requirement")
	if err != nil {
		return nil, err
	}

	requirementExpression := "two_factor_requirement"
	if !hasRequirementColumn {
		requirementExpression = "CASE WHEN is_two_factor_auth_required THEN 'EMAIL' ELSE 'OFF' END"
	}

	var rows []settings.Settings

	// nosemgrep: go.lang.security.audit.database.string-formatted-query.string-formatted-query
	err = database.Conn(ctx, r.db).Raw(`SELECT id, is_allow_external_registrations, is_allow_member_invitations,
			is_member_allowed_to_create_workspaces, ` + requirementExpression + ` AS two_factor_requirement
		FROM users_settings`).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("read the instance settings: %w", err)
	}

	return rows, nil
}

func (r *stateRepository) hasSettingsColumn(ctx context.Context, column string) (bool, error) {
	var isPresent bool

	err := database.Conn(ctx, r.db).Raw(`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'users_settings' AND column_name = ?
		)`, column).Scan(&isPresent).Error
	if err != nil {
		return false, fmt.Errorf("check for settings column %s: %w", column, err)
	}

	return isPresent, nil
}

// endSessionsAndCodes rotates every password version, which ends every session and pending sign-in
// of the lost instance, and drops the codes, unconfirmed app setups and wrong-code counts it had.
// Connected authenticator apps and their recovery codes belong to the accounts and stay.
func (r *stateRepository) endSessionsAndCodes(ctx context.Context) error {
	for _, statement := range []string{
		"UPDATE users SET password_version = gen_random_uuid()",
		"DELETE FROM users_pending_sign_ins",
		"DELETE FROM users_password_reset_codes",
		"DELETE FROM users_identity_codes",
		"DELETE FROM users_authenticator_setups",
		"DELETE FROM users_second_factor_failures",
	} {
		if err := database.Conn(ctx, r.db).Exec(statement).Error; err != nil {
			return fmt.Errorf("end the recovered sessions: %w", err)
		}
	}

	return nil
}
