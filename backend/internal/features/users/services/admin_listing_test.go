package users_services_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	users_enums "databasus-backend/internal/features/users/enums"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	users_testing "databasus-backend/internal/features/users/testing"
	"databasus-backend/internal/modules/users"
)

func Test_RenderAdminListing_WithSeveralAdministrators_MarksTheBootstrapOneAndOmitsMembers(t *testing.T) {
	users_testing.DeleteAllUsers()

	bootstrapAdmin := users_testing.RecreateInitialAdmin(t.Context())
	ordinaryAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	deactivatedAdmin := users_testing.CreateTestUser(t.Context(), users.RoleAdmin)
	member := users_testing.CreateTestUser(t.Context(), users.RoleMember)

	require.NoError(t, (&users_repositories.UserRepository{}).UpdateUserStatus(
		deactivatedAdmin.UserID,
		users_enums.UserStatusInactive,
	))

	listing, err := users_services.GetManagementService().RenderAdminListing(t.Context())
	require.NoError(t, err)

	assert.Contains(t, listing, bootstrapAdmin.Email+" |")
	assert.Contains(t, listing, ordinaryAdmin.Email+" |")
	assert.Contains(t, listing, deactivatedAdmin.Email+" |")
	assert.NotContains(t, listing, member.Email)

	assert.Contains(t, adminLineFor(t, listing, bootstrapAdmin.Email), "bootstrap administrator")
	assert.NotContains(t, adminLineFor(t, listing, ordinaryAdmin.Email), "bootstrap administrator")

	assert.Contains(t, adminLineFor(t, listing, deactivatedAdmin.Email), "inactive")
	assert.Contains(t, adminLineFor(t, listing, ordinaryAdmin.Email), "active")

	assert.NotContains(t, listing, "$2a$")
}

func Test_RenderAdminListing_WhenNobodyHasRegistered_ReportsNoAdministrator(t *testing.T) {
	users_testing.DeleteAllUsers()

	listing, err := users_services.GetManagementService().RenderAdminListing(t.Context())
	require.NoError(t, err)

	assert.Equal(t, users_services.NoAdminsMessage, listing)
}

func adminLineFor(t *testing.T, listing, email string) string {
	t.Helper()

	for _, line := range strings.Split(listing, "\n") {
		if strings.HasPrefix(line, email+" |") {
			return line
		}
	}

	t.Fatalf("no line for %s in listing:\n%s", email, listing)

	return ""
}
