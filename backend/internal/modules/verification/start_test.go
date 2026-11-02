package verification_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestStartVerification(t *testing.T) {
	t.Parallel()

	t.Run("returns a pending verification before any agent runs when the backup is completed", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		assert.Equal(t, apiclient.VerificationStatusPENDING, verification.Status)
		assert.Equal(t, apiclient.VerificationTriggerMANUAL, verification.Trigger)
		assert.Equal(t, backup.Id, verification.BackupId)
		assert.Equal(t, verification.Id, team.read(t, verification.Id).Id)
	})
	t.Run(
		"returns backup_not_completed without creating a verification when the backup has failed",
		func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			team.EnableBackups(t, team.Database.Id, team.StorageID, apiclient.BackupEncryptionNONE)
			team.App.ExecuteSQL(
				t,
				"UPDATE databases_postgresql_logical SET port=1 WHERE database_id=?",
				team.Database.Id,
			)

			backup := team.StartBackup(t, team.Database.Id)
			failed := team.WaitForBackupOutcome(t, backup.Id)
			require.Equal(t, apiclient.LogicalBackupStatusFAILED, failed.Status)

			response, err := team.Owner.Client.StartVerificationWithResponse(
				t.Context(),
				apiclient.StartVerificationInput{BackupId: backup.Id},
			)
			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusConflict,
				apiclient.ProblemCodeBackupNotCompleted,
			)

			assert.Empty(t, team.list(t))
		},
	)
	for _, status := range []apiclient.VerificationStatus{apiclient.VerificationStatusPENDING, apiclient.VerificationStatusRUNNING} {
		t.Run("returns already_running when a manual verification is "+string(status), func(t *testing.T) {
			t.Parallel()

			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)

			if status == apiclient.VerificationStatusRUNNING {
				team.App.ExecuteSQL(
					t,
					"UPDATE verification_runs SET status='RUNNING',started_at=? WHERE id=?",
					team.App.Clock.Now(),
					verification.Id,
				)
			}

			response, err := team.Owner.Client.StartVerificationWithResponse(
				t.Context(),
				apiclient.StartVerificationInput{BackupId: backup.Id},
			)
			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusConflict,
				apiclient.ProblemCodeAlreadyRunning,
			)

			assert.Len(t, team.list(t), 1)
		})
	}
	t.Run("keeps an automatic verification when a user starts a manual one", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		automatic := team.start(t, backup.Id)
		team.App.ExecuteSQL(t, "UPDATE verification_runs SET trigger='SCHEDULED' WHERE id=?", automatic.Id)

		manual := team.start(t, backup.Id)

		assert.Equal(t, apiclient.VerificationTriggerMANUAL, manual.Trigger)
		assert.Len(t, team.list(t), 2)
		assert.Equal(t, apiclient.VerificationStatusPENDING, team.read(t, automatic.Id).Status)
	})
}

func TestStartVerificationConcurrentRequests(t *testing.T) {
	t.Parallel()

	t.Run("creates one verification when members start it concurrently", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		start := make(chan struct{})
		statuses := make(chan int, 16)
		failures := make(chan error, 16)

		for range 16 {
			go func() {
				<-start
				response, err := team.Owner.Client.StartVerificationWithResponse(
					t.Context(),
					apiclient.StartVerificationInput{BackupId: backup.Id},
				)
				if err != nil {
					failures <- err
					statuses <- 0
					return
				}
				statuses <- response.StatusCode()
			}()
		}

		close(start)
		acceptedCount := 0
		for range 16 {
			status := <-statuses
			if status == http.StatusAccepted {
				acceptedCount++
			} else {
				assert.Equal(t, http.StatusConflict, status)
			}
		}

		close(failures)
		for err := range failures {
			require.NoError(t, err)
		}

		assert.Equal(t, 1, acceptedCount)
		assert.Len(t, team.list(t), 1)
	})
}
