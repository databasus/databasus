package workspaces_bridge_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

func TestIsActorAllowed(t *testing.T) {
	t.Run("lets a member edit the content and refuses a viewer and a non-member", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		owner := app.CreateNonAdminUser(t, admin)
		workspace := apptest.CreateWorkspace(t, owner, "Content")
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		viewer := app.SignUp(t, apptest.NewTestEmail("viewer"))
		stranger := app.SignUp(t, apptest.NewTestEmail("stranger"))
		apptest.AddWorkspaceMember(
			t,
			owner,
			workspace.Id,
			member.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEMEMBER,
		)
		apptest.AddWorkspaceMember(
			t,
			owner,
			workspace.Id,
			viewer.Email,
			apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
		)
		access := workspaces_bridge.GetAccess()

		isMemberAllowed, memberErr := access.IsActorAllowed(t.Context(),
			users.Actor{UserID: member.UserID, Role: users.RoleMember}, workspace.Id, workspaces.ActionEditContent)
		isViewerAllowed, viewerErr := access.IsActorAllowed(t.Context(),
			users.Actor{UserID: viewer.UserID, Role: users.RoleMember}, workspace.Id, workspaces.ActionEditContent)
		isStrangerAllowed, strangerErr := access.IsActorAllowed(t.Context(),
			users.Actor{UserID: stranger.UserID, Role: users.RoleMember}, workspace.Id, workspaces.ActionView)

		require.NoError(t, memberErr)
		require.NoError(t, viewerErr)
		require.NoError(t, strangerErr)
		assert.True(t, isMemberAllowed)
		assert.False(t, isViewerAllowed)
		assert.False(t, isStrangerAllowed)
	})

	t.Run("returns an error when the role lookup fails", func(t *testing.T) {
		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		owner := app.CreateNonAdminUser(t, admin)
		workspace := apptest.CreateWorkspace(t, owner, "Content")
		cancelledContext, cancel := context.WithCancel(t.Context())
		cancel()

		isAllowed, err := workspaces_bridge.GetAccess().IsActorAllowed(cancelledContext,
			users.Actor{UserID: owner.UserID, Role: users.RoleMember}, workspace.Id, workspaces.ActionView)

		require.Error(t, err)
		assert.False(t, isAllowed)
	})
}
