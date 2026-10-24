//go:build testhelpers

package verification_config

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	test_utils "databasus-backend/internal/util/testing"
)

func SaveVerificationConfigViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	databaseID uuid.UUID,
	request SaveBackupVerificationConfigDTO,
) *BackupVerificationConfig {
	t.Helper()

	var response BackupVerificationConfig
	test_utils.MakePutRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verification-config/"+databaseID.String(),
		"Bearer "+userToken,
		request,
		http.StatusOK,
		&response,
	)

	return &response
}
