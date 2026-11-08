package accounts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/users/internal/identity"
	"databasus-backend/internal/modules/users/internal/oauth"
)

// Repositories are the repositories of the package. module.go builds them from the pool, so the
// services never hold it.
type Repositories struct {
	users *userRepository
}

func NewRepositories(db *gorm.DB) Repositories {
	return Repositories{users: &userRepository{db: db}}
}

type userRepository struct {
	db *gorm.DB
}

func (r *userRepository) lockAdministration(ctx context.Context) error {
	if err := database.RequireTransaction(ctx); err != nil {
		return err
	}

	// Authorization reads and their writes must see the same bootstrap assignment during a dev seed.
	err := database.Conn(ctx, r.db).Exec("LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE").Error
	if err != nil {
		return fmt.Errorf("lock user administration: %w", err)
	}

	return nil
}

func (r *userRepository) getBootstrapAdmin(ctx context.Context) (UserRecord, bool, error) {
	return r.getWhere(ctx, "get the bootstrap administrator", "is_root_admin = ?", true)
}

func (r *userRepository) updateBootstrapAdmin(ctx context.Context, userID uuid.UUID, isBootstrapAdmin bool) error {
	err := database.Conn(ctx, r.db).
		Exec("UPDATE users SET is_root_admin = ? WHERE id = ?", isBootstrapAdmin, userID).
		Error
	if err != nil {
		return fmt.Errorf("update the bootstrap administrator designation: %w", err)
	}

	return nil
}

func (r *userRepository) getByID(ctx context.Context, userID uuid.UUID) (UserRecord, bool, error) {
	return r.getWhere(ctx, "get the user by id", "id = ?", userID)
}

func (r *userRepository) listByIDs(ctx context.Context, userIDs []uuid.UUID) ([]UserRecord, error) {
	var records []UserRecord

	err := database.Conn(ctx, r.db).Where("id IN ?", userIDs).Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("list the users by id: %w", err)
	}

	return records, nil
}

func (r *userRepository) getByEmail(ctx context.Context, email string) (UserRecord, bool, error) {
	return r.getWhere(ctx, "get the user by email", "email = ?", email)
}

func (r *userRepository) getByOAuthID(
	ctx context.Context,
	provider oauth.Provider,
	oauthID string,
) (UserRecord, bool, error) {
	column, err := getOAuthIDColumn(provider)
	if err != nil {
		return UserRecord{}, false, err
	}

	return r.getWhere(ctx, "get the user by the "+string(provider)+" account", column+" = ?", oauthID)
}

func (r *userRepository) linkOAuthID(
	ctx context.Context,
	userID uuid.UUID,
	provider oauth.Provider,
	oauthID string,
) error {
	column, err := getOAuthIDColumn(provider)
	if err != nil {
		return err
	}

	err = database.Conn(ctx, r.db).Exec("UPDATE users SET "+column+" = ? WHERE id = ?", oauthID, userID).Error
	if err != nil {
		return fmt.Errorf("link the %s account to the user: %w", provider, err)
	}

	return nil
}

func (r *userRepository) countAll(ctx context.Context) (int64, error) {
	var userCount int64

	err := database.Conn(ctx, r.db).Model(&UserRecord{}).Count(&userCount).Error
	if err != nil {
		return 0, fmt.Errorf("count the users: %w", err)
	}

	return userCount, nil
}

func (r *userRepository) insert(ctx context.Context, record UserRecord) error {
	err := database.Conn(ctx, r.db).Create(&record).Error
	if err != nil {
		return fmt.Errorf("insert the user: %w", err)
	}

	return nil
}

func (r *userRepository) updateProfile(ctx context.Context, userID uuid.UUID, name, email string) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", name, email, userID).Error
	if err != nil {
		return fmt.Errorf("update the profile of the user: %w", err)
	}

	return nil
}

type passwordReplacement struct {
	userID         uuid.UUID
	hashedPassword string
	createdAt      time.Time
}

