package mongodb

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/tools"
)

type mongodbModelVersion struct {
	name    string
	version tools.MongodbVersion
	image   string
}

var mongodbModelVersions = []mongodbModelVersion{
	{"MongoDB 5.0", tools.MongodbVersion5, "mongo:5.0"},
	{"MongoDB 6.0", tools.MongodbVersion6, "mongo:6.0"},
	{"MongoDB 7.0", tools.MongodbVersion7, "mongo:7.0"},
	{"MongoDB 8.2", tools.MongodbVersion8, "mongo:8.2.3-noble"},
	{"MongoDB 9.0", tools.MongodbVersion9, "mongo:9.0.2-noble"},
}

// Test_MongodbModel_AcrossSupportedVersions boots each MongoDB version once and runs every matrix
// model test against it as a subtest. Only one container is alive per package at a time. See docs/testing.md.
func Test_MongodbModel_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range mongodbModelVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMongodb(t, dbVersion.image)

			t.Run("Test_TestConnection_InsufficientPermissions_ReturnsError", func(t *testing.T) {
				testTestConnectionInsufficientPermissions(t, endpoint, dbVersion.version)
			})

			t.Run("Test_TestConnection_SufficientPermissions_Success", func(t *testing.T) {
				testTestConnectionSufficientPermissions(t, endpoint, dbVersion.version)
			})

			t.Run("Test_ShouldSuggestReadOnlyUser_AdminUser_ReturnsTrue", func(t *testing.T) {
				testShouldSuggestReadOnlyUserAdminUser(t, endpoint, dbVersion.version)
			})

			t.Run("Test_CreateReadOnlyUser_UserCanReadButNotWrite", func(t *testing.T) {
				testCreateReadOnlyUserCanReadButNotWrite(t, endpoint, dbVersion.version)
			})

			t.Run(
				"Test_TestConnection_WhenFindIsGrantedOnSingleCollection_ReturnsErrorUntilCollectionIsExcluded",
				func(t *testing.T) {
					testTestConnectionFindGrantedOnSingleCollection(t, endpoint, dbVersion.version)
				},
			)

			t.Run(
				"Test_TestConnection_WhenClusterRoleNameLivesOnAnotherDatabase_ReturnsError",
				func(t *testing.T) {
					testTestConnectionClusterRoleNameOnAnotherDatabase(t, endpoint, dbVersion.version)
				},
			)
		})
	}
}

func testTestConnectionInsufficientPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) {
	container := connectToMongodbEndpoint(t, endpoint, version)
	defer container.Client.Disconnect(t.Context())

	ctx := t.Context()
	db := container.Client.Database(container.Database)

	_ = db.Collection("permission_test").Drop(ctx)
	_, err := db.Collection("permission_test").InsertOne(ctx, bson.M{"data": "test1"})
	assert.NoError(t, err)

	limitedUsername := fmt.Sprintf("limited_%s", uuid.New().String()[:8])
	limitedPassword := "limitedpassword123"

	adminDB := container.Client.Database(container.AuthDatabase)
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "createUser", Value: limitedUsername},
		{Key: "pwd", Value: limitedPassword},
		{Key: "roles", Value: bson.A{}},
	}).Err()
	assert.NoError(t, err)

	defer dropUserSafe(container.Client, limitedUsername, container.AuthDatabase)

	port := container.Port
	mongodbModel := &MongodbDatabase{
		Version:      version,
		Host:         container.Host,
		Port:         &port,
		Username:     limitedUsername,
		Password:     limitedPassword,
		Database:     container.Database,
		AuthDatabase: container.AuthDatabase,
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mongodbModel.TestConnection(logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient permissions")
}

func testTestConnectionSufficientPermissions(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) {
	container := connectToMongodbEndpoint(t, endpoint, version)
	defer container.Client.Disconnect(t.Context())

	ctx := t.Context()
	db := container.Client.Database(container.Database)

	_ = db.Collection("backup_test").Drop(ctx)
	_, err := db.Collection("backup_test").InsertOne(ctx, bson.M{"data": "test1"})
	assert.NoError(t, err)

	backupUsername := fmt.Sprintf("backup_%s", uuid.New().String()[:8])
	backupPassword := "backuppassword123"

	adminDB := container.Client.Database(container.AuthDatabase)
	err = adminDB.RunCommand(ctx, bson.D{
		{Key: "createUser", Value: backupUsername},
		{Key: "pwd", Value: backupPassword},
		{Key: "roles", Value: bson.A{
			bson.D{
				{Key: "role", Value: "read"},
				{Key: "db", Value: container.Database},
			},
		}},
	}).Err()
	assert.NoError(t, err)

	defer dropUserSafe(container.Client, backupUsername, container.AuthDatabase)

	port := container.Port
	mongodbModel := &MongodbDatabase{
		Host:         container.Host,
		Port:         &port,
		Username:     backupUsername,
		Password:     backupPassword,
		Database:     container.Database,
		AuthDatabase: container.AuthDatabase,
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	err = mongodbModel.TestConnection(logger, nil)
	assert.NoError(t, err)
	assert.Equal(t, version, mongodbModel.Version)
}

func testShouldSuggestReadOnlyUserAdminUser(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) {
	container := connectToMongodbEndpoint(t, endpoint, version)
	defer container.Client.Disconnect(t.Context())

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx := t.Context()

	shouldSuggestReadOnlyUser, roles, err := mongodbModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.True(t, shouldSuggestReadOnlyUser, "Root user must be offered a read-only user")
	assert.NotEmpty(t, roles, "Root user should have roles")
}

func Test_ShouldSuggestReadOnlyUser_ReadOnlyUser_ReturnsFalse(t *testing.T) {
	container := connectToMongodbContainer(t, "mongo:7.0", tools.MongodbVersion7)
	defer container.Client.Disconnect(t.Context())

	ctx := t.Context()
	db := container.Client.Database(container.Database)

	_ = db.Collection("readonly_check_test").Drop(ctx)
	_, err := db.Collection("readonly_check_test").InsertOne(ctx, bson.M{"data": "test1"})
	assert.NoError(t, err)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyModel := &MongodbDatabase{
		Version:      mongodbModel.Version,
		Host:         mongodbModel.Host,
		Port:         mongodbModel.Port,
		Username:     username,
		Password:     password,
		Database:     mongodbModel.Database,
		AuthDatabase: mongodbModel.AuthDatabase,
		IsHttps:      false,
		CpuCount:     1,
	}

	shouldSuggestReadOnlyUser, roles, err := readOnlyModel.ShouldSuggestReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.False(t, shouldSuggestReadOnlyUser, "Read-only user must not be offered another one")
	assert.NotEmpty(t, roles, "Read-only user should have roles (read, backup)")

	dropUserSafe(container.Client, username, container.AuthDatabase)
}

func testCreateReadOnlyUserCanReadButNotWrite(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) {
	container := connectToMongodbEndpoint(t, endpoint, version)
	defer container.Client.Disconnect(t.Context())

	ctx := t.Context()
	db := container.Client.Database(container.Database)

	_ = db.Collection("readonly_test").Drop(ctx)
	_ = db.Collection("hack_collection").Drop(ctx)

	_, err := db.Collection("readonly_test").InsertMany(ctx, []any{
		bson.M{"data": "test1"},
		bson.M{"data": "test2"},
	})
	assert.NoError(t, err)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, username)
	assert.NotEmpty(t, password)
	assert.True(t, strings.HasPrefix(username, "databasus-"))

	if err != nil {
		return
	}

	readOnlyClient := connectWithCredentials(t, container, username, password)
	defer readOnlyClient.Disconnect(ctx)

	readOnlyDB := readOnlyClient.Database(container.Database)

	var count int64
	count, err = readOnlyDB.Collection("readonly_test").CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)

	_, err = readOnlyDB.Collection("readonly_test").
		InsertOne(ctx, bson.M{"data": "should-fail"})
	assert.Error(t, err)
	assertWriteDenied(t, err)

	_, err = readOnlyDB.Collection("readonly_test").UpdateOne(
		ctx,
		bson.M{"data": "test1"},
		bson.M{"$set": bson.M{"data": "hacked"}},
	)
	assert.Error(t, err)
	assertWriteDenied(t, err)

	_, err = readOnlyDB.Collection("readonly_test").DeleteOne(ctx, bson.M{"data": "test1"})
	assert.Error(t, err)
	assertWriteDenied(t, err)

	err = readOnlyDB.CreateCollection(ctx, "hack_collection")
	assert.Error(t, err)
	assertWriteDenied(t, err)

	dropUserSafe(container.Client, username, container.AuthDatabase)
}

func Test_ReadOnlyUser_FutureCollections_CanSelect(t *testing.T) {
	container := connectToMongodbContainer(t, "mongo:7.0", tools.MongodbVersion7)
	defer container.Client.Disconnect(t.Context())

	ctx := t.Context()
	db := container.Client.Database(container.Database)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	_ = db.Collection("future_collection").Drop(ctx)
	_, err = db.Collection("future_collection").InsertOne(ctx, bson.M{"data": "future_data"})
	assert.NoError(t, err)

	readOnlyClient := connectWithCredentials(t, container, username, password)
	defer readOnlyClient.Disconnect(ctx)

	readOnlyDB := readOnlyClient.Database(container.Database)

	var result bson.M
	err = readOnlyDB.Collection("future_collection").FindOne(ctx, bson.M{}).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "future_data", result["data"])

	dropUserSafe(container.Client, username, container.AuthDatabase)
}

