package mongodb_logical

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"databasus-backend/internal/config"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	restores_core "databasus-backend/internal/features/restores/core"
	storages_bridge "databasus-backend/internal/features/storages"
	logicaltesting "databasus-backend/internal/features/tests/logical/shared"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/users"
	test_utils "databasus-backend/internal/util/testing"
	"databasus-backend/internal/util/testing/containers"
)

type MongodbContainer struct {
	Host         string
	Port         int
	Username     string
	Password     string
	Database     string
	AuthDatabase string
	Version      mongodb_engine.Version
	Client       *mongo.Client
}

type MongodbTestDataItem struct {
	ID        string    `bson:"_id"`
	Name      string    `bson:"name"`
	Value     int       `bson:"value"`
	CreatedAt time.Time `bson:"created_at"`
}

type mongodbVersion struct {
	name    string
	version mongodb_engine.Version
	image   string
}

var mongodbVersions = []mongodbVersion{
	{"MongoDB 5.0", mongodb_engine.Version5, "mongo:5.0"},
	{"MongoDB 6.0", mongodb_engine.Version6, "mongo:6.0"},
	{"MongoDB 7.0", mongodb_engine.Version7, "mongo:7.0"},
	{"MongoDB 8.2", mongodb_engine.Version8, "mongo:8.2.3-noble"},
	{"MongoDB 9.0", mongodb_engine.Version9, "mongo:9.0.2-noble"},
}

// Test_MongodbBackupRestore_AcrossSupportedVersions boots each MongoDB version once, runs every
// backup/restore test function against it as a subtest, then shuts it down before the next version.
// Only one matrix container is alive per package at a time. See docs/testing.md.
func Test_MongodbBackupRestore_AcrossSupportedVersions(t *testing.T) {
	for _, dbVersion := range mongodbVersions {
		t.Run(dbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMongodb(t, dbVersion.image)

			t.Run("Test_BackupAndRestoreMongodb_RestoreIsSuccessful", func(t *testing.T) {
				testMongodbBackupRestoreForVersion(t, endpoint, dbVersion.version)
			})
			t.Run("Test_BackupAndRestoreMongodbWithEncryption_RestoreIsSuccessful", func(t *testing.T) {
				testMongodbBackupRestoreWithEncryptionForVersion(t, endpoint, dbVersion.version)
			})
			t.Run(
				"Test_BackupAndRestoreMongodb_WithExcludeCollections_ExcludedCollectionsNotRestored",
				func(t *testing.T) {
					testMongodbBackupRestoreWithExcludeCollectionsForVersion(t, endpoint, dbVersion.version)
				},
			)
			t.Run("Test_BackupAndRestoreMongodb_WithReadOnlyUser_RestoreIsSuccessful", func(t *testing.T) {
				testMongodbBackupRestoreWithReadOnlyUserForVersion(t, endpoint, dbVersion.version)
			})
		})
	}
}

func testMongodbBackupRestoreForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mongodbVersion mongodb_engine.Version,
) {
	container := connectToMongodbEndpoint(t, endpoint, mongodbVersion)
	defer container.Client.Disconnect(t.Context())

	setupMongodbTestData(t, container)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "MongoDB Test Workspace", user)

	storage := storages_bridge.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.Id, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.Id, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.Id, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)
	logicaltesting.AssertBackupSizeMatchesDownloadedBytes(t, router, backup, user.Token)

	newDBName := "restoreddb_mongodb_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	ctx := t.Context()
	_ = container.Client.Database(newDBName).Drop(ctx)

	if removeErr := os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String())); removeErr != nil {
		t.Logf("Warning: Failed to delete backup file: %v", removeErr)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.Id.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
}

func testMongodbBackupRestoreWithEncryptionForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mongodbVersion mongodb_engine.Version,
) {
	container := connectToMongodbEndpoint(t, endpoint, mongodbVersion)
	defer container.Client.Disconnect(t.Context())

	setupMongodbTestData(t, container)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MongoDB Encrypted Test Workspace",
		user)

	storage := storages_bridge.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB Encrypted Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.Id, storage.ID,
		backups_core_enums.BackupEncryptionEncrypted, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.Id, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.Id, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)
	assert.Equal(t, backups_core_enums.BackupEncryptionEncrypted, backup.Encryption)

	newDBName := "restoreddb_mongodb_enc_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	ctx := t.Context()
	_ = container.Client.Database(newDBName).Drop(ctx)

	if removeErr := os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String())); removeErr != nil {
		t.Logf("Warning: Failed to delete backup file: %v", removeErr)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.Id.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
}

func testMongodbBackupRestoreWithReadOnlyUserForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mongodbVersion mongodb_engine.Version,
) {
	container := connectToMongodbEndpoint(t, endpoint, mongodbVersion)
	defer container.Client.Disconnect(t.Context())

	setupMongodbTestData(t, container)

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MongoDB ReadOnly Test Workspace",
		user)

	storage := storages_bridge.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB ReadOnly Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	readOnlyUser := logicaltesting.CreateLimitedUserViaAPI(t, router, database, user.Token)
	assert.NotEmpty(t, readOnlyUser.Username)
	assert.NotEmpty(t, readOnlyUser.Password)

	updatedDatabase := updateMongodbDatabaseCredentialsViaAPI(
		t, router, database,
		readOnlyUser.Username, readOnlyUser.Password,
		user.Token,
	)

	logicaltesting.EnableBackupsViaAPI(
		t, router, updatedDatabase.Id, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, updatedDatabase.Id, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, updatedDatabase.Id, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mongodb_ro_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	verifyMongodbDataIntegrity(t, container, newDBName)

	ctx := t.Context()
	_ = container.Client.Database(newDBName).Drop(ctx)

	dropMongodbUserSafe(container.Client, readOnlyUser.Username, container.AuthDatabase)

	if removeErr := os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String())); removeErr != nil {
		t.Logf("Warning: Failed to delete backup file: %v", removeErr)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+updatedDatabase.Id.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
}

func testMongodbBackupRestoreWithExcludeCollectionsForVersion(
	t *testing.T,
	endpoint containers.Endpoint,
	mongodbVersion mongodb_engine.Version,
) {
	container := connectToMongodbEndpoint(t, endpoint, mongodbVersion)
	defer container.Client.Disconnect(t.Context())

	setupMongodbTestData(t, container)

	ctx := t.Context()
	skipCollection := container.Client.Database(container.Database).Collection("skip_me")
	_ = skipCollection.Drop(ctx)
	_, err := skipCollection.InsertMany(ctx, []any{
		MongodbTestDataItem{ID: "1", Name: "skip1", Value: 1, CreatedAt: time.Now().UTC()},
		MongodbTestDataItem{ID: "2", Name: "skip2", Value: 2, CreatedAt: time.Now().UTC()},
	})
	assert.NoError(t, err)

	defer func() {
		_ = container.Client.Database(container.Database).Collection("skip_me").Drop(ctx)
	}()

	router := logicaltesting.CreateTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(),
		"MongoDB Exclude Collections Test Workspace",
		user)

	storage := storages_bridge.CreateTestStorage(workspace.ID)

	database := createMongodbDatabaseViaAPI(
		t, router, "MongoDB Exclude Collections Test Database", workspace.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		container.Database, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	settings, err := database.Settings.AsMongodbSettings()
	require.NoError(t, err)

	settings.ExcludeCollections = []string{"skip_me"}
	logicaltesting.UpdateDatabaseSettingsViaAPI(t, router, database, settings, user.Token)

	logicaltesting.EnableBackupsViaAPI(
		t, router, database.Id, storage.ID,
		backups_core_enums.BackupEncryptionNone, user.Token,
	)

	logicaltesting.CreateBackupViaAPI(t, router, database.Id, user.Token)

	backup := logicaltesting.WaitForBackupCompletion(t, router, database.Id, user.Token, 5*time.Minute)
	assert.Equal(t, backups_core_logical.BackupStatusCompleted, backup.Status)

	newDBName := "restoreddb_mongodb_excl_" + uuid.New().String()[:8]

	createMongodbRestoreViaAPI(
		t, router, backup.ID,
		container.Host, container.Port,
		container.Username, container.Password,
		newDBName, container.AuthDatabase,
		container.Version,
		user.Token,
	)

	restore := waitForMongodbRestoreCompletion(t, router, backup.ID, user.Token, 5*time.Minute)
	assert.Equal(t, restores_core.RestoreStatusCompleted, restore.Status)

	keptCount, err := container.Client.Database(newDBName).
		Collection("test_data").
		CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)
	assert.Equal(t, int64(3), keptCount, "test_data collection should be restored with 3 documents")

	skippedCount, err := container.Client.Database(newDBName).
		Collection("skip_me").
		CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), skippedCount, "skip_me collection should NOT be restored (excluded)")

	_ = container.Client.Database(newDBName).Drop(ctx)

	if removeErr := os.Remove(filepath.Join(config.GetEnv().DataFolder, backup.ID.String())); removeErr != nil {
		t.Logf("Warning: Failed to delete backup file: %v", removeErr)
	}

	test_utils.MakeDeleteRequest(
		t,
		router,
		"/api/v1/databases/"+database.Id.String(),
		"Bearer "+user.Token,
		http.StatusNoContent,
	)
	storages_bridge.RemoveTestStorage(t.Context(), storage.ID)
	workspaces_testing.RemoveTestWorkspace(t.Context(), workspace)
}

func createMongodbDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	name string,
	workspaceID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	authDatabase string,
	version mongodb_engine.Version,
	token string,
) apiclient.Database {
	mongodbVersion := apiclient.MongodbVersion(version)

	request := logicaltesting.NewDatabaseCreate(t, workspaceID, apiclient.MongodbSettings{
		Host:               host,
		Port:               &port,
		Username:           username,
		Password:           &password,
		Database:           database,
		AuthDatabase:       authDatabase,
		Version:            &mongodbVersion,
		IsHttps:            false,
		CpuCount:           1,
		ExcludeCollections: []string{},
		SshTunnel:          apiclient.SshTunnelSettings{IsEnabled: false},
	})
	request.Name = name

	return logicaltesting.SubmitCreateDatabase(t, router, "MongoDB", request, token)
}

func createMongodbRestoreViaAPI(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	host string,
	port int,
	username string,
	password string,
	database string,
	authDatabase string,
	version mongodb_engine.Version,
	token string,
) {
	request := restores_core.RestoreBackupRequest{
		MongodbDatabase: &restores_core.MongodbTargetDatabase{
			Host:         host,
			Port:         &port,
			Username:     username,
			Password:     password,
			Database:     database,
			AuthDatabase: authDatabase,
			Version:      version,
			IsHttps:      false,
			IsSrv:        false,
			CpuCount:     1,
		},
	}

	logicaltesting.SubmitRestore(t, router, backupID, request, token)
}

func waitForMongodbRestoreCompletion(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	token string,
	timeout time.Duration,
) *restores_core.Restore {
	startTime := time.Now()
	pollInterval := 500 * time.Millisecond

	for {
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout waiting for MongoDB restore completion after %v", timeout)
		}

		var restoresList []*restores_core.Restore
		test_utils.MakeGetRequestAndUnmarshal(
			t,
			router,
			fmt.Sprintf("/api/v1/restores/%s", backupID.String()),
			"Bearer "+token,
			http.StatusOK,
			&restoresList,
		)

		for _, restore := range restoresList {
			if restore.Status == restores_core.RestoreStatusCompleted {
				return restore
			}
			if restore.Status == restores_core.RestoreStatusFailed {
				failMsg := "unknown error"
				if restore.FailMessage != nil {
					failMsg = *restore.FailMessage
				}
				t.Fatalf("MongoDB restore failed: %s", failMsg)
			}
		}

		time.Sleep(pollInterval)
	}
}

