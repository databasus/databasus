package verification_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationDatabaseEvents(t *testing.T) {
	t.Parallel()

	for _, isSaved := range []bool{false, true} {
		t.Run(
			"copies the saved config or keeps the default when a database is copied with saved="+map[bool]string{false: "false", true: "true"}[isSaved],
			func(t *testing.T) {
				t.Parallel()

				team := newVerificationTeam(t)
				if isSaved {
					team.enableSchedule(t, apiclient.VerificationScheduleTypeINTERVAL)
				}

				original := team.config(t)
				copied, err := team.Owner.Client.CopyDatabaseWithResponse(t.Context(), team.Database.Id)
				require.NoError(t, err)
				require.NotNil(t, copied.JSON201, string(copied.Body))
				read, err := team.Owner.Client.GetVerificationConfigWithResponse(t.Context(), copied.JSON201.Id)
				require.NoError(t, err)
				require.NotNil(t, read.JSON200)
				assert.Equal(t, configInput(original), configInput(*read.JSON200))

				var count int64
				team.App.QuerySQL(
					t,
					&count,
					"SELECT count(*) FROM verification_configs WHERE database_id=?",
					copied.JSON201.Id,
				)

				if isSaved {
					assert.Equal(t, int64(1), count)
				} else {
					assert.Zero(t, count)
				}
			},
		)
	}
	t.Run("deletes dependent verifications and configs when their backup and database are deleted", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		team.saveConfig(t, configInput(team.config(t)))

		deletedBackup, err := team.Owner.Client.DeleteLogicalBackupWithResponse(t.Context(), backup.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deletedBackup.StatusCode(), string(deletedBackup.Body))
		read, err := team.Owner.Client.GetVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		requireProblem(t, read.StatusCode(), read.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)

		deletedDatabase, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deletedDatabase.StatusCode(), string(deletedDatabase.Body))
		var count int64
		team.App.QuerySQL(t, &count, "SELECT count(*) FROM verification_configs WHERE database_id=?", team.Database.Id)
		assert.Zero(t, count)
	})
}