func Test_ReadOnlyUser_CannotDropOrModifyCollections(t *testing.T) {
	container := connectToMongodbContainer(t, "mongo:7.0", tools.MongodbVersion7)
	defer container.Client.Disconnect(t.Context())

	ctx := t.Context()
	db := container.Client.Database(container.Database)

	_ = db.Collection("drop_test").Drop(ctx)
	_, err := db.Collection("drop_test").InsertOne(ctx, bson.M{"data": "test1"})
	assert.NoError(t, err)

	mongodbModel := createMongodbModel(container)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	username, password, err := mongodbModel.CreateReadOnlyUser(ctx, logger, nil)
	assert.NoError(t, err)

	readOnlyClient := connectWithCredentials(t, container, username, password)
	defer readOnlyClient.Disconnect(ctx)

	readOnlyDB := readOnlyClient.Database(container.Database)

	err = readOnlyDB.Collection("drop_test").Drop(ctx)
	assert.Error(t, err)
	assertWriteDenied(t, err)

	_, err = readOnlyDB.Collection("drop_test").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "data", Value: 1}},
	})
	assert.Error(t, err)
	assertWriteDenied(t, err)

	dropUserSafe(container.Client, username, container.AuthDatabase)
}

type MongodbContainer struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Database     string
	AuthDatabase string
	Version      tools.MongodbVersion
	Client       *mongo.Client
}

func Test_GetRawDbSizeMb_Mongodb_ReturnsPositiveSize(t *testing.T) {
	container := connectToMongodbContainer(t, "mongo:7.0", tools.MongodbVersion7)
	defer container.Client.Disconnect(t.Context())

	collectionName := fmt.Sprintf("size_test_%s", uuid.New().String()[:8])
	collection := container.Client.Database(container.Database).Collection(collectionName)

	defer func() {
		_ = collection.Drop(t.Context())
	}()

	docs := make([]any, 0, 1000)
	for i := 0; i < 1000; i++ {
		docs = append(docs, bson.M{"payload": strings.Repeat("x", 1024)})
	}
	_, err := collection.InsertMany(t.Context(), docs)
	assert.NoError(t, err)

	mongodbModel := createMongodbModel(container)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	sizeMB, err := mongodbModel.GetRawDbSizeMb(t.Context(), logger, nil)
	assert.NoError(t, err)
	assert.Greater(t, sizeMB, 0.0, "raw db size should be > 0 after inserting documents")
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	port := 27017
	mongodbModel := &MongodbDatabase{
		Version:            tools.MongodbVersion7,
		Host:               "db.example.com",
		Port:               &port,
		Username:           "appuser",
		Password:           "supersecret",
		Database:           "appdb",
		AuthDatabase:       "admin",
		IsHttps:            true,
		IsSrv:              true,
		IsDirectConnection: true,
		CpuCount:           4,
		ExcludeCollections: []string{"audit_logs"},
	}

	mongodbModel.HideSensitiveData()

	assert.Empty(t, mongodbModel.Password)
	assert.Equal(t, "db.example.com", mongodbModel.Host)
	assert.Equal(t, &port, mongodbModel.Port)
	assert.Equal(t, "appuser", mongodbModel.Username)
	assert.Equal(t, "appdb", mongodbModel.Database)
	assert.Equal(t, "admin", mongodbModel.AuthDatabase)
	assert.True(t, mongodbModel.IsHttps)
	assert.True(t, mongodbModel.IsSrv)
	assert.True(t, mongodbModel.IsDirectConnection)
	assert.Equal(t, 4, mongodbModel.CpuCount)
	assert.Equal(t, []string{"audit_logs"}, mongodbModel.ExcludeCollections)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var mongodbModel *MongodbDatabase

	assert.NotPanics(t, func() {
		mongodbModel.HideSensitiveData()
	})
}

func connectToMongodbContainer(
	t *testing.T,
	image string,
	version tools.MongodbVersion,
) *MongodbContainer {
	endpoint := containers.StartMongodb(t, image)

	return connectToMongodbEndpoint(t, endpoint, version)
}

