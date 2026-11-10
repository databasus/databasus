package mongodb

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

const mongodbConnectTimeout = 10 * time.Second

func getMongodbVersions() []struct {
	name    string
	version apiclient.MongodbVersion
	image   string
} {
	return []struct {
		name    string
		version apiclient.MongodbVersion
		image   string
	}{
		{"MongoDB 4.2", apiclient.MongodbVersionN4, "mongo:4.2"},
		{"MongoDB 5.0", apiclient.MongodbVersionN5, "mongo:5.0"},
		{"MongoDB 6.0", apiclient.MongodbVersionN6, "mongo:6.0"},
		{"MongoDB 7.0", apiclient.MongodbVersionN7, "mongo:7.0"},
		{"MongoDB 8.2", apiclient.MongodbVersionN8, "mongo:8.2.3-noble"},
		{"MongoDB 9.0", apiclient.MongodbVersionN9, "mongo:9.0.2-noble"},
	}
}

type mongodbTestDocument struct {
	ID        string    `bson:"_id"`
	Name      string    `bson:"name"`
	Value     int       `bson:"value"`
	CreatedAt time.Time `bson:"created_at"`
}

// TestBackupAndRestoreMongodb boots each supported version once and runs every scenario against it,
// so only one server of the matrix is alive at a time.
//
//nolint:tparallel // Each version's scenarios share one server, and a version stops before the next starts.
func TestBackupAndRestoreMongodb(t *testing.T) {
	t.Parallel()

	for _, mongodbVersion := range getMongodbVersions() {
		t.Run(mongodbVersion.name, func(t *testing.T) {
			endpoint := containers.StartMongodb(t, mongodbVersion.image)

			t.Run("restores the documents", func(t *testing.T) {
				testMongodbDocumentsRestored(t, endpoint, mongodbVersion.version, apiclient.BackupEncryptionNONE)
			})
			t.Run("restores the documents of an encrypted backup", func(t *testing.T) {
				testMongodbDocumentsRestored(t, endpoint, mongodbVersion.version, apiclient.BackupEncryptionENCRYPTED)
			})
			t.Run("leaves out an excluded collection", func(t *testing.T) {
				testMongodbExcludedCollectionLeftOut(t, endpoint, mongodbVersion.version)
			})
			t.Run("restores the documents of a backup taken by a limited user", func(t *testing.T) {
				testMongodbLimitedUserBackupRestored(t, endpoint, mongodbVersion.version)
			})
		})
	}
}

func TestBackupAndRestoreMongodbOverTls(t *testing.T) {
	t.Parallel()

	t.Run("restores the documents over a connection that requires TLS", func(t *testing.T) {
		t.Parallel()

		certificateDirectory := enginetest.GetTestdataDirectory(t, "ssl")
		endpoint := containers.StartMongodbSSL(t,
			filepath.Join(certificateDirectory, "server.pem"), filepath.Join(certificateDirectory, "server.crt"))
		server := connectToMongodb(t, endpoint, "&tls=true&tlsInsecure=true")
		server.createTestDataCollection(t)

		settings := server.newSettings(apiclient.MongodbVersionN8)
		settings.IsHttps = true

		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName := server.newRestoreDatabaseName(t, "restored_mongo_ssl")
		target := server.newTarget(restoredDatabaseName)
		target.IsHttps = true
		tm.Restore(t, backup.Id, toMongodbRestoreTarget(t, target))

		server.verifyRestoredDocuments(t, restoredDatabaseName)
	})
}

