package mongodb

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	mongodb_engine "databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

// The engine tests in internal/engines/mongodb cover the behavior; this one shows the model still
// reaches its engine and writes the detected version into itself.
func Test_TestConnection_WhenUserIsRoot_PopulatesVersion(t *testing.T) {
	endpoint := containers.StartMongodb(t, "mongo:7.0")

	mongodbModel := &MongodbDatabase{
		Host:         endpoint.Host,
		Port:         new(endpoint.Port),
		Username:     containers.MongodbUsername,
		Password:     containers.MongodbPassword,
		Database:     containers.MongodbDatabase,
		AuthDatabase: containers.MongodbAuthDatabase,
		IsHttps:      false,
		IsSrv:        false,
		CpuCount:     1,
	}

	err := mongodbModel.TestConnection(t.Context(), slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)

	assert.NoError(t, err)
	assert.Equal(t, mongodb_engine.Version7, mongodbModel.Version)
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	port := 27017
	mongodbModel := &MongodbDatabase{
		Version:            mongodb_engine.Version7,
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

	assert.Contains(t, uri, "localhost:27017")
	assert.Contains(t, uri, "/?authSource=admin")
	assert.NotContains(t, uri, "/mydb")
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
