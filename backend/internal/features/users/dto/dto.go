package users_dto

import (
	"time"

	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
)

type SignUpRequestDTO struct {
	Email                    string  `json:"email"                    binding:"required,email"`
	Password                 string  `json:"password"                 binding:"required,min=8"`
	Name                     string  `json:"name"                     binding:"required"`
	CloudflareTurnstileToken *string `json:"cloudflareTurnstileToken"`
}

type SignInRequestDTO struct {
	Email                    string  `json:"email"                    binding:"required"`
	Password                 string  `json:"password"                 binding:"required"`
	CloudflareTurnstileToken *string `json:"cloudflareTurnstileToken"`
}

type SignInResponseDTO struct {
	UserID uuid.UUID `json:"userId"`
	Email  string    `json:"email"`
	Token  string    `json:"token"`
}

// SignInOutcomeResponseDTO exists for the published API description, which
// cannot express two alternative bodies for one status. The token fields arrive
// when the instance asks for one factor, and pendingSignInId arrives when it
// asks for an emailed code. Nothing serializes this type.
type SignInOutcomeResponseDTO struct {
	UserID          uuid.UUID `json:"userId"`
	Email           string    `json:"email"`
	Token           string    `json:"token,omitzero"`
	PendingSignInID uuid.UUID `json:"pendingSignInId,omitzero"`
}

// A sign-in answers with one of these shapes, never both: a token when the
// instance asks for one factor, and a pending sign-in when it asks for two.
type SignInOutcome struct {
	CompletedSignIn *SignInResponseDTO
	PendingSignIn   *PendingSignInResponseDTO
}

// The address is carried back so the code screen can say where the message went,
// and the identifier is the only way to name the pending sign-in afterwards.
type PendingSignInResponseDTO struct {
	PendingSignInID uuid.UUID `json:"pendingSignInId"`
	Email           string    `json:"email"`
}

type VerifySignInCodeRequestDTO struct {
	PendingSignInID          uuid.UUID `json:"pendingSignInId"          binding:"required"`
	Code                     string    `json:"code"                     binding:"required"`
	CloudflareTurnstileToken *string   `json:"cloudflareTurnstileToken"`
}

type ResendSignInCodeRequestDTO struct {
	PendingSignInID          uuid.UUID `json:"pendingSignInId"          binding:"required"`
	CloudflareTurnstileToken *string   `json:"cloudflareTurnstileToken"`
}

// The wire name is isExist rather than hasAnyUser because the client already
// reads that key (frontend/src/entity/users/api/userApi.ts).
type HasAnyUserResponseDTO struct {
	HasAnyUser      bool `json:"isExist"`
	IsSignUpAllowed bool `json:"isSignUpAllowed"`
}

type ChangePasswordRequestDTO struct {
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

type UpdateUserInfoRequestDTO struct {
	Name  *string `json:"name"`
	Email *string `json:"email" binding:"omitempty,email"`
}

type InviteUserRequestDTO struct {
	Email                 string                     `json:"email"                 binding:"required,email"`
	IntendedWorkspaceID   *uuid.UUID                 `json:"intendedWorkspaceId"`
	IntendedWorkspaceRole *users_enums.WorkspaceRole `json:"intendedWorkspaceRole"`
}

type InviteUserResponseDTO struct {
	ID                    uuid.UUID                  `json:"id"`
	Email                 string                     `json:"email"`
	IntendedWorkspaceID   *uuid.UUID                 `json:"intendedWorkspaceId"`
	IntendedWorkspaceRole *users_enums.WorkspaceRole `json:"intendedWorkspaceRole"`
	CreatedAt             time.Time                  `json:"createdAt"`
}

type UserProfileResponseDTO struct {
	ID        uuid.UUID            `json:"id"`
	Email     string               `json:"email"`
	Name      string               `json:"name"`
	Role      users_enums.UserRole `json:"role"`
	IsActive  bool                 `json:"isActive"`
	CreatedAt time.Time            `json:"createdAt"`
}

type ListUsersResponseDTO struct {
	Users []UserProfileResponseDTO `json:"users"`
	Total int64                    `json:"total"`
}

type ChangeUserRoleRequestDTO struct {
	Role users_enums.UserRole `json:"role" binding:"required"`
}

type ListUsersRequestDTO struct {
	Limit      int        `form:"limit"      json:"limit"`
	Offset     int        `form:"offset"     json:"offset"`
	BeforeDate *time.Time `form:"beforeDate" json:"beforeDate"`
	Query      string     `form:"query"      json:"query"`
}

type OAuthCallbackRequestDTO struct {
	Code        string `json:"code"        binding:"required"`
	RedirectUri string `json:"redirectUri" binding:"required"`
}

type OAuthCallbackResponseDTO struct {
	UserID    uuid.UUID `json:"userId"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	IsNewUser bool      `json:"isNewUser"`
}

type SendResetPasswordCodeRequestDTO struct {
	Email                    string  `json:"email"                    binding:"required,email"`
	CloudflareTurnstileToken *string `json:"cloudflareTurnstileToken"`
}

type ResetPasswordRequestDTO struct {
	Email       string `json:"email"       binding:"required,email"`
	Code        string `json:"code"        binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// The mail-server answer is the instance's own, not the build-time flag the
// interface used to predict it from, and it is derived rather than stored, so
// the update request cannot carry it back.
type SettingsResponseDTO struct {
	IsAllowExternalRegistrations      bool `json:"isAllowExternalRegistrations"`
	IsAllowMemberInvitations          bool `json:"isAllowMemberInvitations"`
	IsMemberAllowedToCreateWorkspaces bool `json:"isMemberAllowedToCreateWorkspaces"`
	IsTwoFactorAuthRequired           bool `json:"isTwoFactorAuthRequired"`
	IsEmailConfigured                 bool `json:"isEmailConfigured"`
}

type SendTestEmailResponseDTO struct {
	RecipientEmail string `json:"recipientEmail"`
}

type UpdateSettingsRequestDTO struct {
	IsAllowExternalRegistrations      bool `json:"isAllowExternalRegistrations"`
	IsAllowMemberInvitations          bool `json:"isAllowMemberInvitations"`
	IsMemberAllowedToCreateWorkspaces bool `json:"isMemberAllowedToCreateWorkspaces"`
	IsTwoFactorAuthRequired           bool `json:"isTwoFactorAuthRequired"`
}
