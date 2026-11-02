package workspaces_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestCreateWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("makes the creator the owner when the instance lets members create workspaces", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)

		response, err := member.Client.CreateWorkspaceWithResponse(
			t.Context(),
			apiclient.WorkspaceCreate{Name: "Analytics"},
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, "Analytics", response.JSON201.Name)
		require.NotNil(t, response.JSON201.CurrentUserRole)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, *response.JSON201.CurrentUserRole)
		ownerMember := findMemberOfUser(t, listAllMembers(t, member.Client, response.JSON201.Id), member.UserID)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, ownerMember.Role)
	})

	t.Run("returns 403 when the instance does not let members create workspaces", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		forbidMemberWorkspaces(t, admin)

		response, err := member.Client.CreateWorkspaceWithResponse(
			t.Context(),
			apiclient.WorkspaceCreate{Name: "Analytics"},
		)

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Empty(t, listAllWorkspaces(t, admin.Client, 100))
	})

	t.Run("lets an instance admin create a workspace when members may not", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		forbidMemberWorkspaces(t, admin)

		response, err := admin.Client.CreateWorkspaceWithResponse(
			t.Context(),
			apiclient.WorkspaceCreate{Name: "Platform"},
		)

		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
	})

	t.Run("returns 400 when the name is empty", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response, err := admin.Client.CreateWorkspaceWithResponse(t.Context(), apiclient.WorkspaceCreate{Name: ""})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response, err := app.NewClient(t).CreateWorkspaceWithResponse(t.Context(), apiclient.WorkspaceCreate{
			Name: "Analytics",
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})
}

func TestListWorkspaces(t *testing.T) {
	t.Parallel()

	t.Run("lists only the workspaces the user is a member of with the user's role", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		otherOwner := tm.app.CreateNonAdminUser(t, tm.admin)
		apptest.CreateWorkspace(t, otherOwner, "Other team")
		viewer, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		listedWorkspaces := listAllWorkspaces(t, viewer.Client, 20)

		require.Len(t, listedWorkspaces, 1)
		assert.Equal(t, tm.workspace.Id, listedWorkspaces[0].Id)
		require.NotNil(t, listedWorkspaces[0].CurrentUserRole)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEVIEWER, *listedWorkspaces[0].CurrentUserRole)
	})

	t.Run("lists every workspace for an instance admin without a role where they are not a member", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.CreateNonAdminUser(t, admin)
		ownedWorkspace := apptest.CreateWorkspace(t, admin, "A owned")
		foreignWorkspace := apptest.CreateWorkspace(t, member, "B foreign")

		listedWorkspaces := listAllWorkspaces(t, admin.Client, 20)

		require.Len(t, listedWorkspaces, 2)
		assert.Equal(t, ownedWorkspace.Id, listedWorkspaces[0].Id)
		require.NotNil(t, listedWorkspaces[0].CurrentUserRole)
		assert.Equal(t, apiclient.WorkspaceRoleWORKSPACEOWNER, *listedWorkspaces[0].CurrentUserRole)
		assert.Equal(t, foreignWorkspace.Id, listedWorkspaces[1].Id)
		assert.Nil(t, listedWorkspaces[1].CurrentUserRole)
	})

	t.Run("lists every workspace once in the order of one page when following the cursor", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		createdNames := []string{"Delta", "alpha", "Charlie", "Bravo", "Bravo", "Echo"}
		for _, name := range createdNames {
			apptest.CreateWorkspace(t, admin, name)
		}

		pagedWorkspaces := listAllWorkspaces(t, admin.Client, 2)

		listedNames := make([]string, 0, len(pagedWorkspaces))
		for _, listedWorkspace := range pagedWorkspaces {
			listedNames = append(listedNames, listedWorkspace.Name)
		}

		assert.ElementsMatch(t, createdNames, listedNames)
		assert.Equal(t, listAllWorkspaces(t, admin.Client, 100), pagedWorkspaces)
	})

	t.Run("returns 400 when the cursor is not one of this list", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response, err := admin.Client.ListWorkspacesWithResponse(t.Context(), &apiclient.ListWorkspacesParams{
			Limit: nil,
			After: new("not-a-cursor"),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})
}

func TestGetWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("returns 404 when the user is not a member of the workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)

		response := getWorkspace(t, stranger, tm.workspace.Id)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("answers a non-member the same as a missing workspace", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)
		missingWorkspaceID := uuid.New()

		existingResponse := getWorkspace(t, stranger, tm.workspace.Id)
		missingResponse := getWorkspace(t, stranger, missingWorkspaceID)

		requireProblem(t, existingResponse.StatusCode(), existingResponse.Body, http.StatusNotFound,
			apiclient.ProblemCodeNotFound)
		assert.Equal(t,
			strings.ReplaceAll(string(missingResponse.Body), missingWorkspaceID.String(), tm.workspace.Id.String()),
			string(existingResponse.Body))
	})

	t.Run("lets a viewer read the workspace and list its members", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		viewer, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		response := getWorkspace(t, viewer, tm.workspace.Id)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, tm.workspace.Name, response.JSON200.Name)
		assert.Len(t, listAllMembers(t, viewer.Client, tm.workspace.Id), 2)
	})

	t.Run("lets an instance admin outside the workspace read, rename and list its members", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		readResponse := getWorkspace(t, tm.admin, tm.workspace.Id)
		renameResponse := renameWorkspace(t, tm.admin, tm.workspace.Id, "Renamed by the instance admin")

		require.Equal(t, http.StatusOK, readResponse.StatusCode(), string(readResponse.Body))
		assert.Nil(t, readResponse.JSON200.CurrentUserRole)
		require.Equal(t, http.StatusOK, renameResponse.StatusCode(), string(renameResponse.Body))
		assert.Equal(t, "Renamed by the instance admin", getWorkspace(t, tm.owner, tm.workspace.Id).JSON200.Name)
		assert.Len(t, listAllMembers(t, tm.admin.Client, tm.workspace.Id), 1)
	})
}

func TestUpdateWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("renames the workspace when an admin of it renames it", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)

		response := renameWorkspace(t, workspaceAdmin, tm.workspace.Id, "Renamed")

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, "Renamed", getWorkspace(t, tm.owner, tm.workspace.Id).JSON200.Name)
	})

	roleCases := []struct {
		name string
		role apiclient.AssignableWorkspaceRole
	}{
		{
			name: "returns 403 and keeps the name when a viewer renames the workspace",
			role: apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
		},
		{
			name: "returns 403 and keeps the name when a member renames the workspace",
			role: apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		},
	}
	for _, roleCase := range roleCases {
		t.Run(roleCase.name, func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			lowRoleUser, _ := tm.addMember(t, roleCase.role)

			response := renameWorkspace(t, lowRoleUser, tm.workspace.Id, "Renamed")

			requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
				apiclient.ProblemCodePermissionDenied)
			assert.Equal(t, tm.workspace.Name, getWorkspace(t, tm.owner, tm.workspace.Id).JSON200.Name)
		})
	}

	t.Run("returns 400 when the name is longer than 255 characters", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response := renameWorkspace(t, tm.owner, tm.workspace.Id, strings.Repeat("n", 256))

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})
}

func forbidMemberWorkspaces(t *testing.T, admin apptest.Session) {
	t.Helper()

	response, err := admin.Client.UpdateInstanceSettingsWithResponse(t.Context(), apiclient.InstanceSettingsUpdate{
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: false,
		IsTwoFactorAuthRequired:           false,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}
