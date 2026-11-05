package catalog_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupAccess(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name          string
		role          string
		listStatus    int
		triggerStatus int
		code          apiclient.ProblemCode
	}{
		{"allows both operations when the user owns the workspace", "OWNER", http.StatusOK, http.StatusAccepted, ""},
		{"allows both operations when the user is a workspace member", "MEMBER", http.StatusOK, http.StatusAccepted, ""},
		{"allows reading and returns 403 for changes when the user is a viewer", "VIEWER", http.StatusOK, http.StatusForbidden, apiclient.ProblemCodePermissionDenied},
		{"returns 404 for both operations when the user is not a workspace member", "OUTSIDER", http.StatusNotFound, http.StatusNotFound, apiclient.ProblemCodeNotFound},
		{"allows both operations when the user is an instance admin", "ADMIN", http.StatusOK, http.StatusAccepted, ""},
		{"returns 401 for both operations when the request has no session", "NO_SESSION", http.StatusUnauthorized, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)

			apptest.SaveBackupConfigWithStorage(t, catalog.Owner,
				apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID})

			session := catalog.Owner

			switch scenario.role {
			case "MEMBER":
				session = catalog.App.CreateNonAdminUser(t, catalog.Admin)
				apptest.AddWorkspaceMember(t, catalog.Owner, catalog.WorkspaceID, session.Email,
					apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER)
			case "VIEWER":
				session = catalog.addViewer(t)
			case "OUTSIDER":
				session = catalog.App.CreateNonAdminUser(t, catalog.Admin)
			case "ADMIN":
				session = catalog.Admin
			case "NO_SESSION":
				session = apptest.Session{}
			}

			listResponse := catalog.call(t, catalogCall{
				Request:       apptest.PhysicalListRequest(catalog.DatabaseID, ""),
				Authorization: "Bearer " + session.Token, StatusCode: scenario.listStatus,
			})

			startResponse := catalog.call(t, catalogCall{
				Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, "FULL"),
				Authorization: "Bearer " + session.Token, StatusCode: scenario.triggerStatus,
			})

			if scenario.triggerStatus >= 400 {
				var problem apiclient.Problem
				require.NoError(t, json.Unmarshal(startResponse.Body, &problem))
				assert.Equal(t, scenario.code, problem.Code)
				assert.Equal(t, backupRequests{}, catalog.getBackupRequests(t))
			}

			if scenario.listStatus >= 400 {
				var problem apiclient.Problem
				require.NoError(t, json.Unmarshal(listResponse.Body, &problem))
				assert.Equal(t, scenario.code, problem.Code)
			}
		})
	}
}
