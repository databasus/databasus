package workspaces_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

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

// team is an instance with a workspace owned by a user who is not an instance admin.
type team struct {
	app       *apptest.App
	admin     apptest.Session
	owner     apptest.Session
	workspace apiclient.Workspace
}

func newTeam(t *testing.T) team {
	t.Helper()

	app := apptest.New(t)
	admin := app.CreateAdmin(t)
	owner := app.CreateNonAdminUser(t, admin)

	return team{
		app:       app,
		admin:     admin,
		owner:     owner,
		workspace: apptest.CreateWorkspace(t, owner, "Team "+uuid.NewString()[:8]),
	}
}

// addMember signs up a user and adds them to the team's workspace with the role, a second after
// the previous change, so the member list has an order to keep.
func (tm team) addMember(
	t *testing.T,
	role apiclient.AssignableWorkspaceRole,
) (apptest.Session, apiclient.AddedWorkspaceMember) {
	t.Helper()

	tm.app.Clock.Advance(time.Second)
	user := tm.app.SignUp(t, apptest.NewTestEmail("member"))
	addedMember := apptest.AddWorkspaceMember(t, tm.owner, tm.workspace.Id, user.Email, role)

	return user, addedMember
}

func (tm team) getOwnerMember(t *testing.T) apiclient.WorkspaceMember {
	t.Helper()

	return findMemberOfUser(t, listAllMembers(t, tm.owner.Client, tm.workspace.Id), tm.owner.UserID)
}

func listAllMembers(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	workspaceID uuid.UUID,
) []apiclient.WorkspaceMember {
	t.Helper()

	var listedMembers []apiclient.WorkspaceMember

	var after *string

	for {
		response, err := client.ListWorkspaceMembersWithResponse(t.Context(), &apiclient.ListWorkspaceMembersParams{
			WorkspaceId: workspaceID,
			Limit:       new(100),
			After:       after,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		listedMembers = append(listedMembers, response.JSON200.Items...)

		if response.JSON200.NextCursor == nil {
			return listedMembers
		}

		after = response.JSON200.NextCursor
	}
}

func findMemberOfUser(
	t *testing.T,
	listedMembers []apiclient.WorkspaceMember,
	userID uuid.UUID,
) apiclient.WorkspaceMember {
	t.Helper()

	for _, listedMember := range listedMembers {
		if listedMember.UserId == userID {
			return listedMember
		}
	}

	require.FailNow(t, "the user is not a member", userID.String())

	return apiclient.WorkspaceMember{}
}

func countOwners(listedMembers []apiclient.WorkspaceMember) int {
	ownerCount := 0

	for _, listedMember := range listedMembers {
		if listedMember.Role == apiclient.WorkspaceRoleWORKSPACEOWNER {
			ownerCount++
		}
	}

	return ownerCount
}

func listAllWorkspaces(t *testing.T, client *apiclient.ClientWithResponses, limit int) []apiclient.Workspace {
	t.Helper()

	var listedWorkspaces []apiclient.Workspace

	var after *string

	for {
		response, err := client.ListWorkspacesWithResponse(t.Context(), &apiclient.ListWorkspacesParams{
			Limit: &limit,
			After: after,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

		listedWorkspaces = append(listedWorkspaces, response.JSON200.Items...)

		if response.JSON200.NextCursor == nil {
			return listedWorkspaces
		}

		after = response.JSON200.NextCursor
	}
}

func getWorkspace(t *testing.T, session apptest.Session, workspaceID uuid.UUID) *apiclient.GetWorkspaceResponse {
	t.Helper()

	response, err := session.Client.GetWorkspaceWithResponse(t.Context(), workspaceID)
	require.NoError(t, err)

	return response
}

func renameWorkspace(
	t *testing.T,
	session apptest.Session,
	workspaceID uuid.UUID,
	name string,
) *apiclient.UpdateWorkspaceResponse {
	t.Helper()

	response, err := session.Client.UpdateWorkspaceWithResponse(t.Context(), workspaceID, apiclient.WorkspaceUpdate{
		Name: name,
	})
	require.NoError(t, err)

	return response
}

// sendRawRequest sends a body the contract cannot describe, such as an address that is not one.
func sendRawRequest(t *testing.T, app *apptest.App, session apptest.Session, method, path, body string) (int, []byte) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, app.URL+"/api/v1"+path,
		bytes.NewBufferString(body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.Token)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(response.Body)
	require.NoError(t, err)

	return response.StatusCode, responseBody.Bytes()
}

func createWebhookNotifier(t *testing.T, session apptest.Session, workspaceID uuid.UUID) {
	t.Helper()

	var settings apiclient.NotifierSettings
	require.NoError(t, settings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
		Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         "https://example.com/hooks/databasus",
		WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
		BodyTemplate:       nil,
		Headers:            []apiclient.WebhookHeader{},
		AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
	}))

	apptest.CreateNotifier(t, session, workspaceID, settings)
}

// countContent lists the workspace's content through the contract and counts each kind.
func countContent(t *testing.T, session apptest.Session, workspaceID uuid.UUID) workspaceContent {
	t.Helper()

	return workspaceContent{
		databaseCount: countDatabases(t, session, workspaceID),
		notifierCount: countNotifiers(t, session, workspaceID),
		storageCount:  countStorages(t, session, workspaceID),
	}
}

type workspaceContent struct {
	databaseCount int
	notifierCount int
	storageCount  int
}

func countDatabases(t *testing.T, session apptest.Session, workspaceID uuid.UUID) int {
	t.Helper()

	response, err := session.Client.ListDatabasesWithResponse(t.Context(), &apiclient.ListDatabasesParams{
		WorkspaceId: workspaceID,
		NotifierId:  nil,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return len(response.JSON200.Items)
}

func countNotifiers(t *testing.T, session apptest.Session, workspaceID uuid.UUID) int {
	t.Helper()

	response, err := session.Client.ListNotifiersWithResponse(t.Context(), &apiclient.ListNotifiersParams{
		WorkspaceId: workspaceID,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return len(response.JSON200.Items)
}

func countStorages(t *testing.T, session apptest.Session, workspaceID uuid.UUID) int {
	t.Helper()

	response, err := session.Client.ListStoragesWithResponse(t.Context(), &apiclient.ListStoragesParams{
		WorkspaceId: workspaceID,
		Limit:       nil,
		After:       nil,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return len(response.JSON200.Items)
}
