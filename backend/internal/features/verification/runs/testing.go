//go:build testhelpers

package verification_runs

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_bridge "databasus-backend/internal/features/backups/config"
	"databasus-backend/internal/modules/backups"
	test_utils "databasus-backend/internal/util/testing"
)

func EnqueueManualVerificationViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	backupID uuid.UUID,
) *RestoreVerification {
	t.Helper()

	var response RestoreVerification
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verifications/enqueue",
		"Bearer "+userToken,
		EnqueueManualRequest{BackupID: backupID},
		http.StatusOK,
		&response,
	)

	return &response
}

func CancelVerificationViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	verificationID uuid.UUID,
) {
	t.Helper()

	test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/verifications/"+verificationID.String()+"/cancel",
		"Bearer "+userToken,
		nil,
		http.StatusNoContent,
	)
}

func GetVerificationByIDViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	verificationID uuid.UUID,
) *RestoreVerification {
	t.Helper()

	var response RestoreVerification
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verifications/"+verificationID.String(),
		"Bearer "+userToken,
		http.StatusOK,
		&response,
	)

	return &response
}

func ClaimVerificationViaAPI(
	t *testing.T,
	router *gin.Engine,
	agentID uuid.UUID,
	agentToken string,
	capacity AgentCapacity,
) *JobAssignment {
	t.Helper()

	var response JobAssignment
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		"/api/v1/agent/verifications/"+agentID.String()+"/claim",
		"Bearer "+agentToken,
		ClaimRequest{Capacity: capacity},
		http.StatusOK,
		&response,
	)

	return &response
}

func GetLogicalBackupForTest(t *testing.T, backupID uuid.UUID) backups.LogicalBackup {
	t.Helper()

	foundBackups, err := backups_bridge.GetModule().GetLogicalBackups(t.Context(), []uuid.UUID{backupID})
	if err != nil {
		t.Fatalf("read logical backup %s: %v", backupID, err)
	}

	if len(foundBackups) == 0 {
		t.Fatalf("logical backup %s not found", backupID)
	}

	return foundBackups[0]
}

func ListVerificationsByDatabaseViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	databaseID uuid.UUID,
) []*RestoreVerification {
	t.Helper()

	var response GetVerificationsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/verifications/by-database/"+databaseID.String()+"?limit=1000",
		"Bearer "+userToken,
		http.StatusOK,
		&response,
	)

	return response.Verifications
}
