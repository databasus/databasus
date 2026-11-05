package app_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestAuditEventsCleanup(t *testing.T) {
	t.Run("deletes an event older than one year and keeps a younger one", func(t *testing.T) {
		testApp := apptest.New(t)
		admin := testApp.CreateAdmin(t)
		expiringWorkspaceID := apptest.CreateWorkspace(t, admin, "expiring-"+uuid.NewString()).Id

		testApp.Clock.Advance(366 * 24 * time.Hour)
		keptWorkspaceID := apptest.CreateWorkspace(t, admin, "kept-"+uuid.NewString()).Id

		testApp.RunJobPass(t, "audit_events_cleanup")

		assert.Empty(t, listWorkspaceAuditMessages(t, admin, expiringWorkspaceID))
		assert.NotEmpty(t, listWorkspaceAuditMessages(t, admin, keptWorkspaceID))
	})
}

func listWorkspaceAuditMessages(t *testing.T, reader apptest.Session, workspaceID uuid.UUID) []string {
	t.Helper()

	workspaceEvents := apptest.ListAllAuditEvents(t, reader, apiclient.ListAuditEventsParams{WorkspaceId: &workspaceID})

	var messages []string
	for _, event := range workspaceEvents {
		messages = append(messages, event.Message)
	}

	return messages
}