func TestBackupAndRestoreMongodbOverSshTunnel(t *testing.T) {
	t.Parallel()

	t.Run("restores the documents through the bastion", func(t *testing.T) {
		t.Parallel()

		topology := containers.StartMongodbBehindSSHBastion(t, "mongo:7.0")
		server := connectToMongodb(t, enginetest.OpenForwardedEndpoint(t, topology), "")
		server.createTestDataCollection(t)

		bastionServer := mongodbServer{endpoint: topology.Database, client: nil}
		settings := bastionServer.newSettings(apiclient.MongodbVersionN7)
		settings.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)

		tm := enginetest.NewTeam(t)
		backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

		restoredDatabaseName := server.newRestoreDatabaseName(t, "restored_mongodb_ssh_tunnel")
		target := bastionServer.newTarget(restoredDatabaseName)
		target.SshTunnel = enginetest.NewBastionSSHTunnelSettings(topology)
		tm.Restore(t, backup.Id, toMongodbRestoreTarget(t, target))

		server.verifyRestoredDocuments(t, restoredDatabaseName)
	})
}

func TestBackupMongodbToFailingStorage(t *testing.T) {
	t.Parallel()

	t.Run("fails the backup without hanging when the upload fails", func(t *testing.T) {
		t.Parallel()

		server := connectToMongodb(t, containers.StartMongodb(t, "mongo:7.0"), "")
		server.createTestDataCollection(t)

		tm := enginetest.NewTeam(t)
		database := tm.CreateDatabase(t, server.newSettings(apiclient.MongodbVersionN7))

		tm.RequireBackupFailsWithoutHanging(t, database.Id)
	})
}

func TestStartMongodbRestoreVersionCheck(t *testing.T) {
	t.Parallel()

	server := connectToMongodb(t, containers.StartMongodb(t, "mongo:7.0"), "")
	server.createTestDataCollection(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, server.newSettings(apiclient.MongodbVersionN7))
	backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
	restoredDatabaseName := server.newRestoreDatabaseName(t, "restored_mongodb_version")

	tm.RequireRestoreVersionCheck(t, enginetest.RestoreVersionCheck{
		DatabaseID:    database.Id,
		BackupID:      backup.Id,
		SettingsTable: "databases_mongodb",
		Target:        toMongodbRestoreTarget(t, server.newTarget(restoredDatabaseName)),
		TargetVersion: "7",
	})
}

func testMongodbDocumentsRestored(
	t *testing.T,
	endpoint containers.Endpoint,
	version apiclient.MongodbVersion,
	encryption apiclient.BackupEncryption,
) {
	t.Helper()

	server := connectToMongodb(t, endpoint, "")
	server.createTestDataCollection(t)

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, server.newSettings(version)).Id, encryption)

	assert.Equal(t, encryption, backup.Encryption)
	tm.RequireBackupSizeMatchesDownload(t, backup)

	restoredDatabaseName := server.newRestoreDatabaseName(t, "restored_mongodb")
	tm.Restore(t, backup.Id, toMongodbRestoreTarget(t, server.newTarget(restoredDatabaseName)))

	server.verifyRestoredDocuments(t, restoredDatabaseName)
}

