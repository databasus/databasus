package mongodb

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	containers "databasus-backend/internal/lib/testcontainers"
)

func runReadOnlyUserScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("suggests a read-only user when the user is root", func(t *testing.T) {
		t.Parallel()

		isSuggested, roleNames, err := NewSource(server.getRootSettings(containers.MongodbDatabase)).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.True(t, isSuggested)
		assert.Contains(t, roleNames, "root")
	})

	t.Run("does not suggest a read-only user when the user is the created read-only user", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "suggest_read_only")
		readOnlySettings := createReadOnlyUser(t, server, databaseName)

		isSuggested, roleNames, err := NewSource(readOnlySettings).
			ShouldSuggestReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.False(t, isSuggested)
		assert.ElementsMatch(t, []string{"read", "backup"}, roleNames)
	})

	t.Run("creates a user that can read but not write", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "read_not_write")
		readOnlySettings := createReadOnlyUser(t, server, databaseName)

		assert.True(t, strings.HasPrefix(readOnlySettings.Username, "databasus-"))

		readOnlyDatabase := connectToServer(t, server.Endpoint, readOnlySettings.Username, readOnlySettings.Password).
			Database(databaseName)
		collection := readOnlyDatabase.Collection("documents")

		documentCount, err := collection.CountDocuments(t.Context(), bson.M{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), documentCount)

		_, err = collection.InsertOne(t.Context(), bson.M{"data": "should-fail"})
		assertUnauthorized(t, err)

		_, err = collection.UpdateOne(t.Context(), bson.M{"data": "row1"}, bson.M{"$set": bson.M{"data": "changed"}})
		assertUnauthorized(t, err)

		_, err = collection.DeleteOne(t.Context(), bson.M{"data": "row1"})
		assertUnauthorized(t, err)

		assertUnauthorized(t, readOnlyDatabase.CreateCollection(t.Context(), "new_collection"))
		assertUnauthorized(t, collection.Drop(t.Context()))

		_, err = collection.Indexes().CreateOne(t.Context(), mongo.IndexModel{Keys: bson.D{{Key: "data", Value: 1}}})
		assertUnauthorized(t, err)
	})

	t.Run("creates a user that reads collections created after it", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "future_collections")
		readOnlySettings := createReadOnlyUser(t, server, databaseName)

		_, err := server.RootClient.Database(databaseName).Collection("future_collection").
			InsertOne(t.Context(), bson.M{"data": "future_data"})
		require.NoError(t, err)

		var futureDocument bson.M

		err = connectToServer(t, server.Endpoint, readOnlySettings.Username, readOnlySettings.Password).
			Database(databaseName).Collection("future_collection").
			FindOne(t.Context(), bson.M{}).Decode(&futureDocument)

		require.NoError(t, err)
		assert.Equal(t, "future_data", futureDocument["data"])
	})

	t.Run("retries with a new name when the generated name is taken", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "name_taken")
		takenUsername := generateReadOnlyUsername()
		createUserNamed(t, server, takenUsername, "takenpassword123", bson.A{})
		freshUsername := generateReadOnlyUsername()
		dropUserOnCleanup(t, server, freshUsername)

		var generatedNameCount atomic.Int32

		source := NewSource(server.getRootSettings(databaseName))
		source.generateUsername = func() string {
			if generatedNameCount.Add(1) == 1 {
				return takenUsername
			}

			return freshUsername
		}

		createdUsername, _, err := source.CreateReadOnlyUser(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.Equal(t, freshUsername, createdUsername)
		assert.Equal(t, int32(2), generatedNameCount.Load())
	})

	t.Run("returns an error without retrying when the user may not create users", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "create_refused")
		username, password := createUser(t, server, bson.A{
			bson.D{{Key: "role", Value: "read"}, {Key: "db", Value: databaseName}},
		})

		var generatedNameCount atomic.Int32

		source := NewSource(server.getSettings(username, password, databaseName))
		source.generateUsername = func() string {
			generatedNameCount.Add(1)

			return generateReadOnlyUsername()
		}

		_, _, err := source.CreateReadOnlyUser(t.Context(), newTestLogger())

		require.Error(t, err)
		assert.Equal(t, serverErrorUnauthorized, classifyServerError(err))
		assert.Equal(t, int32(1), generatedNameCount.Load())
	})
}

// createReadOnlyUser creates the read-only user for databaseName as root and returns the settings
// that connect as it.
func createReadOnlyUser(t *testing.T, server testServer, databaseName string) Settings {
	t.Helper()

	username, password, err := NewSource(server.getRootSettings(databaseName)).
		CreateReadOnlyUser(t.Context(), newTestLogger())
	require.NoError(t, err)

	dropUserOnCleanup(t, server, username)

	return server.getSettings(username, password, databaseName)
}

func assertUnauthorized(t *testing.T, err error) {
	t.Helper()

	require.Error(t, err)

	errorText := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errorText, "not authorized") || strings.Contains(errorText, "unauthorized"),
		"expected an authorization error, got: %v", err)
}
