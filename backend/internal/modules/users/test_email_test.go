package users_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestSendTestEmail(t *testing.T) {
	t.Parallel()

	t.Run("sends a message to the admin and records the delivery in the audit log", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		response := sendTestEmail(t, admin.Client)

		require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
		assert.Equal(t, admin.Email, response.JSON201.RecipientEmail)
		assert.True(t, response.JSON201.IsDelivered)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 1)
		assert.Contains(t, listGlobalAuditMessages(t, admin), "Test email sent to "+admin.Email)
	})

	t.Run("answers the mail server's error and records the failure when the server refuses the message",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			app.Mail.RefuseSends()

			response := sendTestEmail(t, admin.Client)

			require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))
			assert.False(t, response.JSON201.IsDelivered)
			require.NotNil(t, response.JSON201.FailureDetail)
			assert.Contains(t, *response.JSON201.FailureDetail, "refuses the message")
			assert.Contains(t, listGlobalAuditMessages(t, admin), "Test email to "+admin.Email+" failed")
		})

	t.Run("returns 403 permission_denied and sends nothing when a member asks", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		member := app.CreateNonAdminUser(t, app.CreateAdmin(t))

		response := sendTestEmail(t, member.Client)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusForbidden,
			apiclient.ProblemCodePermissionDenied)
		assert.Zero(t, app.Mail.GetMessageCount())
	})

	t.Run("returns 409 email_not_configured when the instance has no mail server", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t, apptest.WithoutMailServer())
		admin := app.CreateAdmin(t)

		response := sendTestEmail(t, admin.Client)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
			apiclient.ProblemCodeEmailNotConfigured)
	})

	t.Run("returns 409 admin_email_missing and sends nothing when the admin holds the login admin",
		func(t *testing.T) {
			t.Parallel()

			app := apptest.New(t)
			admin := app.CreateAdmin(t)
			app.ExecuteSQL(t, "UPDATE users SET email = 'admin' WHERE id = ?", admin.UserID)

			response := sendTestEmail(t, admin.Client)

			requireProblem(t, response.StatusCode(), response.Body, http.StatusConflict,
				apiclient.ProblemCodeAdminEmailMissing)
			assert.Zero(t, app.Mail.GetMessageCount())
		})

	t.Run("returns 429 rate_limited and sends nothing on the sixth request within ten minutes", func(t *testing.T) {
		t.Parallel()

		app := apptest.New(t)
		admin := app.CreateAdmin(t)

		for range 5 {
			require.Equal(t, http.StatusCreated, sendTestEmail(t, admin.Client).StatusCode())
		}

		response := sendTestEmail(t, admin.Client)

		requireProblem(t, response.StatusCode(), response.Body, http.StatusTooManyRequests,
			apiclient.ProblemCodeRateLimited)
		assert.Len(t, app.Mail.GetMessagesTo(admin.Email), 5)
	})
}

func sendTestEmail(t *testing.T, client *apiclient.ClientWithResponses) *apiclient.SendTestEmailResponse {
	t.Helper()

	response, err := client.SendTestEmailWithResponse(t.Context())
	require.NoError(t, err)

	return response
}