func connectToMongodbEndpoint(
	t *testing.T,
	endpoint containers.Endpoint,
	version tools.MongodbVersion,
) *MongodbContainer {
	dbName := containers.MongodbDatabase
	username := containers.MongodbUsername
	password := containers.MongodbPassword
	authDatabase := containers.MongodbAuthDatabase

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s&serverSelectionTimeoutMS=5000&connectTimeoutMS=5000",
		username,
		password,
		endpoint.Host,
		endpoint.Port,
		dbName,
		authDatabase,
	)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB %s: %v", version, err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping MongoDB %s: %v", version, err)
	}

	return &MongodbContainer{
		Host:         endpoint.Host,
		Port:         endpoint.Port,
		Username:     username,
		Password:     password,
		Database:     dbName,
		AuthDatabase: authDatabase,
		Version:      version,
		Client:       client,
	}
}

func createMongodbModel(container *MongodbContainer) *MongodbDatabase {
	port := container.Port
	return &MongodbDatabase{
		Version:      container.Version,
		Host:         container.Host,
		Port:         &port,
		Username:     container.Username,
		Password:     container.Password,
		Database:     container.Database,
		AuthDatabase: container.AuthDatabase,
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}
}

func connectWithCredentials(
	t *testing.T,
	container *MongodbContainer,
	username, password string,
) *mongo.Client {
	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s",
		url.QueryEscape(username), url.QueryEscape(password),
		container.Host, container.Port,
		container.Database, container.AuthDatabase,
	)

	ctx := t.Context()
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	assert.NoError(t, err)

	return client
}

func dropUserSafe(client *mongo.Client, username, authDatabase string) {
	ctx := context.Background()
	adminDB := client.Database(authDatabase)
	_ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
}

func assertWriteDenied(t *testing.T, err error) {
	errStr := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errStr, "not authorized") ||
			strings.Contains(errStr, "unauthorized") ||
			strings.Contains(errStr, "permission denied"),
		"Expected authorization error, got: %v", err)
}

func Test_BuildConnectionURI_WithSrvFormat_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb+srv://")
	assert.Contains(t, uri, "testuser")
	assert.Contains(t, uri, "testpass123")
	assert.Contains(t, uri, "cluster0.example.mongodb.net")
	assert.Contains(t, uri, "/mydb")
	assert.Contains(t, uri, "authSource=admin")
	assert.Contains(t, uri, "connectTimeoutMS=15000")
	assert.NotContains(t, uri, ":27017")
}

func Test_BuildConnectionURI_WithStandardFormat_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "testuser")
	assert.Contains(t, uri, "testpass123")
	assert.Contains(t, uri, "localhost:27017")
	assert.Contains(t, uri, "/mydb")
	assert.Contains(t, uri, "authSource=admin")
	assert.Contains(t, uri, "connectTimeoutMS=15000")
	assert.NotContains(t, uri, "mongodb+srv://")
}

func Test_BuildConnectionURI_WithNullPort_UsesDefault(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "localhost:27017")
}

func Test_Validate_SrvConnection_AllowsNullPort(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
		CpuCount:     1,
	}

	err := model.Validate()

	assert.NoError(t, err)
}

func Test_BuildConnectionURI_WithDirectConnection_ReturnsCorrectUri(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:               "mongo.example.local",
		Port:               &port,
		Username:           "testuser",
		Password:           "testpass123",
		Database:           "mydb",
		AuthDatabase:       "admin",
		IsHttps:            false,
		IsSrv:              false,
		IsDirectConnection: true,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "directConnection=true")
	assert.Contains(t, uri, "mongo.example.local:27017")
	assert.Contains(t, uri, "authSource=admin")
}

func Test_BuildConnectionURI_WithoutDirectConnection_OmitsParam(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:               "localhost",
		Port:               &port,
		Username:           "testuser",
		Password:           "testpass123",
		Database:           "mydb",
		AuthDatabase:       "admin",
		IsHttps:            false,
		IsSrv:              false,
		IsDirectConnection: false,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.NotContains(t, uri, "directConnection")
}

func Test_BuildConnectionURI_WithDirectConnectionAndTls_ReturnsBothParams(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:               "mongo.example.local",
		Port:               &port,
		Username:           "testuser",
		Password:           "testpass123",
		Database:           "mydb",
		AuthDatabase:       "admin",
		IsHttps:            true,
		IsSrv:              false,
		IsDirectConnection: true,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "directConnection=true")
	assert.Contains(t, uri, "tls=true")
	assert.Contains(t, uri, "tlsInsecure=true")
}

func Test_Validate_StandardConnection_RequiresPort(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}

	err := model.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "port is required for standard connections")
}

