package databases_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

//nolint:tparallel // The subtests share one source server.
func TestPhysicalDatabaseWalSummarization(t *testing.T) {
	t.Parallel()

	incrementalBackupTypes := []apiclient.PhysicalBackupType{
		apiclient.PhysicalBackupTypeFULLINCREMENTAL,
		apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	}

	for _, image := range []string{"postgres:17", "postgres:18"} {
		server := containers.StartPhysicalPostgres(t, image, containers.WithoutSummarizer())

		for _, backupType := range incrementalBackupTypes {
			t.Run("returns 400 database_connection_failed and creates nothing for "+string(backupType)+
				" backups when "+image+" runs without summarize_wal", func(t *testing.T) {
				tm := newTeam(t)

				response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(),
					newCreation(tm, fromPhysicalSettings(t, newPhysicalSettings(server, backupType))))

				require.NoError(t, err)
				requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
					apiclient.ProblemCodeDatabaseConnectionFailed)
				assert.Contains(t, response.ApplicationproblemJSON400.Detail,
					string(pgshared.ConnErrWalSummaryDisabled))
				assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
			})
		}

		t.Run("keeps full backups when a change to incremental backups meets "+image+" without summarize_wal",
			func(t *testing.T) {
				tm := newTeam(t)
				createdDatabase := tm.createDatabaseNamed(t, "Full only",
					fromPhysicalSettings(t, newPhysicalSettings(server, apiclient.PhysicalBackupTypeFULL)))
				incrementalPhysical := getPhysicalSettings(t, createdDatabase)
				incrementalPhysical.BackupType = apiclient.PhysicalBackupTypeFULLINCREMENTAL
				update := toUpdate(createdDatabase)
				update.Settings = fromPhysicalSettings(t, incrementalPhysical)

				response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

				requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
					apiclient.ProblemCodeDatabaseConnectionFailed)
				assert.Contains(t, response.ApplicationproblemJSON400.Detail,
					string(pgshared.ConnErrWalSummaryDisabled))
				assert.Equal(t, apiclient.PhysicalBackupTypeFULL,
					getPhysicalSettings(t, getDatabase(t, tm.owner, createdDatabase.Id)).BackupType)
			})
	}
}