func testMongodbExcludedCollectionLeftOut(
	t *testing.T,
	endpoint containers.Endpoint,
	version apiclient.MongodbVersion,
) {
	t.Helper()

	server := connectToMongodb(t, endpoint, "")
	server.createTestDataCollection(t)

	skippedCollection := server.client.Database(containers.MongodbDatabase).Collection("skip_me")
	_ = skippedCollection.Drop(t.Context())
	_, err := skippedCollection.InsertMany(t.Context(), []any{
		mongodbTestDocument{ID: "1", Name: "skip1", Value: 1, CreatedAt: time.Now().UTC()},
		mongodbTestDocument{ID: "2", Name: "skip2", Value: 2, CreatedAt: time.Now().UTC()},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = skippedCollection.Drop(context.WithoutCancel(t.Context())) })

	settings := server.newSettings(version)
	settings.ExcludeCollections = []string{"skip_me"}

	tm := enginetest.NewTeam(t)
	backup := tm.BackUp(t, tm.CreateDatabase(t, settings).Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName := server.newRestoreDatabaseName(t, "restored_mongodb_excl")
	tm.Restore(t, backup.Id, toMongodbRestoreTarget(t, server.newTarget(restoredDatabaseName)))

	restoredDatabase := server.client.Database(restoredDatabaseName)

	keptCount, err := restoredDatabase.Collection("test_data").CountDocuments(t.Context(), bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(3), keptCount, "test_data is restored with its 3 documents")

	skippedCount, err := restoredDatabase.Collection("skip_me").CountDocuments(t.Context(), bson.M{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), skippedCount, "the excluded skip_me collection is left out")
}

func testMongodbLimitedUserBackupRestored(
	t *testing.T,
	endpoint containers.Endpoint,
	version apiclient.MongodbVersion,
) {
	t.Helper()

	server := connectToMongodb(t, endpoint, "")
	server.createTestDataCollection(t)

	tm := enginetest.NewTeam(t)
	database := tm.CreateDatabase(t, server.newSettings(version))

	limitedUser := tm.CreateLimitedUser(t, database)
	assert.NotEmpty(t, limitedUser.Username)
	assert.NotEmpty(t, limitedUser.Password)

	adminDatabase := server.client.Database(containers.MongodbAuthDatabase)
	t.Cleanup(func() {
		_ = adminDatabase.RunCommand(context.WithoutCancel(t.Context()),
			bson.D{{Key: "dropUser", Value: limitedUser.Username}}).Err()
	})

	settings, err := database.Settings.AsMongodbSettings()
	require.NoError(t, err)

	settings.Username = limitedUser.Username
	settings.Password = &limitedUser.Password

	limitedDatabase := tm.UpdateDatabaseSettings(t, database, settings)
	backup := tm.BackUp(t, limitedDatabase.Id, apiclient.BackupEncryptionNONE)

	restoredDatabaseName := server.newRestoreDatabaseName(t, "restored_mongodb_ro")
	tm.Restore(t, backup.Id, toMongodbRestoreTarget(t, server.newTarget(restoredDatabaseName)))

	server.verifyRestoredDocuments(t, restoredDatabaseName)
}

// mongodbServer is a MongoDB server of a test, reached with the container's root user.
type mongodbServer struct {
	endpoint containers.Endpoint
	client   *mongo.Client
}

// connectToMongodb connects with the URI options appended to the defaults, such as the TLS ones.
func connectToMongodb(t *testing.T, endpoint containers.Endpoint, uriOptions string) mongodbServer {
	t.Helper()

	uri := fmt.Sprintf(
		"mongodb://%s:%s@%s:%d/%s?authSource=%s&serverSelectionTimeoutMS=5000&connectTimeoutMS=5000%s",
		containers.MongodbUsername, containers.MongodbPassword, endpoint.Host, endpoint.Port,
		containers.MongodbDatabase, containers.MongodbAuthDatabase, uriOptions,
	)

	connectCtx, cancelConnect := context.WithTimeout(t.Context(), mongodbConnectTimeout)
	defer cancelConnect()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(context.WithoutCancel(t.Context())) })

	require.NoError(t, client.Ping(connectCtx, nil))

	return mongodbServer{endpoint: endpoint, client: client}
}

// createTestDataCollection fills test_data with three documents, replacing the ones an earlier
// scenario left.
func (s mongodbServer) createTestDataCollection(t *testing.T) {
	t.Helper()

	collection := s.client.Database(containers.MongodbDatabase).Collection("test_data")
	_ = collection.Drop(t.Context())

	_, err := collection.InsertMany(t.Context(), []any{
		mongodbTestDocument{ID: "1", Name: "test1", Value: 100, CreatedAt: time.Now().UTC()},
		mongodbTestDocument{ID: "2", Name: "test2", Value: 200, CreatedAt: time.Now().UTC()},
		mongodbTestDocument{ID: "3", Name: "test3", Value: 300, CreatedAt: time.Now().UTC()},
	})
	require.NoError(t, err)
}

