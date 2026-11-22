package neon

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testreport"
)

func TestBackupAndRestoreOnNeon(t *testing.T) {
	t.Parallel()

	t.Run("restores every backup of the owner and a limited user into a new database and over the source",
		func(t *testing.T) {
			t.Parallel()

			runBackupAndRestoreCycle(t, requireNeonConnectionURL(t))
		})
}

func TestBackupAndRestoreCycleOnContainer(t *testing.T) {
	t.Parallel()

	t.Run("restores every backup of the owner and a limited user into a new database and over the source",
		func(t *testing.T) {
			t.Parallel()
			_, connectionURL := startContainerServer(t)

			runBackupAndRestoreCycle(t, connectionURL)
		})

	t.Run("passes both cycles and leaves no marked object when two cycles share one server", func(t *testing.T) {
		t.Parallel()
		_, connectionURL := startContainerServer(t)

		t.Run("two cycles at once", func(t *testing.T) {
			for _, cycleName := range []string{"first cycle", "second cycle"} {
				t.Run(cycleName, func(t *testing.T) {
					t.Parallel()

					runBackupAndRestoreCycle(t, connectionURL)
				})
			}
		})

		server := parseContainerServer(t, connectionURL)
		admin := server.open(t, server.maintenanceDatabase)
		assert.Empty(t, queryNames(t, server, admin,
			`SELECT datname FROM pg_database WHERE starts_with(datname, $1)`, objectMarkerPrefix))
		assert.Empty(t, queryNames(t, server, admin,
			`SELECT rolname FROM pg_roles WHERE starts_with(rolname, $1)
				OR starts_with(COALESCE(shobj_description(oid, 'pg_authid'), ''), $1)`, objectMarkerPrefix))
	})

	t.Run("fails without the password or the connection string in the message when the password is refused",
		func(t *testing.T) {
			t.Parallel()
			endpoint, _ := startContainerServer(t)
			const refusedPassword = "refused-Pa55word"
			connectionURL := newContainerConnectionURL(endpoint, refusedPassword)
			reporter := testreport.NewRecorder(t)

			reporter.RunWithCleanups(func(reporter testreport.Reporter) {
				openManagedServer(reporter, connectionURL)
			})

			failures := reporter.GetFailures()
			require.NotEmpty(t, failures, "a refused password fails the test instead of skipping it")

			for _, failure := range failures {
				assert.NotContains(t, failure, refusedPassword)
				assert.NotContains(t, failure, connectionURL)
				assert.Contains(t, failure, endpoint.Host)
				assert.Contains(t, failure, containers.PostgresUsername)
			}
		})
}

// runBackupAndRestoreCycle seeds a database of its own on the server behind the URL, backs it up
// through the API as the URL's role and as a limited user Databasus creates, and restores each
// backup into a new database and over the source, checking the data after every restore.
func runBackupAndRestoreCycle(t *testing.T, connectionURL string) {
	t.Helper()

	server := openManagedServer(t, connectionURL)
	sweepAbandonedObjects(t, server, time.Now())
	run := startTestRun(t, server, time.Now())

	sourceDatabase := run.createDatabase(t, "source")
	server.execute(t, server.open(t, sourceDatabase), seedStatements)
	sourceSnapshot := takeSnapshot(t, server, sourceDatabase)
	require.Len(t, sourceSnapshot.values, seededValueCount, "the comparison needs every seeded value")

	tm := enginetest.NewTeam(t)
	ownerSettings := server.newLogicalSettings(sourceDatabase)
	requireConnectionTestPasses(t, tm, ownerSettings)

	database := tm.CreateDatabase(t, ownerSettings)
	ownerBackup := tm.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)

	limitedUser := tm.CreateLimitedUser(t, database)
	run.markRole(t, limitedUser.Username)

	limitedSettings := server.newLogicalSettings(sourceDatabase)
	limitedSettings.Username = limitedUser.Username
	limitedSettings.Password = &limitedUser.Password
	requireNoExcessPrivileges(t, tm, limitedSettings)

	limitedDatabase := tm.UpdateDatabaseSettings(t, database, limitedSettings)
	limitedBackup := tm.BackUp(t, limitedDatabase.Id, apiclient.BackupEncryptionNONE)

	for _, backup := range []struct {
		role   string
		backup apiclient.LogicalBackup
	}{{"owner", ownerBackup}, {"limited", limitedBackup}} {
		newDatabase := run.createDatabase(t, "restored_"+backup.role)
		tm.Restore(t, backup.backup.Id, toRestoreTarget(t, server.newRestoreTarget(newDatabase)))
		requireSameData(t, sourceSnapshot, takeSnapshot(t, server, newDatabase))

		changeSourceData(t, server, sourceSnapshot)
		tm.Restore(t, backup.backup.Id, toRestoreTarget(t, server.newRestoreTarget(sourceDatabase)))
		requireSameData(t, sourceSnapshot, takeSnapshot(t, server, sourceDatabase))
	}
}

