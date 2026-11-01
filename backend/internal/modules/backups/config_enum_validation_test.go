package backups_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestBackupConfigEnumValidation(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"encryption", "schedule", "retention", "notificationTriggers"} {
		t.Run("returns 400 and preserves the config with an unsupported "+field, func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			databaseID := tm.insertLogicalDatabase(t)
			storageID := tm.createLocalStorage(t)
			originalConfig := saveLogicalBackupConfig(t, tm.owner, databaseID, storageID)

			logical := newLogicalBackupSettings()
			switch field {
			case "schedule":
				logical.Schedule.Type = "YEARLY"
			case "retention":
				logical.Retention.Policy = "ALL"
			case "notificationTriggers":
				logical.NotificationTriggers = []apiclient.LogicalNotificationTrigger{"WAL_GAP"}
			}

			config := newLogicalBackupConfigInput(t, &storageID, logical)
			if field == "encryption" {
				config.Encryption = "AES"
			}

			response, err := tm.owner.Client.UpdateBackupConfigWithResponse(t.Context(), databaseID, config)

			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest,
			)
			assert.Equal(t, originalConfig, *getBackupConfig(t, tm.owner, databaseID).JSON200)
		})
	}
}
