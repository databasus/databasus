package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func GetStorageSwitch(t *testing.T, session Session, operationID uuid.UUID) apiclient.StorageSwitch {
	t.Helper()
	response, err := session.Client.GetStorageSwitchWithResponse(t.Context(), operationID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	require.NotNil(t, response.JSON200)
	return *response.JSON200
}
