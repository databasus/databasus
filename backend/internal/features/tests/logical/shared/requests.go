//go:build testhelpers

package logicaltesting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	restores_core "databasus-backend/internal/features/restores/core"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/lib/apiclient"
	test_utils "databasus-backend/internal/util/testing"
)

// NewDatabaseCreate builds the contract body that creates a database with the settings of one type,
// without notifiers, under a unique name.
func NewDatabaseCreate(t *testing.T, workspaceID uuid.UUID, settings any) apiclient.DatabaseCreate {
	t.Helper()

	return apiclient.DatabaseCreate{
		Name:        "test " + uuid.NewString(),
		NotifierIds: []uuid.UUID{},
		Settings:    NewDatabaseSettings(t, settings),
		WorkspaceId: workspaceID,
	}
}

// SubmitCreateDatabase creates a database through the contract and returns it, failing the test
// with a labelled message on any answer other than 201.
func SubmitCreateDatabase(
	t *testing.T,
	router *gin.Engine,
	label string,
	request apiclient.DatabaseCreate,
	token string,
) apiclient.Database {
	t.Helper()

	w := workspaces_testing.MakeAPIRequest(router, "POST", "/api/v1/databases", "Bearer "+token, request)
	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create %s database. Status: %d, Body: %s", label, w.Code, w.Body.String())
	}

	var created apiclient.Database
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to unmarshal %s response: %v", label, err)
	}

	return created
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
