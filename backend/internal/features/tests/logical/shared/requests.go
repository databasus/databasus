//go:build testhelpers

package logicaltesting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	restores_core "databasus-backend/internal/features/restores/core"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	test_utils "databasus-backend/internal/util/testing"
)

// SubmitCreateDatabase POSTs a database create request and returns the created
// database, failing the test with a labelled message on a non-201 response.
func SubmitCreateDatabase(
	t *testing.T,
	router *gin.Engine,
	label string,
	request databases.Database,
	token string,
) *databases.Database {
	w := workspaces_testing.MakeAPIRequest(
		router, "POST", "/api/v1/databases/create", "Bearer "+token, request,
	)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create %s database. Status: %d, Body: %s",
			label, w.Code, w.Body.String())
	}

	var created databases.Database
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to unmarshal %s response: %v", label, err)
	}
	return &created
}

// SubmitRestore POSTs a restore request for a backup, asserting a 200 response.
func SubmitRestore(
	t *testing.T,
	router *gin.Engine,
	backupID uuid.UUID,
	request restores_core.RestoreBackupRequest,
	token string,
) {
	test_utils.MakePostRequest(
		t, router,
		fmt.Sprintf("/api/v1/restores/%s/restore", backupID.String()),
		"Bearer "+token, request, http.StatusOK,
	)
}
