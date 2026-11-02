package users_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestListUsers(t *testing.T) {
	t.Parallel()

	t.Run("pages through every user, the newest first", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		signedUpEmails := []string{admin.Email}

		for range 4 {
			app.Clock.Advance(time.Second)
			signedUpEmails = append(signedUpEmails, app.SignUp(t, apptest.NewTestEmail("member")).Email)
		}

		limit := 2
		var listedEmails []string
		var after *string
		pageCount := 0

		for {
			page := requireUserPage(
				t,
				admin.Client,
				&apiclient.ListUsersParams{Limit: &limit, After: after, Query: nil},
			)
			pageCount++

			for _, user := range page.Items {
				listedEmails = append(listedEmails, user.Email)
			}

			if page.NextCursor == nil {
				break
			}

			after = page.NextCursor
		}

		assert.Equal(t, 3, pageCount)
		assert.Equal(t, []string{
			signedUpEmails[4], signedUpEmails[3], signedUpEmails[2], signedUpEmails[1], signedUpEmails[0],
		}, listedEmails)
	})

	t.Run("lists users created at the same moment once each", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		for range 3 {
			app.SignUp(t, apptest.NewTestEmail("member"))
		}

		limit := 1
		listedIDs := map[uuid.UUID]bool{}
		var after *string

		for {
			page := requireUserPage(
				t,
				admin.Client,
				&apiclient.ListUsersParams{Limit: &limit, After: after, Query: nil},
			)
			for _, user := range page.Items {
				assert.False(t, listedIDs[user.Id], "user %s listed twice", user.Email)
				listedIDs[user.Id] = true
			}

			if page.NextCursor == nil {
				break
			}

			after = page.NextCursor
		}

		assert.Len(t, listedIDs, 4)
	})

	t.Run("keeps the users whose address or name contains the query, ignoring case", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		matchingMember := app.SignUp(t, apptest.NewTestEmail("findme"))
		app.SignUp(t, apptest.NewTestEmail("other"))
		query := "FINDME"

		page := requireUserPage(t, admin.Client, &apiclient.ListUsersParams{Limit: nil, After: nil, Query: &query})

		require.Len(t, page.Items, 1)
		assert.Equal(t, matchingMember.Email, page.Items[0].Email)
	})

	t.Run("matches a wildcard character of the query literally", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		app.SignUp(t, apptest.NewTestEmail("member"))
		query := "%"

		page := requireUserPage(t, admin.Client, &apiclient.ListUsersParams{Limit: nil, After: nil, Query: &query})

		assert.Empty(t, page.Items)
	})

	t.Run("returns 400 naming the cursor when it is not a cursor of the list", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		after := "not-a-cursor"

		response, err := admin.Client.ListUsersWithResponse(t.Context(),
			&apiclient.ListUsersParams{Limit: nil, After: &after, Query: nil})
		require.NoError(t, err)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusBadRequest,
			apiclient.ProblemCodeInvalidRequest)
		require.NotNil(t, response.ApplicationproblemJSON400.Errors)
		assert.Equal(t, "after", (*response.ApplicationproblemJSON400.Errors)[0].Field)
	})

	t.Run("returns 403 permission_denied to a member", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		response, err := member.Client.ListUsersWithResponse(t.Context(),
			&apiclient.ListUsersParams{Limit: nil, After: nil, Query: nil})
		require.NoError(t, err)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})
}

