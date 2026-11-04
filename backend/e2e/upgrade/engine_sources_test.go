package upgrade

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"databasus-backend/internal/lib/testwait"
)

// The MySQL, MariaDB and MongoDB servers hold the databases the previous release saves next to the
// PostgreSQL one. They are never backed up, so each holds only the empty object its exclusion list
// names.
const (
	engineSourceDatabaseName = "upgrade_source"
	engineSourcePassword     = "upgrade-engine-password"
	excludedObjectName       = "upgrade_excluded"

	mysqlSourceImage         = "mysql:8.4"
	mysqlSourceVersion       = "8.4"
	mysqlSourceNetworkAlias  = "upgrade-mysql"
	mysqlSourcePort          = 3306
	mysqlSourcePublishedPort = "3306/tcp"

	mariadbSourceImage        = "mariadb:10.11"
	mariadbSourceVersion      = "10.11"
	mariadbSourceNetworkAlias = "upgrade-mariadb"

	mongodbSourceImage         = "mongo:7.0"
	mongodbSourceVersion       = "7"
	mongodbSourceNetworkAlias  = "upgrade-mongodb"
	mongodbSourcePort          = 27017
	mongodbSourcePublishedPort = "27017/tcp"
	mongodbAuthDatabase        = "admin"

	engineSourceUsername = "root"
)

// prepareEngineSources starts the MySQL, MariaDB and MongoDB servers and creates the empty object
// each exclusion list names.
func prepareEngineSources(t *testing.T, docker *dockerHost, resources upgradeResources) error {
	t.Helper()

	err := startEngineSources(t.Context(), docker, resources)
	if err != nil {
		return err
	}

	for _, mysqlFamilySource := range []struct {
		containerName string
		label         string
	}{
		{resources.mysqlContainerName, "the MySQL server"},
		{resources.mariadbContainerName, "the MariaDB server"},
	} {
		err = prepareEngineSource(t, docker, engineSourceSpec{
			containerName: mysqlFamilySource.containerName,
			publishedPort: mysqlSourcePublishedPort,
			label:         mysqlFamilySource.label,
			createObject:  createExcludedMysqlTable,
		})
		if err != nil {
			return err
		}
	}

	return prepareEngineSource(t, docker, engineSourceSpec{
		containerName: resources.mongodbContainerName,
		publishedPort: mongodbSourcePublishedPort,
		label:         "the MongoDB server",
		createObject:  createExcludedMongodbCollection,
	})
}

func startEngineSources(ctx context.Context, docker *dockerHost, resources upgradeResources) error {
	for _, spec := range []containerSpec{
		{
			name:       resources.mysqlContainerName,
			image:      mysqlSourceImage,
			entrypoint: nil,
			env: []string{
				"MYSQL_ROOT_PASSWORD=" + engineSourcePassword,
				"MYSQL_DATABASE=" + engineSourceDatabaseName,
			},
			volumeMount:   "",
			networkName:   resources.networkName,
			networkAlias:  mysqlSourceNetworkAlias,
			publishedPort: mysqlSourcePublishedPort,
		},
		{
			name:       resources.mariadbContainerName,
			image:      mariadbSourceImage,
			entrypoint: nil,
			env: []string{
				"MARIADB_ROOT_PASSWORD=" + engineSourcePassword,
				"MARIADB_DATABASE=" + engineSourceDatabaseName,
			},
			volumeMount:   "",
			networkName:   resources.networkName,
			networkAlias:  mariadbSourceNetworkAlias,
			publishedPort: mysqlSourcePublishedPort,
		},
		{
			name:       resources.mongodbContainerName,
			image:      mongodbSourceImage,
			entrypoint: nil,
			env: []string{
				"MONGO_INITDB_ROOT_USERNAME=" + engineSourceUsername,
				"MONGO_INITDB_ROOT_PASSWORD=" + engineSourcePassword,
			},
			volumeMount:   "",
			networkName:   resources.networkName,
			networkAlias:  mongodbSourceNetworkAlias,
			publishedPort: mongodbSourcePublishedPort,
		},
	} {
		err := docker.startContainer(ctx, spec)
		if err != nil {
			return err
		}
	}

	return nil
}

type engineSourceSpec struct {
	containerName string
	publishedPort string
	label         string
	// createObject creates the empty object on the server at the published host port. It fails
	// while the server does not accept connections yet, so the wait retries it.
	createObject func(ctx context.Context, hostPort int) error
}

// prepareEngineSource waits until the server accepts connections from the host and creates its
// empty object. Each image restarts its server once after initializing it, and only the final
// server listens on the network, so the first object created survives.
func prepareEngineSource(t *testing.T, docker *dockerHost, spec engineSourceSpec) error {
	t.Helper()

	err := testwait.Within(startupLimit).Poll(t, spec.label+" to accept connections", func(ctx context.Context) error {
		state, stateErr := docker.getContainerState(ctx, spec.containerName, spec.publishedPort)
		if stateErr != nil {
			return stateErr
		}

		if !state.isRunning {
			return fmt.Errorf("container %s is not running", spec.containerName)
		}

		return spec.createObject(ctx, state.hostPort)
	})
	if err != nil {
		return docker.withLogTail(t.Context(), spec.containerName, err)
	}

	return nil
}

func createExcludedMysqlTable(ctx context.Context, hostPort int) error {
	connectionConfig := mysqldriver.NewConfig()
	connectionConfig.User = engineSourceUsername
	connectionConfig.Passwd = engineSourcePassword
	connectionConfig.Net = "tcp"
	connectionConfig.Addr = "127.0.0.1:" + strconv.Itoa(hostPort)
	connectionConfig.DBName = engineSourceDatabaseName

	connector, err := mysqldriver.NewConnector(connectionConfig)
	if err != nil {
		return fmt.Errorf("configure the MySQL client: %w", err)
	}

	database := sql.OpenDB(connector)
	defer func() { _ = database.Close() }()

	_, err = database.ExecContext(ctx,
		"CREATE TABLE IF NOT EXISTS "+excludedObjectName+" (id INT PRIMARY KEY)")
	if err != nil {
		return fmt.Errorf("create the excluded table: %w", err)
	}

	return nil
}

func createExcludedMongodbCollection(ctx context.Context, hostPort int) error {
	client, err := mongo.Connect(ctx, options.Client().
		ApplyURI(fmt.Sprintf("mongodb://127.0.0.1:%d/?directConnection=true", hostPort)).
		SetAuth(options.Credential{
			Username:   engineSourceUsername,
			Password:   engineSourcePassword,
			AuthSource: mongodbAuthDatabase,
		}))
	if err != nil {
		return fmt.Errorf("connect to the MongoDB server: %w", err)
	}
	defer func() { _ = client.Disconnect(ctx) }()

	database := client.Database(engineSourceDatabaseName)

	collectionNames, err := database.ListCollectionNames(ctx, map[string]any{"name": excludedObjectName})
	if err != nil {
		return fmt.Errorf("list the collections: %w", err)
	}

	if len(collectionNames) > 0 {
		return nil
	}

	err = database.CreateCollection(ctx, excludedObjectName)
	if err != nil {
		return fmt.Errorf("create the excluded collection: %w", err)
	}

	return nil
}
