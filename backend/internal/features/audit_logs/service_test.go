package audit_logs

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs_models "databasus-backend/internal/features/audit_logs/models"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/modules/users"
)

func TestWriteAuditLog(t *testing.T) {
	t.Run("stores the author's email and name when the entry has a user ID", func(t *testing.T) {
		author := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Storage created: nightly-backups",
			UserID:      &author.UserID,
			WorkspaceID: &workspaceID,
		})

		storedEvent := getSingleWorkspaceEvent(t, workspaceID)
		assert.Equal(t, "Storage created: nightly-backups", storedEvent.Message)
		require.NotNil(t, storedEvent.Author)
		assert.Equal(t, author.UserID, storedEvent.Author.UserID)
		assert.Equal(t, author.Email, storedEvent.Author.Email)
		assert.NotEmpty(t, storedEvent.Author.Name)
	})

	t.Run("keeps the author's email and name when the user changes both afterwards", func(t *testing.T) {
		author := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		authorBeforeChange := users_testing.GetTestUser(t.Context(), author.UserID)
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Storage created: nightly-backups",
			UserID:      &author.UserID,
			WorkspaceID: &workspaceID,
		})
		renameUser(t, author.UserID)

		storedEvent := getSingleWorkspaceEvent(t, workspaceID)
		require.NotNil(t, storedEvent.Author)
		assert.Equal(t, authorBeforeChange.Email, storedEvent.Author.Email)
		assert.Equal(t, authorBeforeChange.Name, storedEvent.Author.Name)
	})

	t.Run("stores no author when the entry has no user ID", func(t *testing.T) {
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Backup failed: payments",
			UserID:      nil,
			WorkspaceID: &workspaceID,
		})

		storedEvent := getSingleWorkspaceEvent(t, workspaceID)
		assert.Nil(t, storedEvent.Author)
	})

	t.Run("stores the event without an author when the account no longer exists", func(t *testing.T) {
		removedUser := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		users_testing.DeleteTestUser(t.Context(), removedUser.UserID)
		workspaceID := uuid.New()

		GetAuditLogService().WriteAuditLog(t.Context(), audit_logs_models.AuditEntry{
			Message:     "Storage deleted: nightly-backups",
			UserID:      &removedUser.UserID,
			WorkspaceID: &workspaceID,
		})

		storedEvent := getSingleWorkspaceEvent(t, workspaceID)
		assert.Equal(t, "Storage deleted: nightly-backups", storedEvent.Message)
		assert.Nil(t, storedEvent.Author)
	})
}

func getSingleWorkspaceEvent(t *testing.T, workspaceID uuid.UUID) audit.RecordedEvent {
	t.Helper()

	listedEvents, err := ListTestWorkspaceEvents(t.Context(), workspaceID)
	require.NoError(t, err)
	require.Len(t, listedEvents, 1)

	return listedEvents[0]
}

func renameUser(t *testing.T, userID uuid.UUID) {
	t.Helper()

	changedName := "Renamed " + uuid.NewString()
	changedEmail := "renamed-" + uuid.NewString() + "@example.com"
	users_testing.SetTestUserName(t.Context(), userID, changedName)
	users_testing.SetTestUserEmail(t.Context(), userID, changedEmail)
}
