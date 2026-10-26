package upgrade

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the database API of the previous release, like previous_release_client_test.go.

const (
	mysqlDatabaseName   = "upgrade mysql database"
	mariadbDatabaseName = "upgrade mariadb database"
	mongodbDatabaseName = "upgrade mongodb database"
)

// previousDatabaseRequest carries the settings of one database type, in the field the previous
// release names after that type.
type previousDatabaseRequest struct {
	WorkspaceID       string                        `json:"workspaceId"`
	Name              string                        `json:"name"`
	Type              string                        `json:"type"`
	PostgresqlLogical *previousPostgresqlConnection `json:"postgresqlLogical,omitempty"`
	Mysql             *previousMysqlConnection      `json:"mysql,omitempty"`
	Mariadb           *previousMariadbConnection    `json:"mariadb,omitempty"`
	Mongodb           *previousMongodbConnection    `json:"mongodb,omitempty"`
	Notifiers         []previousNotifierReference   `json:"notifiers"`
}

type previousPostgresqlConnection struct {
	Version          string   `json:"version"`
	Host             string   `json:"host"`
	Port             int      `json:"port"`
	Username         string   `json:"username"`
	Password         string   `json:"password"`
	Database         string   `json:"database"`
	SslMode          string   `json:"sslMode"`
	IncludeSchemas   []string `json:"includeSchemas"`
	ExcludeTables    []string `json:"excludeTables"`
	ExcludeTableData []string `json:"excludeTableData"`
	CPUCount         int      `json:"cpuCount"`
}

type previousMysqlConnection struct {
	Version       string   `json:"version"`
	Host          string   `json:"host"`
	Port          int      `json:"port"`
	Username      string   `json:"username"`
	Password      string   `json:"password"`
	Database      string   `json:"database"`
	IsHTTPS       bool     `json:"isHttps"`
	ExcludeTables []string `json:"excludeTables"`
}

type previousMariadbConnection struct {
	Version             string   `json:"version"`
	Host                string   `json:"host"`
	Port                int      `json:"port"`
	Username            string   `json:"username"`
	Password            string   `json:"password"`
	Database            string   `json:"database"`
	IsHTTPS             bool     `json:"isHttps"`
	IsExcludeEvents     bool     `json:"isExcludeEvents"`
	IsSkipGaleraDisable bool     `json:"isSkipGaleraDisable"`
	ExcludeTables       []string `json:"excludeTables"`
}

type previousMongodbConnection struct {
	Version            string   `json:"version"`
	Host               string   `json:"host"`
	Port               int      `json:"port"`
	Username           string   `json:"username"`
	Password           string   `json:"password"`
	Database           string   `json:"database"`
	AuthDatabase       string   `json:"authDatabase"`
	IsHTTPS            bool     `json:"isHttps"`
	IsSrv              bool     `json:"isSrv"`
	IsDirectConnection bool     `json:"isDirectConnection"`
	CPUCount           int      `json:"cpuCount"`
	ExcludeCollections []string `json:"excludeCollections"`
}

// writtenDatabases are the databases the previous release saved, keyed by name.
type writtenDatabases struct {
	items []upgradecheck.Item
	ids   map[string]string
}

