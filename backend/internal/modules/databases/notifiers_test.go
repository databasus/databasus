package databases_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

var errAttachmentNotWaiting = errors.New("no attachment waits for the lock on the notifier")

//nolint:tparallel // The subtests share one source server.
func TestDatabaseNotifiers(t *testing.T) {
	t.Parallel()

	server := containers.StartPostgres(t, "postgres:16")

	t.Run("returns 400 invalid_request naming the notifiers and creates nothing with a notifier of another workspace",
		func(t *testing.T) {
			tm := newTeam(t)
			otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Other "+uuid.NewString()[:8])
			foreignNotifier := apptest.CreateNotifier(t, tm.owner, otherWorkspace.Id, newWebhookNotifierSettings(t))

			response, err := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Orders",
				NotifierIds: []uuid.UUID{foreignNotifier.Id},
				Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
			})

			require.NoError(t, err)
			requireInvalidField(t, response.StatusCode(), response.Body, "notifierIds")
			assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
		})

	t.Run("returns 400 invalid_request and keeps the notifiers when a change adds a notifier of another workspace",
		func(t *testing.T) {
			tm := newTeam(t)
			ownNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
			otherWorkspace := apptest.CreateWorkspace(t, tm.owner, "Other "+uuid.NewString()[:8])
			foreignNotifier := apptest.CreateNotifier(t, tm.owner, otherWorkspace.Id, newWebhookNotifierSettings(t))
			createdDatabase := createDatabase(t, tm.owner, apiclient.DatabaseCreate{
				WorkspaceId: tm.workspace.Id,
				Name:        "Orders",
				NotifierIds: []uuid.UUID{ownNotifier.Id},
				Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
			})
			update := toUpdate(createdDatabase)
			update.NotifierIds = []uuid.UUID{ownNotifier.Id, foreignNotifier.Id}

			response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

			requireInvalidField(t, response.StatusCode(), response.Body, "notifierIds")
			assert.Equal(t, []uuid.UUID{ownNotifier.Id}, getDatabase(t, tm.owner, createdDatabase.Id).NotifierIds)
		})

	t.Run("returns 400 invalid_request and creates nothing when the notifier moves away during the create",
		func(t *testing.T) {
			tm := newTeam(t)
			targetWorkspace := apptest.CreateWorkspace(t, tm.owner, "Target "+uuid.NewString()[:8])
			movingNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
			commitTransfer := tm.app.HoldNotifierTransfer(t, apptest.NotifierTransfer{
				NotifierID:        movingNotifier.Id,
				TargetWorkspaceID: targetWorkspace.Id,
			})

			settings := fromLogicalSettings(t, newLogicalSettings(server))
			createResponses := make(chan *apiclient.CreateDatabaseResponse, 1)
			go func() {
				response, _ := tm.owner.Client.CreateDatabaseWithResponse(t.Context(), apiclient.DatabaseCreate{
					WorkspaceId: tm.workspace.Id,
					Name:        "Orders",
					NotifierIds: []uuid.UUID{movingNotifier.Id},
					Settings:    settings,
				})
				createResponses <- response
			}()

			testwait.Until(t, "the create waits for the transfer's lock on the notifier", func(context.Context) error {
				if !isAttachmentWaitingForLock(t, tm) {
					return errAttachmentNotWaiting
				}

				return nil
			})
			commitTransfer()
			response := <-createResponses

			require.NotNil(t, response)
			requireInvalidField(t, response.StatusCode(), response.Body, "notifierIds")
			assert.Empty(t, listDatabases(t, tm.owner, tm.workspace.Id))
		})

	t.Run("replaces the notifiers with those of the request", func(t *testing.T) {
		tm := newTeam(t)
		firstNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
		secondNotifier := apptest.CreateNotifier(t, tm.owner, tm.workspace.Id, newWebhookNotifierSettings(t))
		createdDatabase := createDatabase(t, tm.owner, apiclient.DatabaseCreate{
			WorkspaceId: tm.workspace.Id,
			Name:        "Orders",
			NotifierIds: []uuid.UUID{firstNotifier.Id},
			Settings:    fromLogicalSettings(t, newLogicalSettings(server)),
		})
		update := toUpdate(createdDatabase)
		update.NotifierIds = []uuid.UUID{secondNotifier.Id}

		response := updateDatabase(t, tm.owner, createdDatabase.Id, update)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, []uuid.UUID{secondNotifier.Id}, getDatabase(t, tm.owner, createdDatabase.Id).NotifierIds)
	})
}

func isAttachmentWaitingForLock(t *testing.T, tm team) bool {
	t.Helper()

	var waitingPIDs []int
	tm.app.QuerySQL(t, &waitingPIDs, `
		SELECT pid FROM pg_stat_activity
		WHERE datname = current_database() AND wait_event_type = 'Lock' AND query LIKE '%databases_notifiers%'
	`)

	return len(waitingPIDs) > 0
}
