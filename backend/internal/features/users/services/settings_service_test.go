package users_services_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
)

func TestCanInviteUsers(t *testing.T) {
	testCases := []struct {
		name                 string
		role                 users.Role
		setMemberInvitations func(ctx context.Context)
		isInvitationAllowed  bool
	}{
		{
			name:                 "allows an admin when member invitations are disabled",
			role:                 users.RoleAdmin,
			setMemberInvitations: users_testing.DisableMemberInvitations,
			isInvitationAllowed:  true,
		},
		{
			name:                 "allows a member when member invitations are enabled",
			role:                 users.RoleMember,
			setMemberInvitations: users_testing.EnableMemberInvitations,
			isInvitationAllowed:  true,
		},
		{
			name:                 "forbids a member when member invitations are disabled",
			role:                 users.RoleMember,
			setMemberInvitations: users_testing.DisableMemberInvitations,
			isInvitationAllowed:  false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setMemberInvitations(t.Context())
			t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

			canInviteUsers, err := users_services.GetSettingsService().CanInviteUsers(
				t.Context(),
				users.Actor{UserID: uuid.New(), Role: testCase.role},
			)

			require.NoError(t, err)
			require.Equal(t, testCase.isInvitationAllowed, canInviteUsers)
		})
	}
}

func TestCanCreateWorkspaces(t *testing.T) {
	testCases := []struct {
		name                       string
		role                       users.Role
		setMemberWorkspaceCreation func(ctx context.Context)
		isCreationAllowed          bool
	}{
		{
			name:                       "allows an admin when member workspace creation is disabled",
			role:                       users.RoleAdmin,
			setMemberWorkspaceCreation: users_testing.DisableMemberWorkspaceCreation,
			isCreationAllowed:          true,
		},
		{
			name:                       "allows a member when member workspace creation is enabled",
			role:                       users.RoleMember,
			setMemberWorkspaceCreation: users_testing.EnableMemberWorkspaceCreation,
			isCreationAllowed:          true,
		},
		{
			name:                       "forbids a member when member workspace creation is disabled",
			role:                       users.RoleMember,
			setMemberWorkspaceCreation: users_testing.DisableMemberWorkspaceCreation,
			isCreationAllowed:          false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.setMemberWorkspaceCreation(t.Context())
			t.Cleanup(func() { users_testing.ResetSettingsToDefaults(context.Background()) })

			canCreateWorkspaces, err := users_services.GetSettingsService().CanCreateWorkspaces(
				t.Context(),
				users.Actor{UserID: uuid.New(), Role: testCase.role},
			)

			require.NoError(t, err)
			require.Equal(t, testCase.isCreationAllowed, canCreateWorkspaces)
		})
	}
}
