package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	containers "databasus-backend/internal/lib/testcontainers"
)

type supportedServer struct {
	name    string
	version Version
	image   string
}

func getSupportedServers() []supportedServer {
	return []supportedServer{
		{"MongoDB 5.0", Version5, "mongo:5.0"},
		{"MongoDB 6.0", Version6, "mongo:6.0"},
		{"MongoDB 7.0", Version7, "mongo:7.0"},
		{"MongoDB 8.2", Version8, "mongo:8.2.3-noble"},
		{"MongoDB 9.0", Version9, "mongo:9.0.2-noble"},
	}
}

// testServer is one MongoDB server shared by the subtests of its version. Each subtest works in a
// database and with users of its own, so the subtests run in parallel.
type testServer struct {
	Endpoint   containers.Endpoint
	Version    Version
	RootClient *mongo.Client
}

// TestSourceAcrossSupportedVersions boots each version once and runs every scenario that needs a
// server against it. It is not parallel: docs/testing.md keeps one server per package alive at a
// time, and TestOpenTunnel boots one of its own.
func TestSourceAcrossSupportedVersions(t *testing.T) {
	for _, server := range getSupportedServers() {
		t.Run(server.name, func(t *testing.T) {
			startedServer := startTestServer(t, server.image, server.version)

			runConnectionScenarios(t, startedServer)
			runDumpPrivilegeScenarios(t, startedServer)
			runReadOnlyUserScenarios(t, startedServer)
		})
	}
}

func runConnectionScenarios(t *testing.T, server testServer) {
	t.Helper()

	t.Run("returns the version when the user has the read role on the database", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "read_role")
		username, password := createUser(t, server, bson.A{
			bson.D{{Key: "role", Value: "read"}, {Key: "db", Value: databaseName}},
		})

		detectedVersion, err := NewSource(server.getSettings(username, password, databaseName)).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.NoError(t, err)
		assert.Equal(t, server.Version, detectedVersion)
	})

	t.Run("returns insufficient permissions when the user has no roles", func(t *testing.T) {
		t.Parallel()
		databaseName := createDatabaseWithDocument(t, server, "no_roles")
		username, password := createUser(t, server, bson.A{})

		_, err := NewSource(server.getSettings(username, password, databaseName)).
			TestConnection(t.Context(), newTestLogger(), nil)

		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})

	t.Run("detects the server version", func(t *testing.T) {
		t.Parallel()

		detectedVersion, err := NewSource(server.getRootSettings(containers.MongodbDatabase)).
			DetectVersion(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.Equal(t, server.Version, detectedVersion)
	})

	t.Run("returns a positive raw size after documents are inserted", func(t *testing.T) {
		t.Parallel()
		databaseName := "size_" + newNameSuffix()
		collection := server.RootClient.Database(databaseName).Collection("documents")
		t.Cleanup(func() { _ = server.RootClient.Database(databaseName).Drop(getCleanupContext(t)) })

		documents := make([]any, 0, 1000)
		for range 1000 {
			documents = append(documents, bson.M{"payload": strings.Repeat("x", 1024)})
		}

		_, err := collection.InsertMany(t.Context(), documents)
		require.NoError(t, err)

		rawSizeMb, err := NewSource(server.getRootSettings(databaseName)).GetRawSizeMb(t.Context(), newTestLogger())

		require.NoError(t, err)
		assert.Greater(t, rawSizeMb, 0.0)
	})
}

func startTestServer(t *testing.T, image string, version Version) testServer {
	t.Helper()

	endpoint := containers.StartMongodb(t, image)

	rootClient := connectToServer(t, endpoint, containers.MongodbUsername, containers.MongodbPassword)

	return testServer{Endpoint: endpoint, Version: version, RootClient: rootClient}
}

func connectToServer(t *testing.T, endpoint containers.Endpoint, username, password string) *mongo.Client {
	t.Helper()

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/?authSource=%s&serverSelectionTimeoutMS=5000&connectTimeoutMS=5000",
		url.QueryEscape(username),
		url.QueryEscape(password),
		endpoint.Host,
		endpoint.Port,
		containers.MongodbAuthDatabase,
	)

	client, err := mongo.Connect(t.Context(), options.Client().ApplyURI(uri))
	require.NoError(t, err)

	t.Cleanup(func() { _ = client.Disconnect(getCleanupContext(t)) })

	require.NoError(t, client.Ping(t.Context(), nil))

	return client
}

func (s testServer) getSettings(username, password, databaseName string) Settings {
	return Settings{
		Host:               s.Endpoint.Host,
		Port:               s.Endpoint.Port,
		Username:           username,
		Password:           password,
		Database:           databaseName,
		AuthDatabase:       containers.MongodbAuthDatabase,
		IsTLS:              false,
		IsSrv:              false,
		IsDirectConnection: false,
		SSHTunnel:          nil,
	}
}

func (s testServer) getRootSettings(databaseName string) Settings {
	return s.getSettings(containers.MongodbUsername, containers.MongodbPassword, databaseName)
}

// createUser creates a user on admin with the given roles and drops it when the test ends.
func createUser(t *testing.T, server testServer, roles bson.A) (string, string) {
	t.Helper()

	username := "user_" + newNameSuffix()
	password := "userpassword123"

	createUserNamed(t, server, username, password, roles)

	return username, password
}

func createUserNamed(t *testing.T, server testServer, username, password string, roles bson.A) {
	t.Helper()

	adminDatabase := server.RootClient.Database(containers.MongodbAuthDatabase)

	require.NoError(t, adminDatabase.RunCommand(t.Context(), bson.D{
		{Key: "createUser", Value: username},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: roles},
	}).Err())

	dropUserOnCleanup(t, server, username)
}

func dropUserOnCleanup(t *testing.T, server testServer, username string) {
	t.Helper()

	t.Cleanup(func() {
		_ = server.RootClient.Database(containers.MongodbAuthDatabase).
			RunCommand(getCleanupContext(t), bson.D{{Key: "dropUser", Value: username}}).
			Err()
	})
}

// createDatabaseWithDocument creates a database of the test's own with one collection, and drops
// it when the test ends.
func createDatabaseWithDocument(t *testing.T, server testServer, prefix string) string {
	t.Helper()

	databaseName := prefix + "_" + newNameSuffix()
	database := server.RootClient.Database(databaseName)

	_, err := database.Collection("documents").InsertOne(t.Context(), bson.M{"data": "row1"})
	require.NoError(t, err)

	t.Cleanup(func() { _ = database.Drop(getCleanupContext(t)) })

	return databaseName
}

func newNameSuffix() string {
	return uuid.New().String()[:8]
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

// t.Context() is already canceled when cleanups run.
func getCleanupContext(t *testing.T) context.Context {
	t.Helper()

	return context.WithoutCancel(t.Context())
}