// newRestoreDatabaseName names a database the restore creates and drops it when the test ends.
func (s mongodbServer) newRestoreDatabaseName(t *testing.T, prefix string) string {
	t.Helper()

	databaseName := prefix + "_" + enginetest.NewUniqueSuffix()
	t.Cleanup(func() { _ = s.client.Database(databaseName).Drop(context.WithoutCancel(t.Context())) })

	return databaseName
}

func (s mongodbServer) verifyRestoredDocuments(t *testing.T, restoredDatabaseName string) {
	t.Helper()

	sourceCollection := s.client.Database(containers.MongodbDatabase).Collection("test_data")
	restoredCollection := s.client.Database(restoredDatabaseName).Collection("test_data")

	sourceCount, err := sourceCollection.CountDocuments(t.Context(), bson.M{})
	require.NoError(t, err)

	restoredCount, err := restoredCollection.CountDocuments(t.Context(), bson.M{})
	require.NoError(t, err)
	assert.Equal(t, sourceCount, restoredCount, "the restored collection has as many documents as the source")

	sourceDocuments := findMongodbDocumentsByID(t, sourceCollection)
	restoredDocuments := findMongodbDocumentsByID(t, restoredCollection)

	require.Len(
		t,
		restoredDocuments,
		len(sourceDocuments),
		"the restored collection has as many documents as the source",
	)

	for documentIndex := range sourceDocuments {
		assert.Equal(t, sourceDocuments[documentIndex].ID, restoredDocuments[documentIndex].ID, "_id")
		assert.Equal(t, sourceDocuments[documentIndex].Name, restoredDocuments[documentIndex].Name, "name")
		assert.Equal(t, sourceDocuments[documentIndex].Value, restoredDocuments[documentIndex].Value, "value")
	}
}

func (s mongodbServer) newSettings(version apiclient.MongodbVersion) apiclient.MongodbSettings {
	port := s.endpoint.Port
	password := containers.MongodbPassword

	return apiclient.MongodbSettings{
		Type:               apiclient.MongodbSettingsTypeMONGODB,
		Host:               s.endpoint.Host,
		Port:               &port,
		Username:           containers.MongodbUsername,
		Password:           &password,
		Database:           containers.MongodbDatabase,
		AuthDatabase:       containers.MongodbAuthDatabase,
		Version:            &version,
		IsHttps:            false,
		IsSrv:              false,
		IsDirectConnection: false,
		CpuCount:           1,
		ExcludeCollections: []string{},
		SshTunnel:          enginetest.NewDisabledSSHTunnelSettings(),
	}
}

func (s mongodbServer) newTarget(databaseName string) apiclient.MongodbRestoreTarget {
	port := s.endpoint.Port
	password := containers.MongodbPassword

	return apiclient.MongodbRestoreTarget{
		Type:               apiclient.MongodbRestoreTargetTypeMONGODB,
		Host:               s.endpoint.Host,
		Port:               &port,
		Username:           containers.MongodbUsername,
		Password:           &password,
		Database:           databaseName,
		AuthDatabase:       containers.MongodbAuthDatabase,
		IsHttps:            false,
		IsSrv:              false,
		IsDirectConnection: false,
		CpuCount:           1,
		SshTunnel:          enginetest.NewDisabledSSHTunnelSettings(),
	}
}

func toMongodbRestoreTarget(t *testing.T, target apiclient.MongodbRestoreTarget) apiclient.RestoreTarget {
	t.Helper()

	var restoreTarget apiclient.RestoreTarget
	require.NoError(t, restoreTarget.FromMongodbRestoreTarget(target))

	return restoreTarget
}

func findMongodbDocumentsByID(t *testing.T, collection *mongo.Collection) []mongodbTestDocument {
	t.Helper()

	cursor, err := collection.Find(t.Context(), bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	require.NoError(t, err)

	var documents []mongodbTestDocument
	require.NoError(t, cursor.All(t.Context(), &documents))

	return documents
}
