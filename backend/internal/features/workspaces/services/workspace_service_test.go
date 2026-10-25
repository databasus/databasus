package workspaces_services_test

import (
	"context"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

var allActions = []workspaces.Action{
	workspaces.ActionView,
	workspaces.ActionEditContent,
	workspaces.ActionUpdateWorkspace,
	workspaces.ActionManageMembers,
	workspaces.ActionManageAdmins,
	workspaces.ActionTransferOwnership,
	workspaces.ActionDeleteWorkspace,
}

func TestIsActorAllowed(t *testing.T) {
	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
	)
	workspaceService := workspaces_services.GetWorkspaceService()

	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Roles "+uuid.NewString()[:8], owner, router)
	t.Cleanup(func() { workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router) })

	addMember := func(role workspaces.Role) users.Actor {
		member := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		workspaces_testing.AddMemberToWorkspace(workspace, member, role, owner.Token, router)

		return users.Actor{UserID: member.UserID, Role: users.RoleMember}
	}

	actorsByRole := map[workspaces.Role]users.Actor{
		workspaces.RoleOwner:  {UserID: owner.UserID, Role: users.RoleMember},
		workspaces.RoleAdmin:  addMember(workspaces.RoleAdmin),
		workspaces.RoleMember: addMember(workspaces.RoleMember),
		workspaces.RoleViewer: addMember(workspaces.RoleViewer),
	}

	allowedActionsByRole := map[workspaces.Role][]workspaces.Action{
		workspaces.RoleOwner: allActions,
		workspaces.RoleAdmin: {
			workspaces.ActionView,
			workspaces.ActionEditContent,
			workspaces.ActionUpdateWorkspace,
			workspaces.ActionManageMembers,
		},
		workspaces.RoleMember: {workspaces.ActionView, workspaces.ActionEditContent},
		workspaces.RoleViewer: {workspaces.ActionView},
	}

	t.Run("allows each role only the actions of its row", func(t *testing.T) {
		for role, actor := range actorsByRole {
			for _, action := range allActions {
				isAllowed, err := workspaceService.IsActorAllowed(t.Context(), actor, workspace.ID, action)

				require.NoError(t, err)
				assert.Equal(
					t,
					slices.Contains(allowedActionsByRole[role], action),
					isAllowed,
					"role %s, action %s",
					role,
					action,
				)
			}
		}
	})

	t.Run("allows every action when actor is an instance admin outside the workspace", func(t *testing.T) {
		instanceAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
		adminActor := users.Actor{UserID: instanceAdmin.UserID, Role: users.RoleAdmin}

		for _, action := range allActions {
			isAllowed, err := workspaceService.IsActorAllowed(t.Context(), adminActor, workspace.ID, action)

			require.NoError(t, err)
			assert.True(t, isAllowed, "action %s", action)
		}
	})

	t.Run("allows no action when actor is not a member", func(t *testing.T) {
		stranger := users_testing.CreateTestUser(t.Context(), users.RoleMember)
		strangerActor := users.Actor{UserID: stranger.UserID, Role: users.RoleMember}

		for _, action := range allActions {
			isAllowed, err := workspaceService.IsActorAllowed(t.Context(), strangerActor, workspace.ID, action)

			require.NoError(t, err)
			assert.False(t, isAllowed, "action %s", action)
		}
	})

	t.Run("returns an error when the role lookup fails", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(t.Context())
		cancel()

		isAllowed, err := workspaceService.IsActorAllowed(
			cancelledCtx,
			actorsByRole[workspaces.RoleViewer],
			workspace.ID,
			workspaces.ActionView,
		)

		require.Error(t, err)
		assert.False(t, isAllowed)
	})
}