func Test_BuildConnectionURI_WhenIsHttpsFalse_ContainsTlsFalse(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "tls=false")
	assert.NotContains(t, uri, "tls=true")
	assert.NotContains(t, uri, "tlsInsecure")
}

func Test_BuildConnectionURI_WhenSrvAndIsHttpsFalse_ContainsTlsFalse(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "mongodb+srv://")
	assert.Contains(t, uri, "tls=false")
	assert.NotContains(t, uri, "tls=true")
	assert.NotContains(t, uri, "tlsInsecure")
}

func Test_BuildRestoreURI_OmitsDatabaseFromPath(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.BuildRestoreURI("testpass123")

	assert.Contains(t, uri, "mongodb://")
	assert.Contains(t, uri, "localhost:27017")
	assert.Contains(t, uri, "/?authSource=admin")
	assert.NotContains(t, uri, "/mydb")
}

func Test_BuildRestoreURI_WithSrvOmitsDatabase(t *testing.T) {
	model := &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        true,
	}

	uri := model.BuildRestoreURI("testpass123")

	assert.Contains(t, uri, "mongodb+srv://")
	assert.Contains(t, uri, "/?authSource=admin")
	assert.NotContains(t, uri, "/mydb")
}

func Test_BuildConnectionURI_WhenDatabaseHasSpecialChars_EscapesPathSegment(t *testing.T) {
	port := 27017
	model := &MongodbDatabase{
		Host:         "localhost",
		Port:         &port,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "db/with#hash?and%percent",
		AuthDatabase: "admin",
		IsHttps:      false,
		IsSrv:        false,
	}

	uri := model.BuildConnectionURI("testpass123")

	assert.Contains(t, uri, "/db%2Fwith%23hash%3Fand%25percent?")
	assert.NotContains(t, uri, "/db/with#hash?and%percent")
}

func Test_MapMongodbVersion_VersionMatrix_ReturnsExpected(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name            string
		reportedVersion string
		isRefused       bool
		mappedVersion   tools.MongodbVersion
		errorFragments  []string
	}{
		{"4.2 supported", "4.2.0", false, tools.MongodbVersion4, nil},
		{"4.4 supported", "4.4.29", false, tools.MongodbVersion4, nil},
		{"5.0 supported", "5.0.0", false, tools.MongodbVersion5, nil},
		{"6.0 supported", "6.0.0", false, tools.MongodbVersion6, nil},
		{"7.0 supported", "7.0.0", false, tools.MongodbVersion7, nil},
		{"8.0 supported", "8.0.0", false, tools.MongodbVersion8, nil},
		{"8.10 supported", "8.10.1", false, tools.MongodbVersion8, nil},
		{"9.0 supported", "9.0.2", false, tools.MongodbVersion9, nil},
		{"9.2 supported", "9.2.0", false, tools.MongodbVersion9, nil},
		{"3.6 rejected", "3.6.23", true, "", []string{"3.6", "minimum supported: 4.2"}},
		{"4.0 rejected", "4.0.28", true, "", []string{"4.0", "minimum supported: 4.2"}},
		{"4.1 rejected", "4.1.0", true, "", []string{"4.1", "minimum supported: 4.2"}},
		{"10.0 rejected", "10.0.0", true, "", []string{"10.0", "supported: 4.2 through 9.x"}},
		{"unreadable rejected", "abc", true, "", []string{"could not parse MongoDB version", "abc"}},
		{"empty rejected", "", true, "", []string{"could not parse MongoDB version", `""`}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mappedVersion, err := mapMongodbVersion(tc.reportedVersion)

			if tc.isRefused {
				assert.Error(t, err)
				for _, fragment := range tc.errorFragments {
					assert.Contains(t, err.Error(), fragment)
				}
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tc.mappedVersion, mappedVersion)
		})
	}
}

func srvDatabase() *MongodbDatabase {
	return &MongodbDatabase{
		Host:         "cluster0.example.mongodb.net",
		Port:         nil,
		Username:     "testuser",
		Password:     "testpass123",
		Database:     "mydb",
		AuthDatabase: "admin",
		CpuCount:     1,
		IsSrv:        true,
	}
}

func Test_Validate_WhenSrvIsEnabledBehindAnSshTunnel_IsRejected(t *testing.T) {
	model := srvDatabase()
	model.SshTunnel = sshtunnel.Config{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunneluser",
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  "tunnelpassword",
	}

	err := model.Validate()

	assert.ErrorContains(t, err, "SRV")
}

func Test_Validate_WhenSrvIsEnabledWithoutAnSshTunnel_IsAccepted(t *testing.T) {
	assert.NoError(t, srvDatabase().Validate())
}
