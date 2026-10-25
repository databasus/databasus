package apptest

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// TestPassword is the password of every user the helpers sign up.
const TestPassword = "test-password-123"

// Session is a signed-in user with a client that sends their session token.
type Session struct {
	Client *apiclient.ClientWithResponses
	UserID uuid.UUID
	Email  string
	Token  string
}

// CreateAdmin signs up the first user of the fresh test database, which claims the instance and
// administers it.
func (a *App) CreateAdmin(tb testing.TB) Session {
	tb.Helper()

	return a.SignUp(tb, NewTestEmail("admin"))
}

// SignUp signs up a user with TestPassword through the API and fails the test unless it succeeds. It
// answers the challenge of WithTurnstile, which an instance without one ignores.
func (a *App) SignUp(tb testing.TB, email string) Session {
	tb.Helper()

	response, err := a.NewClient(tb).SignUpWithResponse(tb.Context(), apiclient.SignUpRequest{
		Email:                    openapi_types.Email(email),
		Password:                 new(TestPassword),
		Name:                     "Test User",
		CloudflareTurnstileToken: new(TurnstileValidAnswer),
	})
	if err != nil {
		tb.Fatalf("sign up %s: %v", email, err)
	}

	if response.StatusCode() != http.StatusCreated {
		tb.Fatalf("sign up %s answered %d: %s", email, response.StatusCode(), response.Body)
	}

	return Session{
		Client: a.NewSessionClient(tb, response.JSON201.Token),
		UserID: response.JSON201.UserId,
		Email:  response.JSON201.Email,
		Token:  response.JSON201.Token,
	}
}

// NewTestEmail returns a unique address that starts with the label.
func NewTestEmail(label string) string {
	return label + "-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12] + "@example.com"
}

// CreateNonAdminUser opens registration as the administrator, keeping the other settings, and signs
// up a second user, who is not an instance admin.
func (a *App) CreateNonAdminUser(tb testing.TB, admin Session) Session {
	tb.Helper()

	currentSettings, err := admin.Client.GetInstanceSettingsWithResponse(tb.Context())
	if err != nil || currentSettings.JSON200 == nil {
		tb.Fatalf("read the instance settings: %v", err)
	}

	updatedSettings, err := admin.Client.UpdateInstanceSettingsWithResponse(
		tb.Context(),
		apiclient.InstanceSettingsUpdate{
			IsAllowExternalRegistrations:      true,
			IsAllowMemberInvitations:          currentSettings.JSON200.IsAllowMemberInvitations,
			IsMemberAllowedToCreateWorkspaces: currentSettings.JSON200.IsMemberAllowedToCreateWorkspaces,
			IsTwoFactorAuthRequired:           currentSettings.JSON200.IsTwoFactorAuthRequired,
		},
	)
	if err != nil || updatedSettings.StatusCode() != http.StatusOK {
		tb.Fatalf("open registration: %v", err)
	}

	return a.SignUp(tb, NewTestEmail("member"))
}
