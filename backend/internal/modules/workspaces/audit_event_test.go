package workspaces_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestListAuditEvents(t *testing.T) {
	t.Parallel()

	t.Run("returns 403 when a user who is not an instance admin reads the global view", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.ListAuditEventsWithResponse(t.Context(), &apiclient.ListAuditEventsParams{})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("lists only the user's own events when the user reads their own view", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		apptest.CreateWorkspace(t, tm.admin, "Admin's workspace")

		ownEvents := apptest.ListAllAuditEvents(t, tm.owner, apiclient.ListAuditEventsParams{
			AuthorUserId: &tm.owner.UserID,
		})

		require.NotEmpty(t, ownEvents)
		for _, event := range ownEvents {
			require.NotNil(t, event.Author)
			assert.Equal(t, tm.owner.UserID, event.Author.UserId)
		}
	})

	t.Run("returns 403 when a user who is not an instance admin reads another user's view", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)

		response, err := tm.owner.Client.ListAuditEventsWithResponse(t.Context(), &apiclient.ListAuditEventsParams{
			AuthorUserId: &tm.admin.UserID,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("lists only the workspace's events when a viewer reads the workspace's view", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		apptest.CreateWorkspace(t, tm.owner, "Other workspace")
		viewer, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)

		workspaceEvents := apptest.ListAllAuditEvents(t, viewer, apiclient.ListAuditEventsParams{
			WorkspaceId: &tm.workspace.Id,
		})

		require.NotEmpty(t, workspaceEvents)
		assertAllInWorkspace(t, workspaceEvents, tm.workspace.Id)
	})

	t.Run("lists the events of every workspace and outside any workspace when an instance admin reads the global view",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Other workspace")
			forbidMemberWorkspaces(t, tm.admin)

			globalEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{})

			assert.True(t, hasEventInWorkspace(globalEvents, tm.workspace.Id))
			assert.True(t, hasEventInWorkspace(globalEvents, otherWorkspace.Id))
			settingsEvent := findEventWithMessage(t, globalEvents, "isMemberAllowedToCreateWorkspaces: true -> false")
			assert.Nil(t, settingsEvent.Workspace)
		})

	t.Run("lists only that user's events when an instance admin reads another user's view", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		apptest.CreateWorkspace(t, tm.admin, "Admin's workspace")

		ownerEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{
			AuthorUserId: &tm.owner.UserID,
		})

		require.NotEmpty(t, ownerEvents)
		for _, event := range ownerEvents {
			require.NotNil(t, event.Author)
			assert.Equal(t, tm.owner.UserID, event.Author.UserId)
		}
	})

	t.Run("lists the workspace's events when an instance admin outside the workspace reads its view",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			apptest.CreateWorkspace(t, tm.admin, "Admin's workspace")

			workspaceEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{
				WorkspaceId: &tm.workspace.Id,
			})

			require.NotEmpty(t, workspaceEvents)
			assertAllInWorkspace(t, workspaceEvents, tm.workspace.Id)
		})

	t.Run("lists only that member's events in the workspace when a viewer narrows the view to one author",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			viewer, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER)
			workspaceAdmin, _ := tm.addMember(t, apiclient.AssignableWorkspaceRoleWORKSPACEADMIN)
			renameWorkspaceSuccessfully(t, workspaceAdmin, tm.workspace.Id, "Renamed by the admin")
			apptest.CreateWorkspace(t, workspaceAdmin, "Admin's own workspace")

			narrowedEvents := apptest.ListAllAuditEvents(t, viewer, apiclient.ListAuditEventsParams{
				WorkspaceId:  &tm.workspace.Id,
				AuthorUserId: &workspaceAdmin.UserID,
			})

			require.Len(t, narrowedEvents, 1)
			require.NotNil(t, narrowedEvents[0].Author)
			assert.Equal(t, workspaceAdmin.UserID, narrowedEvents[0].Author.UserId)
			assertAllInWorkspace(t, narrowedEvents, tm.workspace.Id)
		})

	t.Run("returns 404 when a user who is not a member reads the workspace's view", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		stranger := tm.app.CreateNonAdminUser(t, tm.admin)

		response, err := stranger.Client.ListAuditEventsWithResponse(t.Context(), &apiclient.ListAuditEventsParams{
			WorkspaceId: &tm.workspace.Id,
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})

	t.Run("returns 401 without a session", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)

		response, err := app.NewClient(t).ListAuditEventsWithResponse(t.Context(), &apiclient.ListAuditEventsParams{})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("returns 400 when the cursor is not one of this list", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response, err := admin.Client.ListAuditEventsWithResponse(t.Context(), &apiclient.ListAuditEventsParams{
			After: new("not-a-cursor"),
		})

		require.NoError(t, err)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
	})
}

