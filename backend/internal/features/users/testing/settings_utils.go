package users_testing

import (
	"context"
	"fmt"
)

func EnableMemberInvitations(ctx context.Context) {
	setInstanceSetting(ctx, "is_allow_member_invitations", true)
}

func DisableMemberInvitations(ctx context.Context) {
	setInstanceSetting(ctx, "is_allow_member_invitations", false)
}

func EnableExternalRegistrations(ctx context.Context) {
	setInstanceSetting(ctx, "is_allow_external_registrations", true)
}

func DisableExternalRegistrations(ctx context.Context) {
	setInstanceSetting(ctx, "is_allow_external_registrations", false)
}

func EnableMemberWorkspaceCreation(ctx context.Context) {
	setInstanceSetting(ctx, "is_member_allowed_to_create_workspaces", true)
}

func DisableMemberWorkspaceCreation(ctx context.Context) {
	setInstanceSetting(ctx, "is_member_allowed_to_create_workspaces", false)
}

func EnableTwoFactorAuth(ctx context.Context) {
	setInstanceSetting(ctx, "is_two_factor_auth_required", true)
}

func DisableTwoFactorAuth(ctx context.Context) {
	setInstanceSetting(ctx, "is_two_factor_auth_required", false)
}

func ResetSettingsToDefaults(ctx context.Context) {
	ensureSettingsRow(ctx)
	executeSQL(ctx, `UPDATE users_settings SET
		is_allow_external_registrations = TRUE,
		is_allow_member_invitations = TRUE,
		is_member_allowed_to_create_workspaces = TRUE,
		is_two_factor_auth_required = FALSE`)
}

// setInstanceSetting takes the column from the functions above, never from a caller.
func setInstanceSetting(ctx context.Context, column string, value bool) {
	ensureSettingsRow(ctx)
	executeSQL(ctx, fmt.Sprintf("UPDATE users_settings SET %s = ?", column), value)
}

// ensureSettingsRow creates the row the users module creates on its first read, with the defaults
// of the table.
func ensureSettingsRow(ctx context.Context) {
	executeSQL(
		ctx,
		"INSERT INTO users_settings (id) SELECT gen_random_uuid() WHERE NOT EXISTS (SELECT 1 FROM users_settings)",
	)
}
