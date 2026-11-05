package apptest

import (
	"testing"

	apiclient "databasus-backend/internal/lib/apiclient"
)

// NewClient returns a client without a session.
func (a *App) NewClient(tb testing.TB) *apiclient.ClientWithResponses {
	tb.Helper()

	return apiclient.New(tb, a.URL)
}

// NewSessionClient returns a client that sends the session token.
func (a *App) NewSessionClient(tb testing.TB, token string) *apiclient.ClientWithResponses {
	tb.Helper()

	return apiclient.New(tb, a.URL, apiclient.WithSessionToken(token))
}
