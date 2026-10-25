package users_testing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	users_bridge "databasus-backend/internal/features/users"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
)

// testSessionLifetime outlives any test run.
const testSessionLifetime = 24 * time.Hour

// testPasswordHash matches no password, so a test user signs in only with the token it gets.
const testPasswordHash = "$2a$10$test"

// TestUser is a user the helpers inserted, with a session token the users module accepts.
type TestUser struct {
	UserID uuid.UUID
	Email  string
	Token  string
}

func CreateTestUser(ctx context.Context, role users.Role) *TestUser {
	return insertTestUser(ctx, role, false)
}

// CreateTestUserWithEmail reaches an address the profile update would refuse, such as the
// placeholder address of an upgraded instance.
func CreateTestUserWithEmail(ctx context.Context, role users.Role, email string) *TestUser {
	testUser := insertTestUser(ctx, role, false)

	SetTestUserEmail(ctx, testUser.UserID, email)
	testUser.Email = email

	return testUser
}

func GetTestUser(ctx context.Context, userID uuid.UUID) users.User {
	user, isFound, err := users_bridge.GetModule().GetUser(ctx, userID)
	if err != nil {
		panic(fmt.Sprintf("failed to read test user: %v", err))
	}

	if !isFound {
		panic("test user not found: " + userID.String())
	}

	return user
}

func SetTestUserEmail(ctx context.Context, userID uuid.UUID, email string) {
	executeSQL(ctx, "UPDATE users SET email = ? WHERE id = ?", email, userID)
}

func SetTestUserName(ctx context.Context, userID uuid.UUID, name string) {
	executeSQL(ctx, "UPDATE users SET name = ? WHERE id = ?", name, userID)
}

func DeactivateTestUser(ctx context.Context, userID uuid.UUID) {
	executeSQL(ctx, "UPDATE users SET status = ? WHERE id = ?", users.StatusInactive, userID)
}

func DeleteTestUser(ctx context.Context, userID uuid.UUID) {
	executeSQL(ctx, "DELETE FROM users WHERE id = ?", userID)
}

func RecreateInitAdminAndGetAccess(ctx context.Context) *TestUser {
	return RecreateInitialAdmin(ctx)
}

// RecreateInitialAdmin finds the outgoing bootstrap administrator by its record, not by its
// address: a test that moved it to another address would otherwise leave the record behind, and the
// partial unique index would refuse the next one.
func RecreateInitialAdmin(ctx context.Context) *TestUser {
	executeSQL(ctx, "UPDATE users SET email = ?, is_root_admin = FALSE WHERE is_root_admin",
		"retired-admin-"+uuid.NewString())

	return insertTestUser(ctx, users.RoleAdmin, true)
}

// DeleteAllUsers empties the users table. The tests of one package share the database of their
// test process, so a test keyed on the first-user rule has to make the instance empty itself.
func DeleteAllUsers() {
	executeSQL(context.Background(), "DELETE FROM users")
}

// IssueTestToken signs a session token for a stored user with the claims the users module issues.
func IssueTestToken(ctx context.Context, userID uuid.UUID) string {
	var storedUser struct {
		Role                 string
		PasswordCreationTime time.Time
	}

	err := storage.GetDb().WithContext(ctx).
		Raw("SELECT role, password_creation_time FROM users WHERE id = ?", userID).
		Scan(&storedUser).Error
	if err != nil {
		panic(fmt.Errorf("read a test user: %w", err))
	}

	now := time.Now().UTC()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":                  userID.String(),
		"exp":                  now.Add(testSessionLifetime).Unix(),
		"iat":                  now.Unix(),
		"role":                 storedUser.Role,
		"passwordCreationTime": storedUser.PasswordCreationTime.Unix(),
	})

	signedToken, err := token.SignedString([]byte(encryption.GetMasterKey().GetSecret()))
	if err != nil {
		panic(fmt.Errorf("sign a test session token: %w", err))
	}

	return signedToken
}

func insertTestUser(ctx context.Context, role users.Role, isBootstrapAdmin bool) *TestUser {
	userID := uuid.New()
	email := fmt.Sprintf("%s-%s@test.com", strings.ToLower(string(role)), userID.String()[:8])
	now := time.Now().UTC()

	executeSQL(ctx,
		`INSERT INTO users (id, email, name, hashed_password, password_creation_time, role, status, is_root_admin,
			created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, email, "Test User", testPasswordHash, now, role, users.StatusActive, isBootstrapAdmin, now)

	return &TestUser{UserID: userID, Email: email, Token: IssueTestToken(ctx, userID)}
}

func executeSQL(ctx context.Context, statement string, values ...any) {
	err := storage.GetDb().WithContext(ctx).Exec(statement, values...).Error
	if err != nil {
		panic(fmt.Errorf("run %q: %w", statement, err))
	}
}
