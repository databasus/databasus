package verification_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/agentclient"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestVerificationFailureDiagnostic(t *testing.T) {
	t.Parallel()

	t.Run("preserves the underlying error when an owned failed report includes a diagnostic", func(t *testing.T) {
		t.Parallel()
		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		require.Equal(t, verification.Id, claim(t, team, agent))
		output := `pg_restore: error: type "geometry" does not exist`

		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(verification.Id, agentclient.VerificationReport{
			Status: "FAILED", PgRestoreExitCode: new(1), FailMessage: new("pg_restore failed"),
			FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{Output: output, IsTruncated: false},
		})

		require.Equal(t, http.StatusNoContent, status, string(body))
		details := team.read(t, verification.Id)
		require.NotNil(t, details.FailureDiagnostic)
		assert.Equal(t, output, details.FailureDiagnostic.Output)
		assert.False(t, details.FailureDiagnostic.IsTruncated)
	})
	t.Run("preserves the outcome and diagnostic when the failed report cannot commit", func(t *testing.T) {
		t.Parallel()
		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		original := team.read(t, verification.Id)
		team.App.ExecuteSQL(t, `CREATE FUNCTION reject_verification_report_commit() RETURNS trigger
			LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'deliberate verification commit failure'; END $$`)
		team.App.ExecuteSQL(t, `CREATE CONSTRAINT TRIGGER reject_verification_report_commit
			AFTER UPDATE ON verification_runs DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW WHEN (NEW.status = 'FAILED')
			EXECUTE FUNCTION reject_verification_report_commit()`)

		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(verification.Id, agentclient.VerificationReport{
			Status:            "FAILED",
			PgRestoreExitCode: new(1),
			FailMessage:       new("restore rejected"),
			FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{
				Output:      "diagnostic from rejected transaction",
				IsTruncated: false,
			},
		})

		require.Equal(t, http.StatusInternalServerError, status, string(body))
		assert.Equal(t, original, team.read(t, verification.Id))
	})
}

func TestVerificationFailureDiagnosticValidation(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name       string
		diagnostic any
		status     string
	}{
		{name: "an empty diagnostic", diagnostic: map[string]any{"output": "", "isTruncated": false}, status: "FAILED"},
		{name: "a missing truncation flag", diagnostic: map[string]any{"output": "error"}, status: "FAILED"},
		{name: "an explicit null", diagnostic: nil, status: "FAILED"},
		{name: "a character overflow", diagnostic: map[string]any{"output": strings.Repeat("a", 65537), "isTruncated": false}, status: "FAILED"},
		{name: "a byte overflow", diagnostic: map[string]any{"output": strings.Repeat("\u754c", 22000), "isTruncated": false}, status: "FAILED"},
		{name: "a diagnostic on success", diagnostic: map[string]any{"output": "error", "isTruncated": false}, status: "COMPLETED"},
	} {
		t.Run("rejects the report without mutation when it contains "+scenario.name, func(t *testing.T) {
			t.Parallel()
			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)
			original := team.read(t, verification.Id)
			payload, err := json.Marshal(
				map[string]any{
					"status":            scenario.status,
					"pgRestoreExitCode": 1,
					"failureDiagnostic": scenario.diagnostic,
				},
			)
			require.NoError(t, err)

			status, body := newProtocolAgent(t, team.App, agent).postRawJSON(
				"/api/v1/agent/verifications/"+agent.Agent.ID.String()+
					"/"+verification.Id.String()+
					"/report", string(payload))

			require.Equal(t, http.StatusBadRequest, status, string(body))
			assert.Contains(t, string(body), "invalid_request")
			assert.Equal(t, original, team.read(t, verification.Id))
		})
	}
}

