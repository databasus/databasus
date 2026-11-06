package verification_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/agentclient"
	"databasus-backend/internal/lib/testwait"
)

func TestVerificationFailureDiagnosticCancellationRace(t *testing.T) {
	t.Parallel()

	t.Run("rejects the failed report when cancellation commits first", func(t *testing.T) {
		t.Parallel()

		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		transaction := team.App.BeginSQLTransaction(t)
		transaction.ExecuteSQL(
			t,
			"UPDATE verification_runs SET status='CANCELED', finished_at=? WHERE id=?",
			team.App.Clock.Now(),
			verification.Id,
		)
		client := team.App.NewAgentClient(t, agent)
		type reportOutcome struct {
			response *agentclient.ReportVerificationResponse
			err      error
		}
		outcomes := make(chan reportOutcome, 1)
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		go func() {
			response, err := client.ReportVerificationWithResponse(
				ctx,
				agent.Agent.ID,
				verification.Id,
				agentclient.VerificationReport{
					Status:            "FAILED",
					PgRestoreExitCode: new(1),
					FailMessage:       new("late failure"),
					FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{
						Output:      "late diagnostic",
						IsTruncated: false,
					},
				},
			)
			outcomes <- reportOutcome{response: response, err: err}
		}()
		testwait.Until(t, "the failed report waits for the verification outcome lock", func(context.Context) error {
			var waiting int64
			team.App.QuerySQL(t, &waiting, `SELECT count(*) FROM pg_stat_activity
				WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%verification_runs%'`)
			if waiting == 0 {
				return fmt.Errorf("the report has not reached the verification row")
			}
			return nil
		})

		transaction.Commit(t)

		select {
		case outcome := <-outcomes:
			require.NoError(t, outcome.err)
			require.Equal(t, http.StatusGone, outcome.response.StatusCode(), string(outcome.response.Body))
		case <-ctx.Done():
			t.Fatal("the failed report did not finish after cancellation committed")
		}
		details := team.read(t, verification.Id)
		assert.Equal(t, "CANCELED", string(details.Status))
		assert.Nil(t, details.FailureDiagnostic)
		assert.Nil(t, details.FailMessage)
	})
}
