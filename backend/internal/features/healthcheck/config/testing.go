package healthcheck_config

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
)

func EnableHealthcheckForTestDatabase(databaseID uuid.UUID, managerToken string, router *gin.Engine) {
	request := HealthcheckConfigDTO{
		DatabaseID:                        databaseID,
		IsHealthcheckEnabled:              true,
		IsSentNotificationWhenUnavailable: false,
		IntervalMinutes:                   1,
		AttemptsBeforeConcideredAsDown:    3,
		StoreAttemptsDays:                 7,
	}

	response := workspaces_testing.MakeAPIRequest(
		router,
		"POST",
		"/api/v1/healthcheck-config",
		"Bearer "+managerToken,
		request,
	)

	if response.Code != http.StatusOK {
		panic("Failed to enable healthcheck for test database: " + response.Body.String())
	}
}