// openManagedServer parses the URL and checks that the server accepts it, so a wrong URL or a
// refused password fails before the cycle starts.
func openManagedServer(reporter testreport.Reporter, connectionURL string) managedServer {
	reporter.Helper()

	server, err := parseManagedServer(connectionURL)
	if err != nil {
		reporter.Fatalf("read the connection URL: %s", err)
	}

	server.open(reporter, server.maintenanceDatabase)

	return server
}

func requireConnectionTestPasses(t *testing.T, tm enginetest.Team, settings apiclient.PostgresqlLogicalSettings) {
	t.Helper()

	response, err := tm.Owner.Client.TestDatabaseSettingsWithResponse(t.Context(), apiclient.DatabaseSettingsTarget{
		WorkspaceId: tm.Workspace.Id,
		DatabaseId:  nil,
		Settings:    enginetest.NewDatabaseSettings(t, settings),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	require.True(t, response.JSON200.IsConnected, "the connection test failed: %s", string(response.Body))
}

func requireNoExcessPrivileges(t *testing.T, tm enginetest.Team, settings apiclient.PostgresqlLogicalSettings) {
	t.Helper()

	response, err := tm.Owner.Client.CheckLimitedUserWithResponse(t.Context(), apiclient.DatabaseSettingsTarget{
		WorkspaceId: tm.Workspace.Id,
		DatabaseId:  nil,
		Settings:    enginetest.NewDatabaseSettings(t, settings),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
	assert.False(t, response.JSON200.IsAdvisable, "the limited user may do more than a backup needs")
	assert.Empty(t, response.JSON200.Privileges)
}

// changeSourceData changes a row, removes a row and moves a sequence, so a restore over the source
// that did nothing would leave a difference behind.
func changeSourceData(t *testing.T, server managedServer, sourceSnapshot databaseSnapshot) {
	t.Helper()

	server.execute(t, server.open(t, sourceSnapshot.databaseName), `
		UPDATE public.customers SET name = 'Changed before the restore' WHERE id = 1;
		DELETE FROM public.orders WHERE id = 3;
		SELECT setval('public.invoice_numbers', 5000);`)

	require.NotEmpty(t, findSnapshotDifferences(sourceSnapshot, takeSnapshot(t, server, sourceSnapshot.databaseName)),
		"the source must differ from its snapshot before the restore over it")
}

func requireSameData(t *testing.T, source, restored databaseSnapshot) {
	t.Helper()

	if differences := findSnapshotDifferences(source, restored); len(differences) > 0 {
		t.Fatalf("the restored data differs from the source:\n%s", strings.Join(differences, "\n"))
	}
}

func toRestoreTarget(t *testing.T, target apiclient.PostgresqlRestoreTarget) apiclient.RestoreTarget {
	t.Helper()

	var restoreTarget apiclient.RestoreTarget
	require.NoError(t, restoreTarget.FromPostgresqlRestoreTarget(target))

	return restoreTarget
}