func verifyMongodbDataIntegrity(t *testing.T, container *MongodbContainer, restoredDBName string) {
	ctx := t.Context()

	originalCollection := container.Client.Database(container.Database).Collection("test_data")
	restoredCollection := container.Client.Database(restoredDBName).Collection("test_data")

	originalCount, err := originalCollection.CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)

	restoredCount, err := restoredCollection.CountDocuments(ctx, bson.M{})
	assert.NoError(t, err)

	assert.Equal(t, originalCount, restoredCount, "Should have same number of documents")

	var originalDocs []MongodbTestDataItem
	cursor, err := originalCollection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}),
	)
	assert.NoError(t, err)
	err = cursor.All(ctx, &originalDocs)
	assert.NoError(t, err)

	var restoredDocs []MongodbTestDataItem
	cursor, err = restoredCollection.Find(
		ctx,
		bson.M{},
		options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}),
	)
	assert.NoError(t, err)
	err = cursor.All(ctx, &restoredDocs)
	assert.NoError(t, err)

	assert.Equal(t, len(originalDocs), len(restoredDocs), "Should have same number of documents")

	for i := range originalDocs {
		assert.Equal(t, originalDocs[i].ID, restoredDocs[i].ID, "ID should match")
		assert.Equal(t, originalDocs[i].Name, restoredDocs[i].Name, "Name should match")
		assert.Equal(t, originalDocs[i].Value, restoredDocs[i].Value, "Value should match")
	}
}

func connectToMongodbContainer(
	t *testing.T,
	image string,
	version mongodb_engine.Version,
) *MongodbContainer {
	endpoint := containers.StartMongodb(t, image)

	return connectToMongodbEndpoint(t, endpoint, version)
}

func connectToMongodbEndpoint(
	t *testing.T,
	endpoint containers.Endpoint,
	version mongodb_engine.Version,
) *MongodbContainer {
	dbName := containers.MongodbDatabase
	password := containers.MongodbPassword
	username := containers.MongodbUsername
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		t.Fatalf("failed to connect to MongoDB %s: %v", version, err)
	}

	if err = client.Ping(ctx, nil); err != nil {
		t.Fatalf("failed to ping MongoDB %s: %v", version, err)
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

func setupMongodbTestData(t *testing.T, container *MongodbContainer) {
	ctx := t.Context()
	collection := container.Client.Database(container.Database).Collection("test_data")

	_ = collection.Drop(ctx)

	testDocs := []any{
		MongodbTestDataItem{
			ID:        "1",
			Name:      "test1",
			Value:     100,
			CreatedAt: time.Now().UTC(),
		},
		MongodbTestDataItem{
			ID:        "2",
			Name:      "test2",
			Value:     200,
			CreatedAt: time.Now().UTC(),
		},
		MongodbTestDataItem{
			ID:        "3",
			Name:      "test3",
			Value:     300,
			CreatedAt: time.Now().UTC(),
		},
	}

	_, err := collection.InsertMany(ctx, testDocs)
	assert.NoError(t, err)
}

func updateMongodbDatabaseCredentialsViaAPI(
	t *testing.T,
	router *gin.Engine,
	database apiclient.Database,
	username string,
	password string,
	token string,
) apiclient.Database {
	settings, err := database.Settings.AsMongodbSettings()
	require.NoError(t, err)

	settings.Username = username
	settings.Password = &password

	return logicaltesting.UpdateDatabaseSettingsViaAPI(t, router, database, settings, token)
}

func dropMongodbUserSafe(client *mongo.Client, username, authDatabase string) {
	ctx := context.Background()
	adminDB := client.Database(authDatabase)
	_ = adminDB.RunCommand(ctx, bson.D{{Key: "dropUser", Value: username}})
}