func (r *userRepository) updatePassword(ctx context.Context, replacement passwordReplacement) error {
	err := database.Conn(ctx, r.db).Exec(
		"UPDATE users SET hashed_password = ?, password_creation_time = ?, password_version = ? WHERE id = ?",
		replacement.hashedPassword, replacement.createdAt, uuid.New(), replacement.userID,
	).Error
	if err != nil {
		return fmt.Errorf("update the password of the user: %w", err)
	}

	return nil
}

func (r *userRepository) list(ctx context.Context, filter userListFilter) ([]UserRecord, error) {
	query := database.Conn(ctx, r.db).Order("created_at DESC, id DESC").Limit(filter.limit)

	if filter.after != nil {
		query = query.Where("(created_at, id) < (?, ?)", filter.after.createdAt, filter.after.userID)
	}

	if filter.search != "" {
		pattern := "%" + escapeLikePattern(filter.search) + "%"
		query = query.Where("(email ILIKE ? OR name ILIKE ?)", pattern, pattern)
	}

	var records []UserRecord

	err := query.Find(&records).Error
	if err != nil {
		return nil, fmt.Errorf("list the users: %w", err)
	}

	return records, nil
}

func (r *userRepository) updateStatus(ctx context.Context, userID uuid.UUID, status identity.Status) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE users SET status = ? WHERE id = ?", status, userID).Error
	if err != nil {
		return fmt.Errorf("update the status of the user: %w", err)
	}

	return nil
}

func (r *userRepository) updateRole(ctx context.Context, userID uuid.UUID, role identity.Role) error {
	err := database.Conn(ctx, r.db).Exec("UPDATE users SET role = ? WHERE id = ?", role, userID).Error
	if err != nil {
		return fmt.Errorf("update the role of the user: %w", err)
	}

	return nil
}

// listAdmins lists the users with the admin role, the oldest first.
func (r *userRepository) listAdmins(ctx context.Context) ([]UserRecord, error) {
	var admins []UserRecord

	err := database.Conn(ctx, r.db).
		Where("role = ?", identity.RoleAdmin).
		Order("created_at ASC").
		Find(&admins).Error
	if err != nil {
		return nil, fmt.Errorf("list the admins: %w", err)
	}

	return admins, nil
}

// activateInvited turns an invited user into an active one with the name and the password they
// chose, as one transition.
func (r *userRepository) activateInvited(ctx context.Context, activation invitedUserActivation) error {
	err := database.Conn(ctx, r.db).Exec(
		`UPDATE users
		SET status = ?, name = ?, hashed_password = ?, password_creation_time = ?, password_version = ?
		WHERE id = ? AND status = ?`,
		identity.StatusActive,
		activation.name,
		activation.hashedPassword,
		activation.passwordCreationTime,
		activation.passwordVersion,
		activation.userID,
		identity.StatusInvited,
	).Error
	if err != nil {
		return fmt.Errorf("activate the invited user: %w", err)
	}

	return nil
}

func (r *userRepository) getWhere(
	ctx context.Context,
	operation string,
	condition string,
	value any,
) (UserRecord, bool, error) {
	var record UserRecord

	err := database.Conn(ctx, r.db).Where(condition, value).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return UserRecord{}, false, nil
	}

	if err != nil {
		return UserRecord{}, false, fmt.Errorf("%s: %w", operation, err)
	}

	return record, true, nil
}

// escapeLikePattern makes the wildcards of a search text match themselves.
func escapeLikePattern(text string) string {
	return strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(text)
}

func getOAuthIDColumn(provider oauth.Provider) (string, error) {
	switch provider {
	case oauth.ProviderGitHub:
		return "github_oauth_id", nil
	case oauth.ProviderGoogle:
		return "google_oauth_id", nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownOAuthProvider, provider)
	}
}

type userListFilter struct {
	limit  int
	after  *userCursor
	search string
}

type invitedUserActivation struct {
	userID               uuid.UUID
	name                 string
	hashedPassword       *string
	passwordCreationTime time.Time
	passwordVersion      uuid.UUID
}