func TestPageAuditEvents(t *testing.T) {
	t.Parallel()

	t.Run("lists every event once newest first when following the pages to the last", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		for _, name := range []string{"Alpha", "Bravo", "Charlie", "Delta", "Echo"} {
			app.Clock.Advance(time.Second)
			apptest.CreateWorkspace(t, admin, name)
		}

		pagedEvents := apptest.ListAllAuditEvents(t, admin, apiclient.ListAuditEventsParams{Limit: new(2)})

		assert.Equal(t, apptest.ListAllAuditEvents(t, admin, apiclient.ListAuditEventsParams{Limit: new(100)}),
			pagedEvents)
		assert.Equal(t, "Workspace created: Echo", pagedEvents[0].Message)
		assert.True(t, slices.IsSortedFunc(pagedEvents, func(newer, older apiclient.AuditEvent) int {
			return older.CreatedAt.Compare(newer.CreatedAt)
		}))
	})

	t.Run("continues after the last event of the first page when a new event is written while paging",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			var renamedWorkspace apiclient.Workspace
			for _, name := range []string{"Alpha", "Bravo", "Charlie", "Delta"} {
				app.Clock.Advance(time.Second)
				renamedWorkspace = apptest.CreateWorkspace(t, admin, name)
			}

			eventsBeforeWrite := apptest.ListAllAuditEvents(t, admin, apiclient.ListAuditEventsParams{})
			firstPage := listAuditEventPage(t, admin, apiclient.ListAuditEventsParams{Limit: new(2)})
			app.Clock.Advance(time.Second)
			renameWorkspaceSuccessfully(t, admin, renamedWorkspace.Id, "Renamed while paging")
			secondPage := listAuditEventPage(t, admin, apiclient.ListAuditEventsParams{
				Limit: new(2),
				After: firstPage.NextCursor,
			})

			require.NotEmpty(t, secondPage.Items)
			assert.Equal(t, eventsBeforeWrite[2].Id, secondPage.Items[0].Id)
			for _, firstPageEvent := range firstPage.Items {
				assert.False(t, hasEventWithID(secondPage.Items, firstPageEvent.Id))
			}
		})
}

func TestAuditEventWorkspace(t *testing.T) {
	t.Parallel()

	t.Run("shows the workspace's new name when the workspace is renamed after the event", func(t *testing.T) {
		t.Parallel()

		tm := newTeam(t)
		renameWorkspaceSuccessfully(t, tm.owner, tm.workspace.Id, "Renamed team")

		globalEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{})

		creationEvent := findEventWithMessage(t, globalEvents, "Workspace created: "+tm.workspace.Name)
		require.NotNil(t, creationEvent.Workspace)
		assert.Equal(t, tm.workspace.Id, creationEvent.Workspace.Id)
		assert.Equal(t, new("Renamed team"), creationEvent.Workspace.Name)
	})

	t.Run("shows the workspace's id without a name when the workspace is deleted after the event",
		func(t *testing.T) {
			t.Parallel()

			tm := newTeam(t)
			response, err := tm.owner.Client.DeleteWorkspaceWithResponse(t.Context(), tm.workspace.Id)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, response.StatusCode(), string(response.Body))

			globalEvents := apptest.ListAllAuditEvents(t, tm.admin, apiclient.ListAuditEventsParams{})

			creationEvent := findEventWithMessage(t, globalEvents, "Workspace created: "+tm.workspace.Name)
			require.NotNil(t, creationEvent.Workspace)
			assert.Equal(t, tm.workspace.Id, creationEvent.Workspace.Id)
			assert.Nil(t, creationEvent.Workspace.Name)
		})
}

func listAuditEventPage(
	t *testing.T,
	reader apptest.Session,
	filters apiclient.ListAuditEventsParams,
) apiclient.AuditEventPage {
	t.Helper()

	response, err := reader.Client.ListAuditEventsWithResponse(t.Context(), &filters)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

// renameWorkspaceSuccessfully fails the test unless the rename answers 200.
func renameWorkspaceSuccessfully(t *testing.T, actor apptest.Session, workspaceID uuid.UUID, name string) {
	t.Helper()

	response := renameWorkspace(t, actor, workspaceID, name)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func assertAllInWorkspace(t *testing.T, listedEvents []apiclient.AuditEvent, workspaceID uuid.UUID) {
	t.Helper()

	for _, event := range listedEvents {
		require.NotNil(t, event.Workspace, event.Message)
		assert.Equal(t, workspaceID, event.Workspace.Id, event.Message)
	}
}

func hasEventInWorkspace(listedEvents []apiclient.AuditEvent, workspaceID uuid.UUID) bool {
	return slices.ContainsFunc(listedEvents, func(event apiclient.AuditEvent) bool {
		return event.Workspace != nil && event.Workspace.Id == workspaceID
	})
}

func hasEventWithID(listedEvents []apiclient.AuditEvent, eventID uuid.UUID) bool {
	return slices.ContainsFunc(listedEvents, func(event apiclient.AuditEvent) bool { return event.Id == eventID })
}

func findEventWithMessage(t *testing.T, listedEvents []apiclient.AuditEvent, message string) apiclient.AuditEvent {
	t.Helper()

	eventIndex := slices.IndexFunc(listedEvents, func(event apiclient.AuditEvent) bool {
		return event.Message == message
	})
	require.GreaterOrEqual(t, eventIndex, 0, "no event %q", message)

	return listedEvents[eventIndex]
}
