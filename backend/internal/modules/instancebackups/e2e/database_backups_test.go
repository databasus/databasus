package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

const (
	logicalRowsTable    = "recovery_rows"
	physicalMarkerTable = "recovery_markers"
	// backupLimit bounds a database backup or restore of the small source database, which the
	// containers on a loaded machine finish within minutes.
	backupLimit = 5 * time.Minute
)

var (
	errBackupRunning  = errors.New("the backup is still running")
	errRestoreRunning = errors.New("the restore is still running")
)

func getLogicalRows() []string {
	return []string{"row backed up before the loss", "second row backed up before the loss"}
}

func createLogicalRows(t *testing.T, connection *pgx.Conn) {
	t.Helper()

	_, err := connection.Exec(t.Context(),
		"CREATE TABLE "+logicalRowsTable+" (id SERIAL PRIMARY KEY, label TEXT NOT NULL)")
	require.NoError(t, err)

	for _, label := range getLogicalRows() {
		_, err = connection.Exec(t.Context(), "INSERT INTO "+logicalRowsTable+" (label) VALUES ($1)", label)
		require.NoError(t, err)
	}
}

func readLogicalRows(t *testing.T, connection *pgx.Conn) []string {
	t.Helper()

	rows, err := connection.Query(t.Context(), "SELECT label FROM "+logicalRowsTable+" ORDER BY id")
	require.NoError(t, err)

	labels, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)

	return labels
}

func newSourceDatabaseSettings(t *testing.T) apiclient.DatabaseSettings {
	t.Helper()

	return enginetest.NewDatabaseSettings(t, apiclient.PostgresqlLogicalSettings{
		Host:               sourceServerAlias,
		Port:               sourceServerPort,
		Username:           sourceUsername,
		Password:           new(sourcePassword),
		Database:           new(sourceDatabaseName),
		SslMode:            apiclient.PostgresqlSslModeDisable,
		SshTunnel:          apiclient.SshTunnelSettings{IsEnabled: false},
		CpuCount:           1,
		ExcludeTableData:   []string{},
		ExcludeTables:      []string{},
		IncludeSchemas:     []string{},
		IsSkipUserMappings: false,
	})
}

func newPhysicalSourceSettings(t *testing.T) apiclient.DatabaseSettings {
	t.Helper()

	return enginetest.NewDatabaseSettings(t, apiclient.PostgresqlPhysicalSettings{
		Host:       sourceServerAlias,
		Port:       sourceServerPort,
		Username:   sourceUsername,
		Password:   new(sourcePassword),
		SslMode:    apiclient.PostgresqlSslModeDisable,
		SshTunnel:  apiclient.SshTunnelSettings{IsEnabled: false},
		BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	})
}

