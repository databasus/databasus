package users_models

import (
	"time"

	"github.com/google/uuid"

	users_enums "databasus-backend/internal/features/users/enums"
	"databasus-backend/internal/modules/users"
)

// MinPasswordLength applies to every way a password is set. The request DTOs
// repeat it as a literal min=8, because a struct tag cannot reference a constant.
const MinPasswordLength = 8

type User struct {
	ID                   uuid.UUID              `json:"id"`
	Email                string                 `json:"email"`
	Name                 string                 `json:"name"`
	HashedPassword       *string                `json:"-"         gorm:"column:hashed_password"`
	PasswordCreationTime time.Time              `json:"-"         gorm:"column:password_creation_time"`
	Role                 users.Role             `json:"role"`
	Status               users_enums.UserStatus `json:"status"`
	GitHubOAuthID        *string                `json:"-"         gorm:"column:github_oauth_id"`
	GoogleOAuthID        *string                `json:"-"         gorm:"column:google_oauth_id"`
	IsRootAdmin          bool                   `json:"-"         gorm:"column:is_root_admin"`
	CreatedAt            time.Time              `json:"createdAt"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) IsActiveUser() bool {
	return u.Status == users_enums.UserStatusActive
}

func (u *User) HasPassword() bool {
	return u.HashedPassword != nil && *u.HashedPassword != ""
}