// createDatabases saves a PostgreSQL database that uses every notifier and a MySQL, a MariaDB and a
// MongoDB database. Each has a password and exclusion lists that name only objects without rows.
func createDatabases(
	ctx context.Context,
	api *apiClient,
	workspaceID string,
	notifierReferences []previousNotifierReference,
) (writtenDatabases, error) {
	databases := writtenDatabases{items: nil, ids: map[string]string{}}

	postgresqlConnection := previousPostgresqlConnection{
		Version:          sourceVersion,
		Host:             sourceNetworkAlias,
		Port:             sourcePort,
		Username:         sourceUsername,
		Password:         sourcePassword,
		Database:         sourceDatabaseName,
		SslMode:          "disable",
		IncludeSchemas:   []string{sourceSchemaName},
		ExcludeTables:    []string{sourceSchemaName + "." + excludedTableName},
		ExcludeTableData: []string{sourceSchemaName + "." + excludedDataTableName},
		CPUCount:         1,
	}
	mysqlConnection := previousMysqlConnection{
		Version:       mysqlSourceVersion,
		Host:          mysqlSourceNetworkAlias,
		Port:          mysqlSourcePort,
		Username:      engineSourceUsername,
		Password:      engineSourcePassword,
		Database:      engineSourceDatabaseName,
		IsHTTPS:       false,
		ExcludeTables: []string{excludedObjectName},
	}
	mariadbConnection := previousMariadbConnection{
		Version:             mariadbSourceVersion,
		Host:                mariadbSourceNetworkAlias,
		Port:                mysqlSourcePort,
		Username:            engineSourceUsername,
		Password:            engineSourcePassword,
		Database:            engineSourceDatabaseName,
		IsHTTPS:             false,
		IsExcludeEvents:     false,
		IsSkipGaleraDisable: false,
		ExcludeTables:       []string{excludedObjectName},
	}
	mongodbConnection := previousMongodbConnection{
		Version:            mongodbSourceVersion,
		Host:               mongodbSourceNetworkAlias,
		Port:               mongodbSourcePort,
		Username:           engineSourceUsername,
		Password:           engineSourcePassword,
		Database:           engineSourceDatabaseName,
		AuthDatabase:       mongodbAuthDatabase,
		IsHTTPS:            false,
		IsSrv:              false,
		IsDirectConnection: false,
		CPUCount:           1,
		ExcludeCollections: []string{excludedObjectName},
	}

	for _, database := range []struct {
		request previousDatabaseRequest
		values  map[string]string
	}{
		{
			request: previousDatabaseRequest{
				WorkspaceID: workspaceID, Name: databaseName, Type: "POSTGRES_LOGICAL",
				PostgresqlLogical: &postgresqlConnection, Mysql: nil, Mariadb: nil, Mongodb: nil,
				Notifiers: notifierReferences,
			},
			values: map[string]string{
				"type":                "POSTGRES_LOGICAL",
				"version":             postgresqlConnection.Version,
				"host":                postgresqlConnection.Host,
				"port":                strconv.Itoa(postgresqlConnection.Port),
				"username":            postgresqlConnection.Username,
				"database":            postgresqlConnection.Database,
				"sslMode":             postgresqlConnection.SslMode,
				"includeSchemas":      strings.Join(postgresqlConnection.IncludeSchemas, ","),
				"excludeTables":       strings.Join(postgresqlConnection.ExcludeTables, ","),
				"excludeTableData":    strings.Join(postgresqlConnection.ExcludeTableData, ","),
				"cpuCount":            strconv.Itoa(postgresqlConnection.CPUCount),
				"isSkipUserMappings":  "false",
				"sshTunnel.isEnabled": "false",
			},
		},
		{
			request: previousDatabaseRequest{
				WorkspaceID: workspaceID, Name: mysqlDatabaseName, Type: "MYSQL",
				PostgresqlLogical: nil, Mysql: &mysqlConnection, Mariadb: nil, Mongodb: nil,
				Notifiers: []previousNotifierReference{},
			},
			values: map[string]string{
				"type":                "MYSQL",
				"version":             mysqlConnection.Version,
				"host":                mysqlConnection.Host,
				"port":                strconv.Itoa(mysqlConnection.Port),
				"username":            mysqlConnection.Username,
				"database":            mysqlConnection.Database,
				"isHttps":             strconv.FormatBool(mysqlConnection.IsHTTPS),
				"excludeTables":       strings.Join(mysqlConnection.ExcludeTables, ","),
				"sshTunnel.isEnabled": "false",
			},
		},
		{
			request: previousDatabaseRequest{
				WorkspaceID: workspaceID, Name: mariadbDatabaseName, Type: "MARIADB",
				PostgresqlLogical: nil, Mysql: nil, Mariadb: &mariadbConnection, Mongodb: nil,
				Notifiers: []previousNotifierReference{},
			},
			values: map[string]string{
				"type":                "MARIADB",
				"version":             mariadbConnection.Version,
				"host":                mariadbConnection.Host,
				"port":                strconv.Itoa(mariadbConnection.Port),
				"username":            mariadbConnection.Username,
				"database":            mariadbConnection.Database,
				"isHttps":             strconv.FormatBool(mariadbConnection.IsHTTPS),
				"isExcludeEvents":     strconv.FormatBool(mariadbConnection.IsExcludeEvents),
				"isSkipGaleraDisable": strconv.FormatBool(mariadbConnection.IsSkipGaleraDisable),
				"excludeTables":       strings.Join(mariadbConnection.ExcludeTables, ","),
				"sshTunnel.isEnabled": "false",
			},
		},
		{
			request: previousDatabaseRequest{
				WorkspaceID: workspaceID, Name: mongodbDatabaseName, Type: "MONGODB",
				PostgresqlLogical: nil, Mysql: nil, Mariadb: nil, Mongodb: &mongodbConnection,
				Notifiers: []previousNotifierReference{},
			},
			values: map[string]string{
				"type":                "MONGODB",
				"version":             mongodbConnection.Version,
				"host":                mongodbConnection.Host,
				"port":                strconv.Itoa(mongodbConnection.Port),
				"username":            mongodbConnection.Username,
				"database":            mongodbConnection.Database,
				"authDatabase":        mongodbConnection.AuthDatabase,
				"isHttps":             strconv.FormatBool(mongodbConnection.IsHTTPS),
				"isSrv":               strconv.FormatBool(mongodbConnection.IsSrv),
				"isDirectConnection":  strconv.FormatBool(mongodbConnection.IsDirectConnection),
				"cpuCount":            strconv.Itoa(mongodbConnection.CPUCount),
				"excludeCollections":  strings.Join(mongodbConnection.ExcludeCollections, ","),
				"sshTunnel.isEnabled": "false",
			},
		},
	} {
		var createdDatabase previousCreatedItem

		err := api.post(ctx, "/databases/create", database.request, &createdDatabase)
		if err != nil {
			return writtenDatabases{}, fmt.Errorf("create database %q: %w", database.request.Name, err)
		}

		database.values["id"] = createdDatabase.ID
		databases.ids[database.request.Name] = createdDatabase.ID
		databases.items = append(databases.items, upgradecheck.Item{
			Kind: "database", Name: database.request.Name, Values: database.values,
		})
	}

	return databases, nil
}
