package postgresql

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

type blockedTimescaleRestore struct {
	server             postgresqlServer
	team               enginetest.Team
	targetDatabaseName string
	archiveLock        *sqlx.Tx
	proxy              *restoreProxy
	process            *timescaleServerProcess
	client             *apiclient.ClientWithResponses
	restoreID          uuid.UUID
}

func startBlockedTimescaleRestore(t *testing.T, endpoint containers.Endpoint) blockedTimescaleRestore {
	t.Helper()
	server := connectToPostgresql(t, endpoint, postgresqlPlainConnection)
	sourceName, sourceDB := server.createRestoreDatabase(t, "cleanup_source")
	_, err := sourceDB.ExecContext(
		t.Context(),
		"CREATE TABLE cleanup_rows(id integer); INSERT INTO cleanup_rows VALUES (1)",
	)
	require.NoError(t, err)

	tm := enginetest.NewTeam(t)
	settings := server.newSettings()
	settings.Database = &sourceName
	database := tm.CreateDatabase(t, settings)
	backup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)

	targetName, targetDB := server.createRestoreDatabase(t, "cleanup_target")
	_, err = targetDB.ExecContext(t.Context(), "CREATE TABLE cleanup_rows(id integer)")
	require.NoError(t, err)
	lock, err := targetDB.BeginTxx(t.Context(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = lock.Rollback() })
	_, err = lock.ExecContext(t.Context(), "LOCK TABLE cleanup_rows IN ACCESS EXCLUSIVE MODE")
	require.NoError(t, err)

	proxy := startRestoreProxy(t, endpoint)
	process := startTimescaleServer(t, tm.App.GetServerConfig())
	client := apiclient.New(t, process.URL, apiclient.WithSessionToken(tm.Owner.Token))

	target := server.newTarget(targetName)
	target.Host = proxy.endpoint.Host
	target.Port = proxy.endpoint.Port
	response, err := client.StartRestoreWithResponse(
		t.Context(),
		apiclient.RestoreStart{BackupId: backup.Id, Target: toPostgresqlRestoreTarget(t, target)},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))
	restoreID := response.JSON202.Id

	testwait.Until(t, "the archive waits for the target table lock", func(ctx context.Context) error {
		var isWaiting bool
		err := server.admin.GetContext(
			ctx,
			&isWaiting,
			"SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname = $1 AND wait_event_type = 'Lock' AND query LIKE '%cleanup_rows%')",
			targetName,
		)
		if err != nil {
			return err
		}
		if !isWaiting {
			return fmt.Errorf("archive is not waiting for the table lock")
		}
		return nil
	})
	require.True(t, isTimescaleTargetRestoring(t, server, targetName))

	return blockedTimescaleRestore{
		server:             server,
		team:               tm,
		targetDatabaseName: targetName,
		archiveLock:        lock,
		proxy:              proxy,
		process:            process,
		client:             client,
		restoreID:          restoreID,
	}
}
