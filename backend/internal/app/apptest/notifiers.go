package apptest

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/notifiers"
)

// NotifierAttachment names the database that starts using the notifier.
type NotifierAttachment struct {
	DatabaseID uuid.UUID
	NotifierID uuid.UUID
}

// CreateNotifier creates a notifier in the workspace through the API and fails the test unless it
// succeeds.
func CreateNotifier(
	tb testing.TB,
	member Session,
	workspaceID uuid.UUID,
	settings apiclient.NotifierSettings,
) apiclient.Notifier {
	tb.Helper()

	response, err := member.Client.CreateNotifierWithResponse(tb.Context(), apiclient.NotifierCreate{
		WorkspaceId: workspaceID,
		Name:        "Notifier " + uuid.NewString()[:8],
		Settings:    settings,
	})
	if err != nil {
		tb.Fatalf("create the notifier: %v", err)
	}

	if response.StatusCode() != http.StatusCreated {
		tb.Fatalf("create the notifier answered %d: %s", response.StatusCode(), response.Body)
	}

	return *response.JSON201
}

// AttachNotifierToDatabase writes the attachment directly, because no contract operation attaches a
// notifier to a database yet.
func (a *App) AttachNotifierToDatabase(tb testing.TB, attachment NotifierAttachment) {
	tb.Helper()

	a.ExecuteSQL(tb, "INSERT INTO database_notifiers (database_id, notifier_id) VALUES (?, ?)",
		attachment.DatabaseID, attachment.NotifierID)
}

// BeginNotifierAttachment writes the attachment in a transaction it leaves open, like an attach of
// the old code that has not committed yet, and returns the function that commits it.
func (a *App) BeginNotifierAttachment(tb testing.TB, attachment NotifierAttachment) func() {
	tb.Helper()

	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the database of the test application: %v", err)
	}

	transaction := db.WithContext(tb.Context()).Begin()
	tb.Cleanup(func() {
		transaction.Rollback()
		closeConnections(db)
	})

	err = transaction.Exec("INSERT INTO database_notifiers (database_id, notifier_id) VALUES (?, ?)",
		attachment.DatabaseID, attachment.NotifierID).Error
	if err != nil {
		tb.Fatalf("insert the attachment: %v", err)
	}

	return func() {
		err := transaction.Commit().Error
		if err != nil {
			tb.Fatalf("commit the attachment: %v", err)
		}
	}
}

// Notify sends a notification to the notifiers the way the old code does when a backup or a
// healthcheck ends. It returns at once, while the deliveries run.
func (a *App) Notify(tb testing.TB, notifierIDs []uuid.UUID, notification notifiers.Notification) {
	tb.Helper()

	a.app.Notify(tb.Context(), notifierIDs, notification)
}
