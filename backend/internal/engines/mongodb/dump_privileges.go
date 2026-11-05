package mongodb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"databasus-backend/internal/lib/namelist"
)

const findAction = "find"

//nolint:gochecknoglobals // Shared immutable MongoDB role lookup avoids rebuilding the table for each inherited role; it is never mutated.
var clusterWideBackupRoles = map[string]bool{
	"backup":               true,
	"root":                 true,
	"readAnyDatabase":      true,
	"readWriteAnyDatabase": true,
	"clusterAdmin":         true,
	"__system":             true,
}

//nolint:gochecknoglobals // Shared immutable MongoDB role lookup avoids rebuilding the table for each inherited role; it is never mutated.
var databaseScopedBackupRoles = map[string]bool{
	"read":      true,
	"readWrite": true,
	"dbOwner":   true,
}

// dumpScope is what mongodump reads: the database without the excluded collections, as the user
// that authenticates against AuthDatabase.
type dumpScope struct {
	Username           string
	Database           string
	AuthDatabase       string
	ExcludeCollections []string
}

func checkDumpReadPrivileges(ctx context.Context, client *mongo.Client, scope dumpScope) error {
	user, err := readUserInfo(ctx, client, userInfoRequest{
		Username:         scope.Username,
		AuthDatabase:     scope.AuthDatabase,
		IsShowPrivileges: true,
	})

	switch {
	case errors.Is(err, errUserNotListed):
		return fmt.Errorf("%w. User not found", ErrInsufficientPermissions)
	case errors.Is(err, errUserUnparseable):
		return fmt.Errorf("%w. Could not parse user info", ErrInsufficientPermissions)
	case err != nil:
		return fmt.Errorf("failed to get user info: %w", err)
	}

	account := grantedAccount{Scope: scope, User: user}

	if account.hasBackupRole() {
		return nil
	}

	if account.isFindGrantedOnWholeDatabase() {
		return nil
	}

	return account.checkPerCollectionFindPrivileges(ctx, client)
}

type grantedAccount struct {
	Scope dumpScope
	User  userInfo
}

func (a grantedAccount) hasBackupRole() bool {
	for _, role := range a.User.getRoles() {
		roleName, _ := role["role"].(string)
		roleDatabaseName, _ := role["db"].(string)

		// The all-database built-ins exist only on admin; a custom role that merely shares
		// their name on another database grants none of their reach.
		if clusterWideBackupRoles[roleName] && roleDatabaseName == "admin" {
			return true
		}

		if databaseScopedBackupRoles[roleName] &&
			(roleDatabaseName == a.Scope.Database || roleDatabaseName == "") {
			return true
		}
	}

	return false
}

// A find privilege counts for the whole dump only when its resource is the database itself, the
// cluster, or any resource. A privilege naming one collection covers just that collection, which
// is what the per-collection pass below has to work out.
func (a grantedAccount) isFindGrantedOnWholeDatabase() bool {
	for _, privilege := range a.User.getInheritedPrivileges() {
		resource, isResourceDocument := privilege["resource"].(bson.M)
		if !isResourceDocument || !hasFindAction(privilege) {
			continue
		}

		if isCluster, _ := resource["cluster"].(bool); isCluster {
			return true
		}

		resourceCollectionName, _ := resource["collection"].(string)
		if resourceCollectionName != "" {
			continue
		}

		resourceDatabaseName, _ := resource["db"].(string)
		if resourceDatabaseName == a.Scope.Database || resourceDatabaseName == "" {
			return true
		}
	}

	return false
}

func (a grantedAccount) checkPerCollectionFindPrivileges(
	ctx context.Context,
	client *mongo.Client,
) error {
	dumpedCollectionNames, err := readDumpedCollectionNames(ctx, client, a.Scope)
	if err != nil {
		// A role that cannot list the collections certainly cannot dump them, but a transport
		// failure must keep saying what it was rather than pose as a permission problem.
		if classifyServerError(err) != serverErrorUnauthorized {
			return err
		}

		return a.newInsufficientPrivilegesError(nil)
	}

	readableCollectionNames := a.getCollectionsGrantingFind()

	var unreadableCollectionNames []string

	for _, collectionName := range dumpedCollectionNames {
		if !slices.Contains(readableCollectionNames, collectionName) {
			unreadableCollectionNames = append(unreadableCollectionNames, collectionName)
		}
	}

	if len(unreadableCollectionNames) == 0 {
		return nil
	}

	return a.newInsufficientPrivilegesError(unreadableCollectionNames)
}

func (a grantedAccount) getCollectionsGrantingFind() []string {
	var readableCollectionNames []string

	for _, privilege := range a.User.getInheritedPrivileges() {
		resource, isResourceDocument := privilege["resource"].(bson.M)
		if !isResourceDocument || !hasFindAction(privilege) {
			continue
		}

		resourceCollectionName, _ := resource["collection"].(string)
		if resourceCollectionName == "" {
			continue
		}

		resourceDatabaseName, _ := resource["db"].(string)
		if resourceDatabaseName == a.Scope.Database || resourceDatabaseName == "" {
			readableCollectionNames = append(readableCollectionNames, resourceCollectionName)
		}
	}

	return readableCollectionNames
}

func (a grantedAccount) newInsufficientPrivilegesError(unreadableCollectionNames []string) error {
	if len(unreadableCollectionNames) > 0 {
		return fmt.Errorf(
			"%w. Cannot read collections: %s. "+
				"Required: 'read' role on database '%s' OR 'backup' role on admin OR 'readAnyDatabase' role",
			ErrInsufficientPermissions,
			namelist.FormatTruncatedNames(unreadableCollectionNames),
			a.Scope.Database,
		)
	}

	return fmt.Errorf(
		"%w. Current roles: %s. "+
			"Required: 'read' role on database '%s' OR 'backup' role on admin OR 'readAnyDatabase' role",
		ErrInsufficientPermissions,
		strings.Join(a.User.getRoleNames(), ", "),
		a.Scope.Database,
	)
}

func hasFindAction(privilege bson.M) bool {
	return slices.Contains(getPrivilegeActions(privilege), findAction)
}

func readDumpedCollectionNames(
	ctx context.Context,
	client *mongo.Client,
	scope dumpScope,
) ([]string, error) {
	collectionNames, err := client.Database(scope.Database).ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	excludedCollectionNames := namelist.NormalizeUniqueNames(scope.ExcludeCollections)
	dumpedCollectionNames := make([]string, 0, len(collectionNames))

	for _, collectionName := range collectionNames {
		if slices.Contains(excludedCollectionNames, collectionName) {
			continue
		}

		dumpedCollectionNames = append(dumpedCollectionNames, collectionName)
	}

	return dumpedCollectionNames, nil
}
