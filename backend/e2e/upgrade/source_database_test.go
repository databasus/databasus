package upgrade

import (
	"context"
	"fmt"
	"slices"

	"github.com/jackc/pgx/v5"
)

// The source server holds the database the previous release backs up, and later the database the
// current code restores that backup into.
const (
	sourceImage         = "postgres:17"
	sourceVersion       = "17"
	sourceNetworkAlias  = "upgrade-source"
	sourcePort          = 5432
	sourcePublishedPort = "5432/tcp"
	sourceUsername      = "postgres"
	sourcePassword      = "upgrade-source-password"
	sourceDatabaseName  = "upgrade_source"
	restoredDatabase    = "upgrade_restored"
	sourceSchemaName    = "public"
	// The table and table data exclusion lists name these tables, which stay empty, so the lists
	// never change the rows the restore compares.
	excludedTableName     = "upgrade_excluded"
	excludedDataTableName = "upgrade_excluded_data"
)

func getSourceRows() []string {
	return []string{"first row before the upgrade", "second row before the upgrade"}
}

const rowAddedAfterBackup = "row added after the backup"

func startSourceServer(ctx context.Context, docker *dockerHost, resources upgradeResources) error {
	return docker.startContainer(ctx, containerSpec{
		name:  resources.sourceContainerName,
		image: sourceImage,
		entrypoint: []string{
			"sh",
			"-ec",
			`printf '%s\n' '#!/bin/sh' 'echo "host replication all all scram-sha-256" >> "$PGDATA/pg_hba.conf"' > /docker-entrypoint-initdb.d/replication.sh
exec docker-entrypoint.sh postgres -c summarize_wal=on`,
		},
		env: []string{
			"POSTGRES_USER=" + sourceUsername,
			"POSTGRES_PASSWORD=" + sourcePassword,
			"POSTGRES_DB=" + sourceDatabaseName,
		},
		volumeMount:   "",
		networkName:   resources.networkName,
		networkAlias:  sourceNetworkAlias,
		publishedPort: sourcePublishedPort,
	})
}

func connectToSourceDatabase(ctx context.Context, hostPort int, databaseName string) (*pgx.Conn, error) {
	connection, err := pgx.Connect(ctx, fmt.Sprintf(
		"host=127.0.0.1 port=%d user=%s password=%s dbname=%s sslmode=disable",
		hostPort, sourceUsername, sourcePassword, databaseName,
	))
	if err != nil {
		return nil, fmt.Errorf("connect to database %s on the source server: %w", databaseName, err)
	}

	return connection, nil
}

func fillSourceDatabase(ctx context.Context, hostPort int) error {
	return executeOnSourceDatabase(ctx, hostPort, sourceDatabaseName, func(connection *pgx.Conn) error {
		_, err := connection.Exec(ctx, "CREATE TABLE upgrade_rows (id SERIAL PRIMARY KEY, label TEXT NOT NULL)")
		if err != nil {
			return fmt.Errorf("create the source table: %w", err)
		}

		for _, excludedTable := range []string{excludedTableName, excludedDataTableName} {
			_, err = connection.Exec(ctx, "CREATE TABLE "+excludedTable+" (id SERIAL PRIMARY KEY)")
			if err != nil {
				return fmt.Errorf("create the excluded table %s: %w", excludedTable, err)
			}
		}

		for _, label := range getSourceRows() {
			_, err = connection.Exec(ctx, "INSERT INTO upgrade_rows (label) VALUES ($1)", label)
			if err != nil {
				return fmt.Errorf("insert a source row: %w", err)
			}
		}

		return nil
	})
}

// addRowAfterBackup changes the source after the backup, so a restore that read the live source
// instead of the backup would not match the rows at backup time.
func addRowAfterBackup(ctx context.Context, hostPort int) error {
	return executeOnSourceDatabase(ctx, hostPort, sourceDatabaseName, func(connection *pgx.Conn) error {
		_, err := connection.Exec(ctx, "INSERT INTO upgrade_rows (label) VALUES ($1)", rowAddedAfterBackup)
		if err != nil {
			return fmt.Errorf("insert a source row after the backup: %w", err)
		}

		return nil
	})
}

func createRestoredDatabase(ctx context.Context, hostPort int) error {
	return executeOnSourceDatabase(ctx, hostPort, sourceDatabaseName, func(connection *pgx.Conn) error {
		_, err := connection.Exec(ctx, "CREATE DATABASE "+restoredDatabase)
		if err != nil {
			return fmt.Errorf("create the database to restore into: %w", err)
		}

		return nil
	})
}

func checkRestoredRows(ctx context.Context, hostPort int) error {
	return executeOnSourceDatabase(ctx, hostPort, restoredDatabase, func(connection *pgx.Conn) error {
		rows, err := connection.Query(ctx, "SELECT label FROM upgrade_rows ORDER BY id")
		if err != nil {
			return fmt.Errorf("read the restored rows: %w", err)
		}

		restoredRows, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return fmt.Errorf("read the restored rows: %w", err)
		}

		if !slices.Equal(restoredRows, getSourceRows()) {
			return fmt.Errorf("the restored database holds rows %q, the source had %q at backup time",
				restoredRows, getSourceRows())
		}

		return nil
	})
}

func executeOnSourceDatabase(
	ctx context.Context,
	hostPort int,
	databaseName string,
	work func(connection *pgx.Conn) error,
) error {
	connection, err := connectToSourceDatabase(ctx, hostPort, databaseName)
	if err != nil {
		return err
	}
	defer func() { _ = connection.Close(ctx) }()

	return work(connection)
}