func createDatabase(
	t *testing.T,
	owner accountSession,
	workspaceID uuid.UUID,
	settings apiclient.DatabaseSettings,
	notifierIDs []uuid.UUID,
) apiclient.Database {
	t.Helper()

	response, err := owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
		WorkspaceId: workspaceID,
		Name:        "Database " + uuid.NewString()[:8],
		NotifierIds: notifierIDs,
		Settings:    settings,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

// enableLogicalBackups keeps the default schedule and retention of the database's config.
func enableLogicalBackups(t *testing.T, owner accountSession, databaseID, storageID uuid.UUID) {
	t.Helper()

	readResponse, err := owner.Client.GetBackupConfigWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, readResponse.StatusCode(), string(readResponse.Body))

	response, err := owner.Client.UpdateBackupConfigWithResponse(t.Context(), databaseID, apiclient.BackupConfigInput{
		IsBackupsEnabled: true,
		StorageId:        &storageID,
		Encryption:       apiclient.BackupEncryptionENCRYPTED,
		Settings:         readResponse.JSON200.Settings,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func backUpLogically(t *testing.T, owner accountSession, databaseID uuid.UUID) apiclient.LogicalBackup {
	t.Helper()

	response, err := owner.Client.StartLogicalBackupWithResponse(t.Context(),
		apiclient.LogicalBackupStart{DatabaseId: databaseID})
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))

	var finishedBackup apiclient.LogicalBackup

	testwait.Within(backupLimit).Until(t, "the logical backup finishes", func(ctx context.Context) error {
		backupResponse, err := owner.Client.GetLogicalBackupWithResponse(ctx, response.JSON202.Id)
		if err != nil {
			return err
		}

		if backupResponse.StatusCode() != http.StatusOK {
			return fmt.Errorf("get the logical backup answered %d", backupResponse.StatusCode())
		}

		if backupResponse.JSON200.Status == apiclient.LogicalBackupStatusINPROGRESS {
			return errBackupRunning
		}

		finishedBackup = *backupResponse.JSON200

		return nil
	})

	require.Equal(t, apiclient.LogicalBackupStatusCOMPLETED, finishedBackup.Status, "%v", finishedBackup.FailMessage)

	return finishedBackup
}

func listLogicalBackupIDs(t *testing.T, owner accountSession, databaseID uuid.UUID) []uuid.UUID {
	t.Helper()

	response, err := owner.Client.ListLogicalBackupsWithResponse(t.Context(),
		&apiclient.ListLogicalBackupsParams{DatabaseId: databaseID, Limit: new(100)})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	backupIDs := []uuid.UUID{}
	for _, backup := range response.JSON200.Items {
		backupIDs = append(backupIDs, backup.Id)
	}

	return backupIDs
}

// restoreLogically restores the backup into a new database of the source server and returns its
// name.
func restoreLogically(t *testing.T, cycle *recoveryCycle, owner accountSession, backupID uuid.UUID) string {
	t.Helper()

	restoredDatabase := "restored_" + uuid.NewString()[:8]
	_, err := cycle.openSourceDatabase(t, sourceDatabaseName).Exec(t.Context(), "CREATE DATABASE "+restoredDatabase)
	require.NoError(t, err)

	var target apiclient.RestoreTarget
	require.NoError(t, target.FromPostgresqlRestoreTarget(apiclient.PostgresqlRestoreTarget{
		Type:                apiclient.PostgresqlRestoreTargetTypePOSTGRESLOGICAL,
		Host:                sourceServerAlias,
		Port:                sourceServerPort,
		Username:            sourceUsername,
		Password:            new(sourcePassword),
		Database:            restoredDatabase,
		SslMode:             apiclient.PostgresqlSslModeDisable,
		SshTunnel:           apiclient.SshTunnelSettings{IsEnabled: false},
		CpuCount:            1,
		IsExcludeExtensions: false,
		IsRestoreOwnership:  false,
		IsRestorePrivileges: false,
	}))

	response, err := owner.Client.StartRestoreWithResponse(t.Context(),
		apiclient.RestoreStart{BackupId: backupID, Target: target})
	require.NoError(t, err)
	require.Equal(t, http.StatusAccepted, response.StatusCode(), string(response.Body))

	var finishedRestore apiclient.Restore

	testwait.Within(backupLimit).Until(t, "the logical restore finishes", func(ctx context.Context) error {
		restoreResponse, err := owner.Client.GetRestoreWithResponse(ctx, response.JSON202.Id)
		if err != nil {
			return err
		}

		if restoreResponse.StatusCode() != http.StatusOK {
			return fmt.Errorf("get the restore answered %d", restoreResponse.StatusCode())
		}

		if restoreResponse.JSON200.Status == apiclient.RestoreStatusINPROGRESS {
			return errRestoreRunning
		}

		finishedRestore = *restoreResponse.JSON200

		return nil
	})

	require.Equal(t, apiclient.RestoreStatusCOMPLETED, finishedRestore.Status, "%v", finishedRestore.FailMessage)

	return restoredDatabase
}

func requireLogicalRestoreOfSnapshot(t *testing.T, cycle *recoveryCycle, owner accountSession, backupID uuid.UUID) {
	t.Helper()

	restoredDatabase := restoreLogically(t, cycle, owner, backupID)

	require.Equal(t, getLogicalRows(), readLogicalRows(t, cycle.openSourceDatabase(t, restoredDatabase)))
}

func containsBackup(backupIDs []uuid.UUID, backupID uuid.UUID) bool {
	return slices.Contains(backupIDs, backupID)
}