func TestDeactivateUser(t *testing.T) {
	t.Parallel()

	t.Run("returns 401 unauthenticated for a token of the deactivated user", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		deactivated := deactivateUser(t, admin.Client, member.UserID)
		require.Equal(t, http.StatusOK, deactivated.StatusCode(), string(deactivated.Body))
		assert.Equal(t, apiclient.UserStatusINACTIVE, deactivated.JSON200.Status)

		response := getUser(t, member.Client, member.UserID)
		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeUnauthenticated)
	})

	t.Run("refuses a wrong password to the deactivated user like to anyone", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		requireDeactivated(t, admin, member.UserID)

		response := signIn(t, app, member.Email, "wrong-password-123")

		requireProblem(t, response.StatusCode(), response.Body, http.StatusUnauthorized,
			apiclient.ProblemCodeInvalidCredentials)
	})

	t.Run("answers user_deactivated without a token to the correct password", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		requireDeactivated(t, admin, member.UserID)

		response := signIn(t, app, member.Email, apptest.TestPassword)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodeUserDeactivated)
		assert.NotContains(t, string(response.Body), "token")
	})

	t.Run("returns 403 permission_denied when an administrator deactivates themselves", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := deactivateUser(t, admin.Client, admin.UserID)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("returns 404 not_found for a user that does not exist", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := deactivateUser(t, admin.Client, uuid.New())

		requireProblem(t, response.StatusCode(), response.Body, http.StatusNotFound, apiclient.ProblemCodeNotFound)
	})
}

func TestActivateUser(t *testing.T) {
	t.Parallel()

	t.Run("lets the deactivated user sign in again", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		requireDeactivated(t, admin, member.UserID)

		response, err := admin.Client.ActivateUserWithResponse(t.Context(), member.UserID)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
		assert.Equal(t, apiclient.UserStatusACTIVE, response.JSON200.Status)
		requireCompletedSignIn(t, signIn(t, app, member.Email, apptest.TestPassword))
	})
}

