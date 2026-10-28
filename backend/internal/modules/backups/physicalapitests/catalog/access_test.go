package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupAccess(t *testing.T) {
	t.Run(
		"permits readers and managers according to workspace role when listing or triggering backups",
		func(t *testing.T) {
			catalog := newCatalog(t)
			apptest.SaveBackupConfigWithStorage(
				t,
				catalog.Owner,
				apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID},
			)
			member := catalog.App.CreateNonAdminUser(t, catalog.Admin)
			apptest.AddWorkspaceMember(
				t,
				catalog.Owner,
				catalog.WorkspaceID,
				member.Email,
				apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
			)
			viewer := catalog.addViewer(t)
			outsider := catalog.App.CreateNonAdminUser(t, catalog.Admin)
			for _, testCase := range []struct {
				name          string
				session       apptest.Session
				listStatus    int
				triggerStatus int
			}{
				{"allows both operations when the user owns the workspace", catalog.Owner, http.StatusOK, http.StatusAccepted},
				{"allows both operations when the user is a workspace member", member, http.StatusOK, http.StatusAccepted},
				{"allows reading and returns 400 for changes when the user is a viewer", viewer, http.StatusOK, http.StatusBadRequest},
				{"returns 400 for both operations when the user is not a workspace member", outsider, http.StatusBadRequest, http.StatusBadRequest},
				{"allows both operations when the user is an instance admin", catalog.Admin, http.StatusOK, http.StatusAccepted},
				{"returns 401 for both operations when the request has no session", apptest.Session{}, http.StatusUnauthorized, http.StatusUnauthorized},
			} {
				t.Run(testCase.name, func(t *testing.T) {
					path := "/backups/physical/database/" + catalog.DatabaseID.String()
					catalog.call(
						t,
						catalogCall{
							Path:          path + "/backups",
							Authorization: "Bearer " + testCase.session.Token,
							StatusCode:    testCase.listStatus,
						},
					)
					catalog.call(
						t,
						catalogCall{
							Method:        http.MethodPost,
							Path:          path + "/trigger",
							Authorization: "Bearer " + testCase.session.Token,
							Body:          map[string]string{"type": "full"},
							StatusCode:    testCase.triggerStatus,
						},
					)
				})
			}
		},
	)
}

func TestTriggerPhysicalBackup(t *testing.T) {
	t.Run("sets the full request time when a full backup is requested", func(t *testing.T) {
		catalog := newCatalog(t)
		apptest.SaveBackupConfigWithStorage(
			t,
			catalog.Owner,
			apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID},
		)
		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/backups/physical/database/" + catalog.DatabaseID.String() + "/trigger",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          map[string]string{"type": "full"},
				StatusCode:    http.StatusAccepted,
			},
		)
		var requested []struct{ ForceFullRequestedAt *time.Time }
		catalog.App.QuerySQL(
			t,
			&requested,
			"SELECT force_full_requested_at FROM backups_configs_physical WHERE database_id = ?",
			catalog.DatabaseID,
		)
		assert.NotNil(t, requested[0].ForceFullRequestedAt)
	})
	t.Run("returns 409 when an incremental has no extendable chain", func(t *testing.T) {
		catalog := newCatalog(t)
		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/backups/physical/database/" + catalog.DatabaseID.String() + "/trigger",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          map[string]string{"type": "incremental"},
				StatusCode:    http.StatusConflict,
			},
		)
	})
	t.Run("accepts an automatic request when no backup is running", func(t *testing.T) {
		catalog := newCatalog(t)
		apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
		catalog.call(
			t,
			catalogCall{
				Method:        http.MethodPost,
				Path:          "/backups/physical/database/" + catalog.DatabaseID.String() + "/trigger",
				Authorization: "Bearer " + catalog.Owner.Token,
				Body:          map[string]string{"type": "auto"},
				StatusCode:    http.StatusAccepted,
			},
		)
	})
}
