package notifiers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
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
	settings := newWebhookSettings(t, newPostWebhook("https://hooks.example.com"))
	calls := []unauthenticatedCall{
		{
			name: "listNotifiers",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.ListNotifiersWithResponse(ctx,
					&apiclient.ListNotifiersParams{WorkspaceId: anyID, Limit: nil, After: nil})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "createNotifier",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.CreateNotifierWithResponse(ctx,
					apiclient.NotifierCreate{WorkspaceId: anyID, Name: "Hook", Settings: settings})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "getNotifier",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.GetNotifierWithResponse(ctx, anyID)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "updateNotifier",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.UpdateNotifierWithResponse(ctx, anyID,
					apiclient.NotifierUpdate{Name: "Hook", Settings: settings})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "deleteNotifier",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.DeleteNotifierWithResponse(ctx, anyID)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "transferNotifier",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.TransferNotifierWithResponse(ctx, anyID,
					apiclient.NotifierTransfer{TargetWorkspaceId: anyID})
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "testNotifier",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.TestNotifierWithResponse(ctx, anyID)
				return getStatusAndBody(response, err)
			},
		},
		{
			name: "testNotifierSettings",
			call: func(ctx context.Context, client *apiclient.ClientWithResponses) (int, []byte, error) {
				response, err := client.TestNotifierSettingsWithResponse(ctx,
					apiclient.NotifierSettingsTest{WorkspaceId: anyID, NotifierId: nil, Settings: settings})
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