func TestManageAdmins(t *testing.T) {
	t.Parallel()

	t.Run("lets the bootstrap administrator manage admins after changing address", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		bootstrapAdmin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))
		updated := updateUser(t, bootstrapAdmin.Client, bootstrapAdmin.UserID, "Owner", apptest.NewTestEmail("moved"))
		require.Equal(t, http.StatusOK, updated.StatusCode(), string(updated.Body))

		promoted := changeUserRole(t, bootstrapAdmin.Client, member.UserID, apiclient.UserRoleADMIN)
		require.Equal(t, http.StatusOK, promoted.StatusCode(), string(promoted.Body))
		assert.Equal(t, apiclient.UserRoleADMIN, promoted.JSON200.Role)

		deactivated := deactivateUser(t, bootstrapAdmin.Client, member.UserID)
		require.Equal(t, http.StatusOK, deactivated.StatusCode(), string(deactivated.Body))

		activated, err := bootstrapAdmin.Client.ActivateUserWithResponse(t.Context(), member.UserID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, activated.StatusCode(), string(activated.Body))

		demoted := changeUserRole(t, bootstrapAdmin.Client, member.UserID, apiclient.UserRoleMEMBER)
		require.Equal(t, http.StatusOK, demoted.StatusCode(), string(demoted.Body))
		assert.Equal(t, apiclient.UserRoleMEMBER, demoted.JSON200.Role)
	})

	t.Run("refuses every admin change to an admin who is not the bootstrap administrator", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		bootstrapAdmin := app.CreateAdmin(t)
		otherAdmin := appointAdmin(t, app, bootstrapAdmin)
		targetAdmin := appointAdmin(t, app, bootstrapAdmin)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		promoted := changeUserRole(t, otherAdmin.Client, member.UserID, apiclient.UserRoleADMIN)
		demoted := changeUserRole(t, otherAdmin.Client, targetAdmin.UserID, apiclient.UserRoleMEMBER)
		deactivated := deactivateUser(t, otherAdmin.Client, targetAdmin.UserID)
		requireDeactivated(t, bootstrapAdmin, targetAdmin.UserID)
		activated, err := otherAdmin.Client.ActivateUserWithResponse(t.Context(), targetAdmin.UserID)
		require.NoError(t, err)

		requireProblem(t, promoted.StatusCode(), promoted.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireProblem(t, demoted.StatusCode(), demoted.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireProblem(t, deactivated.StatusCode(), deactivated.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireProblem(t, activated.StatusCode(), activated.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
	})

	t.Run("lets an admin who is not the bootstrap administrator manage members", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		bootstrapAdmin := app.CreateAdmin(t)
		otherAdmin := appointAdmin(t, app, bootstrapAdmin)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		deactivated := deactivateUser(t, otherAdmin.Client, member.UserID)

		require.Equal(t, http.StatusOK, deactivated.StatusCode(), string(deactivated.Body))
	})

	t.Run("refuses to strip the bootstrap administrator of the role or deactivate it", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		bootstrapAdmin := app.CreateAdmin(t)
		otherAdmin := appointAdmin(t, app, bootstrapAdmin)

		demoted := changeUserRole(t, otherAdmin.Client, bootstrapAdmin.UserID, apiclient.UserRoleMEMBER)
		deactivated := deactivateUser(t, otherAdmin.Client, bootstrapAdmin.UserID)
		ownRoleChange := changeUserRole(t, bootstrapAdmin.Client, bootstrapAdmin.UserID, apiclient.UserRoleMEMBER)

		requireProblem(t, demoted.StatusCode(), demoted.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireProblem(t, deactivated.StatusCode(), deactivated.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		requireProblem(t, ownRoleChange.StatusCode(), ownRoleChange.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)

		readUser := getUser(t, bootstrapAdmin.Client, bootstrapAdmin.UserID)
		require.Equal(t, http.StatusOK, readUser.StatusCode(), string(readUser.Body))
		assert.Equal(t, apiclient.UserRoleADMIN, readUser.JSON200.Role)
	})

	t.Run("returns 400 naming the role when it is not a role", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		bootstrapAdmin := app.CreateAdmin(t)
		member := app.SignUp(t, apptest.NewTestEmail("member"))

		status, body := sendAuthorizedRawRequest(t, app, bootstrapAdmin, http.MethodPut,
			"/users/"+member.UserID.String()+"/role", `{"role":"OWNER"}`)

		requireProblem(t, status, body, http.StatusBadRequest, apiclient.ProblemCodeInvalidRequest)
	})
}

// appointAdmin signs up a member and grants it the admin role as the bootstrap administrator.
func appointAdmin(t *testing.T, app *apptest.App, bootstrapAdmin apptest.Session) apptest.Session {
	t.Helper()

	appointed := app.SignUp(t, apptest.NewTestEmail("admin"))
	response := changeUserRole(t, bootstrapAdmin.Client, appointed.UserID, apiclient.UserRoleADMIN)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return appointed
}

func requireDeactivated(t *testing.T, admin apptest.Session, userID uuid.UUID) {
	t.Helper()

	response := deactivateUser(t, admin.Client, userID)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func requireUserPage(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	params *apiclient.ListUsersParams,
) apiclient.UserPage {
	t.Helper()

	response, err := client.ListUsersWithResponse(t.Context(), params)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func deactivateUser(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	userID uuid.UUID,
) *apiclient.DeactivateUserResponse {
	t.Helper()

	response, err := client.DeactivateUserWithResponse(t.Context(), userID)
	require.NoError(t, err)

	return response
}

func changeUserRole(
	t *testing.T,
	client *apiclient.ClientWithResponses,
	userID uuid.UUID,
	role apiclient.UserRole,
) *apiclient.ChangeUserRoleResponse {
	t.Helper()

	response, err := client.ChangeUserRoleWithResponse(t.Context(), userID, apiclient.UserRoleChange{Role: role})
	require.NoError(t, err)

	return response
}

// sendAuthorizedRawRequest sends a body the contract cannot describe with the session's token.
func sendAuthorizedRawRequest(
	t *testing.T,
	app *apptest.App,
	session apptest.Session,
	method string,
	path string,
	body string,
) (int, []byte) {
	t.Helper()

	request, err := http.NewRequestWithContext(t.Context(), method, app.URL+"/api/v1"+path, jsonBody(t, body))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+session.Token)

	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)

	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	return response.StatusCode, responseBody
}
