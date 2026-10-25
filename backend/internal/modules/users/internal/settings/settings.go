// Package settings owns the instance settings: who may register, invite and create workspaces,
// and whether password sign-in asks for an emailed code. It also sends the test email.
package settings

import "github.com/google/uuid"

// Settings is the one row of instance settings.
type Settings struct {
	ID uuid.UUID `gorm:"column:id"`
	// IsAllowExternalRegistrations lets anyone sign up without an invitation.
	IsAllowExternalRegistrations bool `gorm:"column:is_allow_external_registrations"`
	// IsAllowMemberInvitations lets a user who is not an instance admin invite users.
	IsAllowMemberInvitations bool `gorm:"column:is_allow_member_invitations"`
	// IsMemberAllowedToCreateWorkspaces lets a user who is not an instance admin create workspaces.
	IsMemberAllowedToCreateWorkspaces bool `gorm:"column:is_member_allowed_to_create_workspaces"`
	// IsTwoFactorAuthRequired makes a password sign-in also present a code emailed to the user.
	IsTwoFactorAuthRequired bool `gorm:"column:is_two_factor_auth_required"`
}

func (Settings) TableName() string {
	return "users_settings"
}

// InstanceSettings is what the settings operations answer. IsEmailConfigured is the instance's
// own judgement about its mail server, so the screen and a refusal never disagree.
type InstanceSettings struct {
	IsAllowExternalRegistrations      bool
	IsAllowMemberInvitations          bool
	IsMemberAllowedToCreateWorkspaces bool
	IsTwoFactorAuthRequired           bool
	IsEmailConfigured                 bool
}

// Switches are the editable settings an administrator replaces.
type Switches struct {
	IsAllowExternalRegistrations      bool
	IsAllowMemberInvitations          bool
	IsMemberAllowedToCreateWorkspaces bool
	IsTwoFactorAuthRequired           bool
}