func TestVerificationFailureDiagnosticPrivacy(t *testing.T) {
	t.Parallel()
	t.Run(
		"sanitizes reported errors and restricts diagnostics to details when an agent reports directly",
		func(t *testing.T) {
			t.Parallel()
			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			claim(t, team, agent)
			output := "pg_restore: error: type geometry is missing postgres://admin:password@db/source\nDETAIL: row-value\ncontinued-private-row\nCommand was: CREATE FUNCTION secret_body() RETURNS text AS $$\nERROR: error-shaped-private-value\nFATAL: fatal-shaped-private-value\npg_restore: error: restore-shaped-private-value\nremaining-private-body\n$$ LANGUAGE sql;\npg_restore: warning: errors ignored on restore: 1\n"

			status, body := newProtocolAgent(
				t,
				team.App,
				agent,
			).reportVerification(verification.Id, agentclient.VerificationReport{
				Status: "FAILED", PgRestoreExitCode: new(1), FailMessage: new(output),
				FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{Output: output, IsTruncated: true},
			})

			require.Equal(t, http.StatusNoContent, status, string(body))
			diagnostic := team.read(t, verification.Id)
			require.NotNil(t, diagnostic.FailureDiagnostic)
			assert.Contains(t, diagnostic.FailureDiagnostic.Output, "type geometry is missing")
			assert.True(t, diagnostic.FailureDiagnostic.IsTruncated)
			for _, privateValue := range []string{"password", "row-value", "continued-private-row", "secret_body", "error-shaped-private-value", "fatal-shaped-private-value", "restore-shaped-private-value", "remaining-private-body"} {
				assert.NotContains(t, diagnostic.FailureDiagnostic.Output, privateValue)
				assert.NotContains(t, *diagnostic.FailMessage, privateValue)
			}
			listed, err := team.Owner.Client.ListVerificationsWithResponse(
				t.Context(),
				&apiclient.ListVerificationsParams{DatabaseId: team.Database.Id},
			)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, listed.StatusCode())
			assert.NotContains(t, string(listed.Body), "failureDiagnostic")
			viewer := team.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			authorized, err := viewer.Client.GetVerificationWithResponse(t.Context(), verification.Id)
			require.NoError(t, err)
			require.NotNil(t, authorized.JSON200)
			assert.Equal(t, diagnostic.FailureDiagnostic, authorized.JSON200.FailureDiagnostic)
			stranger := team.App.SignUp(t, apptest.NewTestEmail("diagnostic-stranger"))
			refused, err := stranger.Client.GetVerificationWithResponse(t.Context(), verification.Id)
			require.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, refused.StatusCode())
			assert.NotContains(t, string(refused.Body), "geometry")
			anonymous, err := team.App.NewSessionClient(t, "").GetVerificationWithResponse(t.Context(), verification.Id)
			require.NoError(t, err)
			assert.Equal(t, http.StatusUnauthorized, anonymous.StatusCode())
		},
	)
	t.Run("keeps output within the byte limit when redaction expands it", func(t *testing.T) {
		t.Parallel()
		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		output := strings.Repeat("token=x\n", 8192)

		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(verification.Id, agentclient.VerificationReport{
			Status:            "FAILED",
			PgRestoreExitCode: new(1),
			FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{Output: output, IsTruncated: false},
		})

		require.Equal(t, http.StatusNoContent, status, string(body))
		diagnostic := team.read(t, verification.Id).FailureDiagnostic
		require.NotNil(t, diagnostic)
		assert.LessOrEqual(t, len(diagnostic.Output), 65536)
		assert.True(t, diagnostic.IsTruncated)
		assert.NotContains(t, diagnostic.Output, "token=x")
	})
}

