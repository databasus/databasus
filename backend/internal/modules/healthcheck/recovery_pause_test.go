package healthcheck_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestHealthcheckDuringRecoveryPause(t *testing.T) {
	t.Parallel()

	for _, pauseCase := range []struct {
		name            string
		options         []apptest.Option
		isProbeExpected bool
	}{
		{name: "probes a due database on an active instance", options: nil, isProbeExpected: true},
		{
			name:            "probes no database while the recovered instance is paused",
			options:         []apptest.Option{apptest.WithRecoveryPause()},
			isProbeExpected: false,
		},
	} {
		t.Run(pauseCase.name, func(t *testing.T) {
			t.Parallel()
			app := apptest.New(t, pauseCase.options...)
			admin := app.CreateAdmin(t)
			workspace := apptest.CreateWorkspace(t, admin, "Team "+uuid.NewString()[:8])
			databaseID := app.InsertDatabase(t, workspace.Id)

			app.RunJobPass(t, "healthcheck_probe")
			app.WaitForNotificationDeliveries(t)

			var attemptCount int64
			app.QuerySQL(
				t,
				&attemptCount,
				"SELECT count(*) FROM healthcheck_attempts WHERE database_id = ?",
				databaseID,
			)
			assert.Equal(t, pauseCase.isProbeExpected, attemptCount > 0)
		})
	}
}
