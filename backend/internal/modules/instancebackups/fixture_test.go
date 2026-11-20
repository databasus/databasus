package instancebackups_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

// instanceOwner is a fresh instance whose root admin is the first user, plus a second instance
// admin who is not root admin.
type instanceOwner struct {
	App        *apptest.App
	RootAdmin  apptest.Session
	OtherAdmin apptest.Session
	// testDay is midnight of the day after the test started, where a test moves the clock to name
	// times of day.
	testDay time.Time
}

func newInstanceOwner(t *testing.T, options ...apptest.Option) instanceOwner {
	t.Helper()

	app := apptest.New(t, options...)
	rootAdmin := app.CreateAdmin(t)
	otherAdmin := app.CreateNonAdminUser(t, rootAdmin)

	promotion, err := rootAdmin.Client.ChangeUserRoleWithResponse(
		t.Context(),
		otherAdmin.UserID,
		apiclient.UserRoleChange{Role: apiclient.UserRoleADMIN},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, promotion.StatusCode(), string(promotion.Body))

	return instanceOwner{
		App:        app,
		RootAdmin:  rootAdmin,
		OtherAdmin: otherAdmin,
		testDay:    time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour),
	}
}

func (o instanceOwner) createLocalStorage(t *testing.T, workspaceName string) uuid.UUID {
	t.Helper()

	workspace := apptest.CreateWorkspace(t, o.RootAdmin, workspaceName)

	return o.App.CreateLocalStorage(t, o.RootAdmin, workspace.Id)
}

func (o instanceOwner) getSettings(t *testing.T) apiclient.InstanceBackupSettings {
	t.Helper()

	response, err := o.RootAdmin.Client.GetInstanceBackupSettingsWithResponse(t.Context())
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	return *response.JSON200
}

func (o instanceOwner) selectDestinations(t *testing.T, storageIDs ...uuid.UUID) apiclient.InstanceBackupSettings {
	t.Helper()

	settings := o.getSettings(t)
	settings.StorageIds = append([]uuid.UUID{}, storageIDs...)

	response, err := o.RootAdmin.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
	require.NoError(t, err)
	require.NotNil(t, response.JSON200, string(response.Body))

	return *response.JSON200
}

func requireProblemCode(t *testing.T, status int, body []byte, expectedStatus int, code apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))
	require.Contains(t, string(body), `"code":"`+string(code)+`"`)
}
