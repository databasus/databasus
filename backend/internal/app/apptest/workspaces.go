package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/workspaces"
)

// CreateWorkspace creates a workspace owned by the session's user through the API and fails the
// test unless it succeeds.
func CreateWorkspace(tb testing.TB, owner Session, name string) apiclient.Workspace {
	tb.Helper()

	response, err := owner.Client.CreateWorkspaceWithResponse(tb.Context(), apiclient.WorkspaceCreate{Name: name})
	if err != nil {
		tb.Fatalf("create the workspace %s: %v", name, err)
	}

	if response.StatusCode() != http.StatusCreated {
		tb.Fatalf("create the workspace %s answered %d: %s", name, response.StatusCode(), response.Body)
	}

	return *response.JSON201
}

// AddWorkspaceMember adds the address to the workspace through the API, inviting a user when the
// address has none, and fails the test unless it succeeds.
func AddWorkspaceMember(
	tb testing.TB,
	actor Session,
	workspaceID uuid.UUID,
	email string,
	role apiclient.AssignableWorkspaceRole,
) apiclient.AddedWorkspaceMember {
	tb.Helper()

	response, err := actor.Client.AddWorkspaceMemberWithResponse(tb.Context(), apiclient.WorkspaceMemberAddition{
		WorkspaceId: workspaceID,
		Email:       openapi_types.Email(email),
		Role:        role,
	})
	if err != nil {
		tb.Fatalf("add %s to the workspace: %v", email, err)
	}

	if response.StatusCode() != http.StatusCreated {
		tb.Fatalf("add %s to the workspace answered %d: %s", email, response.StatusCode(), response.Body)
	}

	return *response.JSON201
}

// SubscribeToWorkspaceDeleting adds a handler after the ones internal/app subscribed, the way a
// higher module would.
func (a *App) SubscribeToWorkspaceDeleting(handler workspaces.WorkspaceDeletingHandler) {
	a.app.SubscribeToWorkspaceDeleting(handler)
}
