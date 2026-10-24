package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

// The all-database built-ins live on admin, so a custom role that merely shares their name on
// another database must not short-circuit the privilege scan.
func testTestConnectionClusterRoleNameOnAnotherDatabase(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) {
	container := connectToMongodbEndpoint(t, endpoint, version)
	t.Cleanup(func() { _ = container.Client.Disconnect(context.Background()) })

	ctx := t.Context()
	suffix := uuid.New().String()[:8]
	databaseName := "rolename_scope_" + suffix
	username := "rolename_reader_" + suffix
	password := "rolenamereaderpassword123"

	database := container.Client.Database(databaseName)

	_, err := database.Collection("orders_"+suffix).InsertOne(ctx, bson.M{"data": "row1"})
	require.NoError(t, err)

	t.Cleanup(func() { _ = database.Drop(context.Background()) })

	require.NoError(t, database.RunCommand(ctx, bson.D{
		{Key: "createRole", Value: "backup"},
		{Key: "privileges", Value: bson.A{}},
		{Key: "roles", Value: bson.A{}},
	}).Err())

	t.Cleanup(func() {
		_ = database.RunCommand(
			context.Background(),
			bson.D{{Key: "dropRole", Value: "backup"}},
		).Err()
	})

	adminDatabase := container.Client.Database(container.AuthDatabase)

	require.NoError(t, adminDatabase.RunCommand(ctx, bson.D{
		{Key: "createUser", Value: username},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{
			bson.D{{Key: "role", Value: "backup"}, {Key: "db", Value: databaseName}},
		}},
	}).Err())

	t.Cleanup(func() { dropUserSafe(container.Client, username, container.AuthDatabase) })

	port := container.Port
	mongodbModel := &MongodbDatabase{
		Version:      version,
		Host:         container.Host,
		Port:         &port,
		Username:     username,
		Password:     password,
		Database:     databaseName,
		AuthDatabase: container.AuthDatabase,
		CpuCount:     1,
	}

	err = mongodbModel.TestConnection(slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

// A privilege naming one collection covers only that collection, so it cannot stand in for
// permission to read the whole database.
func testTestConnectionFindGrantedOnSingleCollection(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) {
	// t.Context() is already canceled when cleanups run, so they carry their own context, and
	// the disconnect registers first so it runs after every drop below.
	container := connectToMongodbEndpoint(t, endpoint, version)
	t.Cleanup(func() { _ = container.Client.Disconnect(context.Background()) })

	ctx := t.Context()
	suffix := uuid.New().String()[:8]
	readableCollectionName := "readable_" + suffix
	blockedCollectionName := "blocked_" + suffix
	// The version's subtests share one server (docs/testing.md, "Isolation"), so an own database keeps the dump set
	// free of the collections the other subtests leave behind.
	databaseName := "collection_scope_" + suffix
	roleName := "single_collection_reader_" + suffix
	username := "collection_reader_" + suffix
	password := "collectionreaderpassword123"

	database := container.Client.Database(databaseName)

	for _, collectionName := range []string{readableCollectionName, blockedCollectionName} {
		_, err := database.Collection(collectionName).InsertOne(ctx, bson.M{"data": "row1"})
		require.NoError(t, err)

		t.Cleanup(func() { _ = database.Collection(collectionName).Drop(context.Background()) })
	}

	adminDatabase := container.Client.Database(container.AuthDatabase)

	require.NoError(t, adminDatabase.RunCommand(ctx, bson.D{
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
		_ = adminDatabase.RunCommand(
			context.Background(),
			bson.D{{Key: "dropRole", Value: roleName}},
		).Err()
	})

	require.NoError(t, adminDatabase.RunCommand(ctx, bson.D{
		{Key: "createUser", Value: username},
		{Key: "pwd", Value: password},
		{Key: "roles", Value: bson.A{
			bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: container.AuthDatabase}},
		}},
	}).Err())

	t.Cleanup(func() { dropUserSafe(container.Client, username, container.AuthDatabase) })
	t.Cleanup(func() { _ = database.Drop(context.Background()) })

	port := container.Port
	mongodbModel := &MongodbDatabase{
		Version:      version,
		Host:         container.Host,
		Port:         &port,
		Username:     username,
		Password:     password,
		Database:     databaseName,
		AuthDatabase: container.AuthDatabase,
		CpuCount:     1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err := mongodbModel.TestConnection(logger, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
	assert.Contains(t, err.Error(), blockedCollectionName)
	assert.NotContains(t, err.Error(), readableCollectionName)

	mongodbModel.ExcludeCollections = []string{blockedCollectionName}

	assert.NoError(
		t,
		mongodbModel.TestConnection(logger, nil),
		fmt.Sprintf("excluding %s should leave only readable collections", blockedCollectionName),
	)
}
