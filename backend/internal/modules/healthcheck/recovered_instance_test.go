package healthcheck_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestHealthcheckAfterRecovery(t *testing.T) {
	t.Parallel()

	t.Run("probes a due database without any confirmation after a recovery", func(t *testing.T) {
		t.Parallel()
		app := apptest.New(t, apptest.WithPublishedRecovery())
		admin := app.CreateAdmin(t)
		workspace := apptest.CreateWorkspace(t, admin, "Team "+uuid.NewString()[:8])
		databaseID := app.InsertDatabase(t, workspace.Id)

		app.RunJobPass(t, "healthcheck_probe")
		app.WaitForNotificationDeliveries(t)

		var attemptCount int64
		app.QuerySQL(t, &attemptCount, "SELECT count(*) FROM healthcheck_attempts WHERE database_id = ?", databaseID)
		assert.Positive(t, attemptCount)
	})
}
