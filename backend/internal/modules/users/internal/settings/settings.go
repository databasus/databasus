// Package settings owns the instance settings: who may register, invite and create workspaces,
// and which second factor password sign-in requires. It also sends the test email.
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
	IsMemberAllowedToCreateWorkspaces bool                 `gorm:"column:is_member_allowed_to_create_workspaces"`
	TwoFactorRequirement              TwoFactorRequirement `gorm:"column:two_factor_requirement"`
}

// TwoFactorRequirement is the second factor the instance requires at password sign-in. An account
// with a connected authenticator app presents the app under every value.
type TwoFactorRequirement string

const (
	TwoFactorRequirementOff              TwoFactorRequirement = "OFF"
	TwoFactorRequirementEmail            TwoFactorRequirement = "EMAIL"
	TwoFactorRequirementAuthenticatorApp TwoFactorRequirement = "AUTHENTICATOR_APP"
)

func (r TwoFactorRequirement) IsValid() bool {
	switch r {
	case TwoFactorRequirementOff, TwoFactorRequirementEmail, TwoFactorRequirementAuthenticatorApp:
		return true
	default:
		return false
	}
}

func (r TwoFactorRequirement) IsRequired() bool {
	return r != TwoFactorRequirementOff
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
	TwoFactorRequirement              TwoFactorRequirement
	IsEmailConfigured                 bool
}

// Switches are the editable settings an administrator replaces.
type Switches struct {
	IsAllowExternalRegistrations      bool
	IsAllowMemberInvitations          bool
	IsMemberAllowedToCreateWorkspaces bool
	TwoFactorRequirement              TwoFactorRequirement
}

// DefaultSwitches are the settings a new instance starts with.
func DefaultSwitches() Switches {
	return Switches{
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
		TwoFactorRequirement:              TwoFactorRequirementOff,
	}
}

func (s Settings) GetSwitches() Switches {
	return Switches{
		IsAllowExternalRegistrations:      s.IsAllowExternalRegistrations,
		IsAllowMemberInvitations:          s.IsAllowMemberInvitations,
		IsMemberAllowedToCreateWorkspaces: s.IsMemberAllowedToCreateWorkspaces,
		TwoFactorRequirement:              s.TwoFactorRequirement,
	}
}
