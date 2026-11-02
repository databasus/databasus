package workspaces_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

type unauthenticatedCall struct {
	name string
	call func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error)
}

//nolint:tparallel // The subtests share one application and its client.
func TestOperationsWithoutSession(t *testing.T) {
	t.Parallel()

	anyID := uuid.New()
	calls := []unauthenticatedCall{
		{
			name: "listWorkspaces",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.ListWorkspacesWithResponse(
					ctx,
					&apiclient.ListWorkspacesParams{Limit: nil, After: nil},
				)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "getWorkspace",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.GetWorkspaceWithResponse(ctx, anyID)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "updateWorkspace",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.UpdateWorkspaceWithResponse(
					ctx,
					anyID,
					apiclient.WorkspaceUpdate{Name: "Renamed"},
				)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "deleteWorkspace",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.DeleteWorkspaceWithResponse(ctx, anyID)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "transferWorkspaceOwnership",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.TransferWorkspaceOwnershipWithResponse(ctx, anyID,
					apiclient.OwnershipTransfer{NewOwnerMemberId: anyID})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "listWorkspaceMembers",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.ListWorkspaceMembersWithResponse(ctx,
					&apiclient.ListWorkspaceMembersParams{WorkspaceId: anyID, Limit: nil, After: nil})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "addWorkspaceMember",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.AddWorkspaceMemberWithResponse(ctx, apiclient.WorkspaceMemberAddition{
					WorkspaceId: anyID,
					Email:       openapi_types.Email("someone@example.com"),
					Role:        apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
				})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "changeWorkspaceMemberRole",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.ChangeWorkspaceMemberRoleWithResponse(ctx, anyID,
					apiclient.WorkspaceMemberRoleChange{Role: apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "removeWorkspaceMember",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.RemoveWorkspaceMemberWithResponse(ctx, anyID)
				return getStatusAndBody(response, err)
			},
		},
	}

	app := apptest.New(t)
	client := app.NewClient(t)

	for _, unauthenticated := range calls {
		t.Run("returns 401 from "+unauthenticated.name+" without a session", func(t *testing.T) {
			status, body, err := unauthenticated.call(t.Context(), client)

			require.NoError(t, err)
			requireProblem(t, status, body, http.StatusUnauthorized, apiclient.ProblemCodeUnauthenticated)
		})
	}
}

type contractResponse interface {
	StatusCode() int
	GetBody() []byte
}

func getStatusAndBody(response contractResponse, err error) (int, []byte, error) {
	if err != nil {
		return 0, nil, err
	}

	return response.StatusCode(), response.GetBody(), nil
}
