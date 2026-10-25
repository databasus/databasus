package users_repositories

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	users_enums "databasus-backend/internal/features/users/enums"
	users_models "databasus-backend/internal/features/users/models"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
)

type UserRepository struct{}

func (r *UserRepository) GetUsersCount() (int64, error) {
	var count int64
	if err := storage.GetDb().Model(&users_models.User{}).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (r *UserRepository) CreateUser(user *users_models.User) error {
	return storage.GetDb().Create(user).Error
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// The record lives in a column rather than in the text of an email address, so
// the account keeps it after its owner moves to a real one. Returns nil when no
// account holds the record yet.
func (r *UserRepository) GetRootAdmin(ctx context.Context) (*users_models.User, error) {
	var user users_models.User

	if err := storage.GetDb().WithContext(ctx).Where("is_root_admin").First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateUserPassword(userID uuid.UUID, hashedPassword string) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"hashed_password":        hashedPassword,
			"password_creation_time": time.Now().UTC(),
		}).Error
}

func (r *UserRepository) GetUsers(
	ctx context.Context,
	limit, offset int,
	beforeCreatedAt *time.Time,
	query string,
) ([]*users_models.User, int64, error) {
	var users []*users_models.User
	var total int64

	countQuery := storage.GetDb().WithContext(ctx).Model(&users_models.User{})
	if beforeCreatedAt != nil {
		countQuery = countQuery.Where("created_at < ?", *beforeCreatedAt)
	}
	if query != "" {
		searchPattern := "%" + query + "%"
		countQuery = countQuery.Where("email ILIKE ? OR name ILIKE ?", searchPattern, searchPattern)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := storage.GetDb().WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC")

	if beforeCreatedAt != nil {
		dataQuery = dataQuery.Where("created_at < ?", *beforeCreatedAt)
	}
	if query != "" {
		searchPattern := "%" + query + "%"
		dataQuery = dataQuery.Where("email ILIKE ? OR name ILIKE ?", searchPattern, searchPattern)
	}

	if err := dataQuery.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *UserRepository) UpdateUserStatus(userID uuid.UUID, status users_enums.UserStatus) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"status": status,
		}).Error
}

func (r *UserRepository) UpdateUserRole(userID uuid.UUID, role users.Role) error {
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(map[string]any{
			"role": role,
		}).Error
}

func (r *UserRepository) GetAdmins(ctx context.Context) ([]*users_models.User, error) {
	var admins []*users_models.User

	if err := storage.GetDb().WithContext(ctx).
		Where("role = ?", users.RoleAdmin).
		Order("created_at ASC").
		Find(&admins).Error; err != nil {
		return nil, err
	}

	return admins, nil
}

func (r *UserRepository) UpdateUserInfo(userID uuid.UUID, name, email *string) error {
	updates := make(map[string]any)

	if name != nil {
		updates["name"] = *name
	}
	if email != nil {
		updates["email"] = *email
	}

	if len(updates) == 0 {
		return nil
	}

	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

func (r *UserRepository) GetUserByGitHubOAuthID(githubID string) (*users_models.User, error) {
	var user users_models.User
	err := storage.GetDb().Where("github_oauth_id = ?", githubID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetUserByGoogleOAuthID(googleID string) (*users_models.User, error) {
	var user users_models.User
	err := storage.GetDb().Where("google_oauth_id = ?", googleID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) LinkOAuthID(userID uuid.UUID, oauthColumn, oauthID string) error {
	updates := map[string]any{oauthColumn: oauthID}
	return storage.GetDb().Model(&users_models.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}
