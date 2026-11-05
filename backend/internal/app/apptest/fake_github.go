package apptest

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/google/uuid"
)

const (
	gitHubClientID     = "test-github-client"
	gitHubClientSecret = "test-github-secret"
)

// GitHubAccount is the account the fake GitHub signs in. GitHub shows PublicEmail on the profile
// only when it is verified; Emails is what the address list of the account returns.
type GitHubAccount struct {
	ID          int64
	Login       string
	Name        string
	PublicEmail string
	Emails      []GitHubEmail
}

type GitHubEmail struct {
	Email      string
	IsPrimary  bool
	IsVerified bool
}

// NewGitHubAccount returns an account with a unique ID whose public address is email.
func NewGitHubAccount(email string) GitHubAccount {
	return GitHubAccount{
		ID:          int64(uuid.New().ID()),
		Login:       "octocat",
		Name:        "GitHub User",
		PublicEmail: email,
		Emails:      nil,
	}
}

// FakeGitHub serves the token, profile and address endpoints of GitHub. Authorize hands out the
// code its authorization page would give the browser.
type FakeGitHub struct {
	URL      string
	accounts *issuedAccounts[GitHubAccount]
}

func (f *FakeGitHub) Authorize(account GitHubAccount) string {
	return f.accounts.issue(account)
}

func startFakeGitHub(tb testing.TB) *FakeGitHub {
	tb.Helper()

	accounts := &issuedAccounts[GitHubAccount]{mutex: sync.Mutex{}, accounts: map[string]GitHubAccount{}}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /login/oauth/access_token", newTokenHandler(accounts, gitHubClientID, gitHubClientSecret))
	mux.HandleFunc("GET /user", func(writer http.ResponseWriter, request *http.Request) {
		account, isFound := accounts.getByToken(writer, request)
		if !isFound {
			return
		}

		writeJSON(writer, map[string]any{
			"id":    account.ID,
			"login": account.Login,
			"name":  account.Name,
			"email": account.PublicEmail,
		})
	})
	mux.HandleFunc("GET /user/emails", func(writer http.ResponseWriter, request *http.Request) {
		account, isFound := accounts.getByToken(writer, request)
		if !isFound {
			return
		}

		listedEmails := make([]map[string]any, 0, len(account.Emails))
		for _, email := range account.Emails {
			listedEmails = append(listedEmails, map[string]any{
				"email":    email.Email,
				"primary":  email.IsPrimary,
				"verified": email.IsVerified,
			})
		}

		writeJSON(writer, listedEmails)
	})

	server := httptest.NewServer(mux)
	tb.Cleanup(server.Close)

	return &FakeGitHub{URL: server.URL, accounts: accounts}
}