func TestVerificationFailureDiagnosticLifecycle(t *testing.T) {
	t.Parallel()
	for _, terminalReport := range []struct {
		name   string
		report agentclient.VerificationReport
	}{
		{name: "a new error", report: agentclient.VerificationReport{Status: "FAILED", PgRestoreExitCode: new(1), FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{Output: "final error", IsTruncated: false}}},
		{name: "no captured output", report: agentclient.VerificationReport{Status: "FAILED", PgRestoreExitCode: new(1)}},
		{name: "a successful verification", report: agentclient.VerificationReport{Status: "COMPLETED"}},
	} {
		t.Run("clears earlier attempts when the final report contains "+terminalReport.name, func(t *testing.T) {
			t.Parallel()
			team := newVerificationTeam(t)
			backup := team.backup(t)
			verification := team.start(t, backup.Id)
			agent := team.App.CreateVerificationAgent(t, team.Admin)
			for attempt := 1; attempt < 3; attempt++ {
				claim(t, team, agent)
				status, body := newProtocolAgent(
					t,
					team.App,
					agent,
				).reportVerification(verification.Id, agentclient.VerificationReport{
					Status:      "FAILED",
					FailMessage: new("setup failed"),
					FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{
						Output:      fmt.Sprintf("attempt %d error", attempt),
						IsTruncated: false,
					},
				})
				require.Equal(t, http.StatusNoContent, status, string(body))
				details := team.read(t, verification.Id)
				assert.Equal(t, apiclient.VerificationStatusPENDING, details.Status)
				assert.Nil(t, details.FailureDiagnostic)
			}
			claim(t, team, agent)

			status, body := newProtocolAgent(
				t,
				team.App,
				agent,
			).reportVerification(verification.Id, terminalReport.report)

			require.Equal(t, http.StatusNoContent, status, string(body))
			details := team.read(t, verification.Id)
			if terminalReport.report.FailureDiagnostic == nil {
				assert.Nil(t, details.FailureDiagnostic)
			} else {
				require.NotNil(t, details.FailureDiagnostic)
				assert.Equal(t, "final error", details.FailureDiagnostic.Output)
			}
		})
	}
	t.Run("refuses a stale diagnostic after cancellation", func(t *testing.T) {
		t.Parallel()
		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		canceled, err := team.Owner.Client.CancelVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, canceled.StatusCode())
		original := team.read(t, verification.Id)

		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(verification.Id, agentclient.VerificationReport{
			Status:            "FAILED",
			PgRestoreExitCode: new(1),
			FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{Output: "stale error", IsTruncated: false},
		})

		require.Equal(t, http.StatusGone, status, string(body))
		assert.Equal(t, original, team.read(t, verification.Id))
	})
	t.Run("refuses the previous owner after reassignment", func(t *testing.T) {
		t.Parallel()
		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		firstAgent := team.App.CreateVerificationAgent(t, team.Admin)
		secondAgent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, firstAgent)
		status, body := newProtocolAgent(
			t,
			team.App,
			firstAgent,
		).reportVerification(verification.Id, agentclient.VerificationReport{Status: "FAILED", FailMessage: new("setup failed")})
		require.Equal(t, http.StatusNoContent, status, string(body))
		require.Equal(t, verification.Id, claim(t, team, secondAgent))
		original := team.read(t, verification.Id)

		status, body = newProtocolAgent(
			t,
			team.App,
			firstAgent,
		).reportVerification(verification.Id, agentclient.VerificationReport{
			Status:            "FAILED",
			PgRestoreExitCode: new(1),
			FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{Output: "stale error", IsTruncated: false},
		})

		require.Equal(t, http.StatusGone, status, string(body))
		assert.Equal(t, original, team.read(t, verification.Id))
	})
	t.Run("deletes the persisted diagnostic with its backup verification", func(t *testing.T) {
		t.Parallel()
		team := newVerificationTeam(t)
		backup := team.backup(t)
		verification := team.start(t, backup.Id)
		agent := team.App.CreateVerificationAgent(t, team.Admin)
		claim(t, team, agent)
		status, body := newProtocolAgent(
			t,
			team.App,
			agent,
		).reportVerification(verification.Id, agentclient.VerificationReport{
			Status:            "FAILED",
			PgRestoreExitCode: new(1),
			FailureDiagnostic: &agentclient.VerificationFailureDiagnostic{
				Output:      "archive rejection",
				IsTruncated: false,
			},
		})
		require.Equal(t, http.StatusNoContent, status, string(body))
		require.NotNil(t, team.read(t, verification.Id).FailureDiagnostic)

		deleted, err := team.Owner.Client.DeleteLogicalBackupWithResponse(t.Context(), backup.Id)

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleted.StatusCode(), string(deleted.Body))
		read, err := team.Owner.Client.GetVerificationWithResponse(t.Context(), verification.Id)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, read.StatusCode())
		var rows int64
		team.App.QuerySQL(t, &rows, "SELECT count(*) FROM verification_runs WHERE id=?", verification.Id)
		assert.Zero(t, rows)
	})
}
