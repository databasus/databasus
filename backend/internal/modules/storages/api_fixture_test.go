package storages_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func requireProblem(t *testing.T, status int, body []byte, expectedStatus int, expectedCode apiclient.ProblemCode) {
	t.Helper()

	require.Equal(t, expectedStatus, status, string(body))

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.Equal(t, expectedCode, problem.Code, string(body))
}

func requireFieldError(t *testing.T, body []byte, expectedField string) {
	t.Helper()

	var problem apiclient.Problem
	require.NoError(t, json.Unmarshal(body, &problem), string(body))
	require.NotNil(t, problem.Errors, string(body))

	refusedFields := make([]string, 0, len(*problem.Errors))
	for _, fieldError := range *problem.Errors {
		refusedFields = append(refusedFields, fieldError.Field)
	}

	require.Contains(t, refusedFields, expectedField, string(body))
}

// team is an instance with a workspace owned by a user who is not an instance admin.
type team struct {
	app       *apptest.App
	admin     apptest.Session
	owner     apptest.Session
	workspace apiclient.Workspace
}

func newTeam(t *testing.T, options ...apptest.Option) team {
	t.Helper()

	app := apptest.New(t, options...)
	admin := app.CreateAdmin(t)
	owner := app.CreateNonAdminUser(t, admin)

	return team{
		app:       app,
		admin:     admin,
		owner:     owner,
		workspace: apptest.CreateWorkspace(t, owner, "Team "+uuid.NewString()[:8]),
	}
}

// addMember signs up a user and adds them to the team's workspace with the role.
func (tm team) addMember(t *testing.T, role apiclient.AssignableWorkspaceRole) apptest.Session {
	t.Helper()

	user := tm.app.SignUp(t, apptest.NewTestEmail("member"))
	apptest.AddWorkspaceMember(t, tm.owner, tm.workspace.Id, user.Email, role)

	return user
}

// stranger signs up a user who is a member of no workspace of the team.
func (tm team) stranger(t *testing.T) apptest.Session {
	t.Helper()

	return tm.app.SignUp(t, apptest.NewTestEmail("stranger"))
}

// createLocalStorage creates a local storage named name in the team's workspace.
func (tm team) createLocalStorage(t *testing.T, name string) apiclient.Storage {
	t.Helper()

	return createStorage(t, tm.owner, apiclient.StorageCreate{
		WorkspaceId: tm.workspace.Id,
		Name:        name,
		Settings:    newLocalSettings(t),
	})
}

func createStorage(t *testing.T, session apptest.Session, creation apiclient.StorageCreate) apiclient.Storage {
	t.Helper()

	response, err := session.Client.CreateStorageWithResponse(t.Context(), creation)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return *response.JSON201
}

func getStorage(t *testing.T, session apptest.Session, storageID uuid.UUID) apiclient.Storage {
	t.Helper()

	response, err := session.Client.GetStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func listStorages(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []apiclient.Storage {
	t.Helper()

	response, err := session.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
		WorkspaceId: workspaceID,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.JSON200.Items
}

func getListedStorageIDs(t *testing.T, session apptest.Session, workspaceID uuid.UUID) []uuid.UUID {
	t.Helper()

	listedStorages := listStorages(t, session, workspaceID)

	storageIDs := make([]uuid.UUID, 0, len(listedStorages))
	for _, listedStorage := range listedStorages {
		storageIDs = append(storageIDs, listedStorage.Id)
	}

	return storageIDs
}

func updateStorage(
	t *testing.T,
	session apptest.Session,
	storageID uuid.UUID,
	update apiclient.StorageUpdate,
) *apiclient.UpdateStorageResponse {
	t.Helper()

	response, err := session.Client.UpdateStorageWithResponse(t.Context(), storageID, update)
	require.NoError(t, err)

	return response
}

func toSettings(t *testing.T, convert func(settings *apiclient.StorageSettings) error) apiclient.StorageSettings {
	t.Helper()

	var settings apiclient.StorageSettings
	require.NoError(t, convert(&settings))

	return settings
}

func newLocalSettings(t *testing.T) apiclient.StorageSettings {
	t.Helper()

	return toSettings(t, func(settings *apiclient.StorageSettings) error {
		return settings.FromLocalStorageSettings(apiclient.LocalStorageSettings{
			Type: apiclient.LocalStorageSettingsTypeLOCAL,
		})
	})
}

func newS3Settings(t *testing.T, s3 apiclient.S3StorageSettings) apiclient.StorageSettings {
	t.Helper()

	s3.Type = apiclient.S3StorageSettingsTypeS3

	return toSettings(t, func(settings *apiclient.StorageSettings) error {
		return settings.FromS3StorageSettings(s3)
	})
}

func newNasSettings(t *testing.T, nas apiclient.NasStorageSettings) apiclient.StorageSettings {
	t.Helper()

	nas.Type = apiclient.NasStorageSettingsTypeNAS

	return toSettings(t, func(settings *apiclient.StorageSettings) error {
		return settings.FromNasStorageSettings(nas)
	})
}

func newAzureBlobSettings(t *testing.T, azureBlob apiclient.AzureBlobStorageSettings) apiclient.StorageSettings {
	t.Helper()

	azureBlob.Type = apiclient.AzureBlobStorageSettingsTypeAZUREBLOB

	return toSettings(t, func(settings *apiclient.StorageSettings) error {
		return settings.FromAzureBlobStorageSettings(azureBlob)
	})
}

func newFtpSettings(t *testing.T, ftp apiclient.FtpStorageSettings) apiclient.StorageSettings {
	t.Helper()

	ftp.Type = apiclient.FtpStorageSettingsTypeFTP

	return toSettings(t, func(settings *apiclient.StorageSettings) error {
		return settings.FromFtpStorageSettings(ftp)
	})
}

func newSftpSettings(t *testing.T, sftp apiclient.SftpStorageSettings) apiclient.StorageSettings {
	t.Helper()

	sftp.Type = apiclient.SftpStorageSettingsTypeSFTP

	return toSettings(t, func(settings *apiclient.StorageSettings) error {
		return settings.FromSftpStorageSettings(sftp)
	})
}
