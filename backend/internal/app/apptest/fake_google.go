package apptest

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
)

const (
	googleClientID     = "test-google-client"
	googleClientSecret = "test-google-secret"
)

// GoogleAccount is the account the fake Google signs in.
type GoogleAccount struct {
	ID              string
	Email           string
	IsEmailVerified bool
	Name            string
}

// NewGoogleAccount returns an account with a unique ID and a verified address.
func NewGoogleAccount(email string) GoogleAccount {
	return GoogleAccount{ID: uuid.NewString(), Email: email, IsEmailVerified: true, Name: "Google User"}
}

// FakeGoogle serves the token and user info endpoints of Google.
type FakeGoogle struct {
	URL      string
	accounts *issuedAccounts[GoogleAccount]
}

func (f *FakeGoogle) Authorize(account GoogleAccount) string {
	return f.accounts.issue(account)
}

func startFakeGoogle(tb testing.TB) *FakeGoogle {
	tb.Helper()

	accounts := &issuedAccounts[GoogleAccount]{mutex: sync.Mutex{}, accounts: map[string]GoogleAccount{}}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", newTokenHandler(accounts, googleClientID, googleClientSecret))
	mux.HandleFunc("GET /userinfo", func(writer http.ResponseWriter, request *http.Request) {
		account, isFound := accounts.getByToken(writer, request)
		if !isFound {
			return
		}

		writeJSON(writer, map[string]any{
			"id":             account.ID,
			"email":          account.Email,
			"verified_email": account.IsEmailVerified,
			"name":           account.Name,
		})
	})

	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)

	return &FakeGoogle{URL: server.URL, accounts: accounts}
}
