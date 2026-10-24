//go:build testhelpers

package verification_runs

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_dto_logical "databasus-backend/internal/features/backups/backups/dto/logical"
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

func GetBackupViaAPI(
	t *testing.T,
	router *gin.Engine,
	userToken string,
	databaseID, backupID uuid.UUID,
) *backups_core_logical.LogicalBackup {
	t.Helper()

	var response backups_dto_logical.GetBackupsResponse
	test_utils.MakeGetRequestAndUnmarshal(
		t,
		router,
		"/api/v1/backups?database_id="+databaseID.String()+"&limit=1000",
		"Bearer "+userToken,
		http.StatusOK,
		&response,
	)

	for _, backup := range response.Backups {
		if backup.ID == backupID {
			return backup
		}
	}

	t.Fatalf("backup %s not found for database %s", backupID, databaseID)

	return nil
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
