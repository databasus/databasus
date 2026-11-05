package databases_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func TestDatabaseEnumValidation(t *testing.T) {
	t.Parallel()

	for _, field := range []string{"sslMode", "backupType"} {
		t.Run("returns 400 and creates nothing with an unsupported "+field, func(t *testing.T) {
			t.Parallel()
			tm := newTeam(t)
			server := containers.Endpoint{Host: "127.0.0.1", Port: 1}
			logical := newLogicalSettings(server)
			logical.SslMode = "prefer"
			settings := fromLogicalSettings(t, logical)
			if field == "backupType" {
				settings = fromPhysicalSettings(t, newPhysicalSettings(server, "DIFFERENTIAL"))
			}

			response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Invalid database",
				Settings:    settings,
				NotifierIds: []uuid.UUID{},
			})

			require.NoError(t, err)
			requireProblem(
				t,
				response.StatusCode(),
				response.Body,
				http.StatusBadRequest,
				apiclient.ProblemCodeInvalidRequest,
			)
			assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
		})
	}
}
