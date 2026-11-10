package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"databasus-backend/internal/lib/crypto"
)

const maxReadOnlyUserAttempts = 3

//nolint:gochecknoglobals // Shared immutable MongoDB capability lookup is reused for inherited roles and actions; it is never mutated.
var writeRoles = map[string]bool{
	"readWrite":            true,
	"readWriteAnyDatabase": true,
	"dbAdmin":              true,
	"dbAdminAnyDatabase":   true,
	"userAdmin":            true,
	"userAdminAnyDatabase": true,
	"clusterAdmin":         true,
	"clusterManager":       true,
	"hostManager":          true,
	"root":                 true,
	"dbOwner":              true,
	"restore":              true,
	"__system":             true,
}

//nolint:gochecknoglobals // Shared immutable MongoDB capability lookup is reused for inherited roles and actions; it is never mutated.
var readOnlyRoles = map[string]bool{
	"read":   true,
	"backup": true,
}

//nolint:gochecknoglobals // Shared immutable MongoDB capability lookup is reused for inherited roles and actions; it is never mutated.
var writeActions = map[string]bool{
	"insert":             true,
	"update":             true,
	"remove":             true,
	"createCollection":   true,
	"dropCollection":     true,
	"createIndex":        true,
	"dropIndex":          true,
	"convertToCapped":    true,
	"dropDatabase":       true,
	"renameCollection":   true,
	"createUser":         true,
	"dropUser":           true,
	"updateUser":         true,
	"grantRole":          true,
	"revokeRole":         true,
	"dropRole":           true,
	"createRole":         true,
	"updateRole":         true,
	"enableSharding":     true,
	"shardCollection":    true,
	"addShard":           true,
	"removeShard":        true,
	"shutdown":           true,
	"replSetReconfig":    true,
	"replSetStateChange": true,
}

// ShouldSuggestReadOnlyUser lists the roles of the connecting user and suggests a read-only user
// when a role, or a privilege a custom role inherits, can write.
func (s *Source) ShouldSuggestReadOnlyUser(ctx context.Context, logger *slog.Logger) (bool, []string, error) {
	client, err := s.connect()
	if err != nil {
		return false, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer disconnect(ctx, client, logger)

	user, err := s.readConnectingUser(ctx, client, false)
	if isUnknownUserError(err) {
		return false, nil, nil
	}

	if err != nil {
		return false, nil, fmt.Errorf("failed to get user info: %w", err)
	}

	if !user.hasRoleList() {
		return false, nil, nil
	}

	roleNames := user.getRoleNames()

	if hasWriteRole(roleNames) {
		return true, roleNames, nil
	}

	if areAllRolesReadOnly(roleNames) {
		return false, roleNames, nil
	}

	isWriteInherited, err := s.hasInheritedWriteAction(ctx, client)
	if err != nil {
		return false, nil, err
	}

	return isWriteInherited, roleNames, nil
}

// CreateReadOnlyUser creates a user with the backup role on admin and read on the database, and
// returns its name and password. A generated name that is already taken is retried with a new one;
// any other failure ends the creation.
func (s *Source) CreateReadOnlyUser(ctx context.Context, logger *slog.Logger) (string, string, error) {
	client, err := s.connect()
	if err != nil {
		return "", "", fmt.Errorf("failed to connect to database: %w", err)
	}
	defer disconnect(ctx, client, logger)

	for range maxReadOnlyUserAttempts {
		username := s.generateUsername()

		password, err := s.createReadOnlyUser(ctx, client, username)
		if classifyServerError(err) == serverErrorNameTaken {
			continue
		}

		if err != nil {
			return "", "", fmt.Errorf("failed to create user: %w", err)
		}

		logger.InfoContext(ctx, "read-only mongodb user created successfully", slog.String("username", username))

		return username, password, nil
	}

	return "", "", fmt.Errorf("%w after %d attempts", ErrUserNameCollision, maxReadOnlyUserAttempts)
}

func (s *Source) createReadOnlyUser(ctx context.Context, client *mongo.Client, username string) (string, error) {
	password := crypto.GeneratePassword()

	err := client.Database(s.settings.getAuthDatabaseName()).RunCommand(ctx, bson.D{
		{Key: "createUser", Value: username},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{
			bson.D{
				{Key: "role", Value: "backup"},
				{Key: "db", Value: "admin"},
			},
			bson.D{
				{Key: "role", Value: "read"},
				{Key: "db", Value: s.settings.Database},
			},
		}},
	}).Err()
	if err != nil {
		return "", err
	}

	return password, nil
}

func (s *Source) readConnectingUser(
	ctx context.Context,
	client *mongo.Client,
	isShowPrivileges bool,
) (userInfo, error) {
	return readUserInfo(ctx, client, userInfoRequest{
		Username:         s.settings.Username,
		AuthDatabase:     s.settings.getAuthDatabaseName(),
		IsShowPrivileges: isShowPrivileges,
	})
}

// hasInheritedWriteAction checks the privileges custom roles add up to, which their names do not
// reveal.
func (s *Source) hasInheritedWriteAction(ctx context.Context, client *mongo.Client) (bool, error) {
	user, err := s.readConnectingUser(ctx, client, true)
	if isUnknownUserError(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to get user privileges: %w", err)
	}

	for _, privilege := range user.getInheritedPrivileges() {
		for _, action := range getPrivilegeActions(privilege) {
			if writeActions[action] {
				return true, nil
			}
		}
	}

	return false, nil
}

func hasWriteRole(roleNames []string) bool {
	return slices.ContainsFunc(roleNames, func(roleName string) bool { return writeRoles[roleName] })
}

func areAllRolesReadOnly(roleNames []string) bool {
	if len(roleNames) == 0 {
		return false
	}

	for _, roleName := range roleNames {
		if !readOnlyRoles[roleName] {
			return false
		}
	}

	return true
}

func isUnknownUserError(err error) bool {
	return errors.Is(err, errUserNotListed) || errors.Is(err, errUserUnparseable)
}
