package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"

	containers "databasus-backend/internal/lib/testcontainers"
)

func runDumpPrivilegeScenarios(t *testing.T, server testServer) {
	t.Helper()

	// The all-database built-ins live on admin, so a custom role that merely shares their name on
	// another database must not short-circuit the privilege scan.
	t.Run("returns insufficient permissions when a cluster role name lives on another database", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "rolename_scope")
		database := server.RootClient.Database(databaseName)

		require.NoError(t, database.RunCommand(t.Context(), bson.D{
			{Key: "createRole", Value: "backup"},
			{Key: "privileges", Value: bson.A{}},
			{Key: "roles", Value: bson.A{}},
		}).Err())

		t.Cleanup(func() {
			_ = database.RunCommand(getCleanupContext(t), bson.D{{Key: "dropRole", Value: "backup"}}).Err()
		})

		username, password := createUser(t, server, bson.A{
			bson.D{{Key: "role", Value: "backup"}, {Key: "db", Value: databaseName}},
		})

		_, err := NewSource(server.getSettings(username, password, databaseName)).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})

	// A privilege naming one collection covers only that collection, so it cannot stand in for
	// permission to read the whole database.
	t.Run("returns insufficient permissions when find is granted on a single collection", func(t *testing.T) {
		t.Parallel()
		fixture := createSingleCollectionReader(t, server)

		_, err := NewSource(fixture.Settings).TestConnection(t.Context(), newTestLogger(), nil)

		require.ErrorIs(t, err, ErrInsufficientPermissions)
		assert.Contains(t, err.Error(), fixture.BlockedCollectionName)
		assert.NotContains(t, err.Error(), fixture.ReadableCollectionName)
	})

	t.Run("accepts the user when the unreadable collection is excluded", func(t *testing.T) {
		t.Parallel()
		fixture := createSingleCollectionReader(t, server)

		_, err := NewSource(fixture.Settings).
			TestConnection(t.Context(), newTestLogger(), []string{fixture.BlockedCollectionName})

		require.NoError(t, err)
	})
}

type singleCollectionReader struct {
	Settings               Settings
	ReadableCollectionName string
	BlockedCollectionName  string
}

// createSingleCollectionReader creates a database with two collections and a user that may find in
// one of them and list the collections of the database.
func createSingleCollectionReader(t *testing.T, server testServer) singleCollectionReader {
	t.Helper()

	suffix := newNameSuffix()
	readableCollectionName := "readable_" + suffix
	blockedCollectionName := "blocked_" + suffix
	databaseName := "collection_scope_" + suffix
	roleName := "single_collection_reader_" + suffix

	database := server.RootClient.Database(databaseName)
	t.Cleanup(func() { _ = database.Drop(getCleanupContext(t)) })

	for _, collectionName := range []string{readableCollectionName, blockedCollectionName} {
		_, err := database.Collection(collectionName).InsertOne(t.Context(), bson.M{"data": "row1"})
		require.NoError(t, err)
	}

	adminDatabase := server.RootClient.Database(containers.MongodbAuthDatabase)

	require.NoError(t, adminDatabase.RunCommand(t.Context(), bson.D{
		{Key: "createRole", Value: roleName},
		{Key: "privileges", Value: bson.A{
			bson.D{
				{Key: "resource", Value: bson.D{
					{Key: "db", Value: databaseName},
					{Key: "collection", Value: readableCollectionName},
				}},
				{Key: "actions", Value: bson.A{"find"}},
			},
			// Listing collections is what lets the check see the whole dump set, so the role has
			// it: without it the role could not even reach the interesting comparison.
			bson.D{
				{Key: "resource", Value: bson.D{
					{Key: "db", Value: databaseName},
					{Key: "collection", Value: ""},
				}},
				{Key: "actions", Value: bson.A{"listCollections"}},
			},
		}},
		{Key: "roles", Value: bson.A{}},
	}).Err())

	t.Cleanup(func() {
		_ = adminDatabase.RunCommand(getCleanupContext(t), bson.D{{Key: "dropRole", Value: roleName}}).Err()
	})

	username, password := createUser(t, server, bson.A{
		bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: containers.MongodbAuthDatabase}},
	})

	return singleCollectionReader{
		Settings:               server.getSettings(username, password, databaseName),
		ReadableCollectionName: readableCollectionName,
		BlockedCollectionName:  blockedCollectionName,
	}
}
