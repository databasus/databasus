package instancebackups_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
)

func TestGetInstanceBackupSetupReminder(t *testing.T) {
	t.Parallel()

	t.Run("suggests no setup when the instance has only an empty workspace", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		apptest.CreateWorkspace(t, owner.RootAdmin, "Empty")

		status := getStatus(t, owner.RootAdmin)

		assert.False(t, status.Reminder.IsEligible)
	})

	t.Run("suggests setup when a storage exists in a workspace of another user", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		apptest.CreateWorkspace(t, owner.RootAdmin, "Empty")
		otherWorkspace := apptest.CreateWorkspace(t, owner.OtherAdmin, "Other")
		statusBeforeStorage := getStatus(t, owner.RootAdmin)

		owner.App.CreateLocalStorage(t, owner.OtherAdmin, otherWorkspace.Id)
		statusAfterStorage := getStatus(t, owner.RootAdmin)

		assert.False(t, statusBeforeStorage.Reminder.IsEligible)
		assert.True(t, statusAfterStorage.Reminder.IsEligible)
	})

	t.Run("suggests setup when a database exists in a workspace of another user", func(t *testing.T) {
		t.Parallel()
		owner := newInstanceOwner(t)
		otherWorkspace := apptest.CreateWorkspace(t, owner.OtherAdmin, "Other")
		statusBeforeDatabase := getStatus(t, owner.RootAdmin)

		owner.App.InsertDatabase(t, otherWorkspace.Id)
		statusAfterDatabase := getStatus(t, owner.RootAdmin)

		assert.False(t, statusBeforeDatabase.Reminder.IsEligible)
		assert.True(t, statusAfterDatabase.Reminder.IsEligible)
	})
}
