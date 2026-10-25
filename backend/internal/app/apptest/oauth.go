package apptest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

const (
	gitHubClientID     = "test-github-client"
	gitHubClientSecret = "test-github-secret"
	googleClientID     = "test-google-client"
	googleClientSecret = "test-google-secret"
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

// GoogleAccount is the account the fake Google signs in.
type GoogleAccount struct {
	ID              string
	Email           string
	IsEmailVerified bool
	Name            string
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

// NewGoogleAccount returns an account with a unique ID and a verified address.
func NewGoogleAccount(email string) GoogleAccount {
	return GoogleAccount{ID: uuid.NewString(), Email: email, IsEmailVerified: true, Name: "Google User"}
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

// FakeGoogle serves the token and user info endpoints of Google.
type FakeGoogle struct {
	URL      string
	accounts *issuedAccounts[GoogleAccount]
}

func (f *FakeGoogle) Authorize(account GoogleAccount) string {
	return f.accounts.issue(account)
}

// issuedAccounts maps each issued code to its account. The code doubles as the access token, so
// the API endpoints find the account by the token.
type issuedAccounts[Account any] struct {
	mutex    sync.Mutex
	accounts map[string]Account
}

func (a *issuedAccounts[Account]) issue(account Account) string {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	code := "code-" + uuid.NewString()
	a.accounts[code] = account

	return code
}

func (a *issuedAccounts[Account]) get(code string) (Account, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	account, isFound := a.accounts[code]

	return account, isFound
}

// getByToken reads the account of the bearer token, answering 401 when there is none.
func (a *issuedAccounts[Account]) getByToken(writer http.ResponseWriter, request *http.Request) (Account, bool) {
	account, isFound := a.get(strings.TrimPrefix(request.Header.Get("Authorization"), "Bearer "))
	if !isFound {
		writer.WriteHeader(http.StatusUnauthorized)
	}

	return account, isFound
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

// newTokenHandler exchanges an issued code for a token when the client sends its own ID and secret,
// in the form or in basic authentication, and refuses anything else as an invalid grant.
func newTokenHandler[Account any](
	accounts *issuedAccounts[Account],
	clientID string,
	clientSecret string,
) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if request.ParseForm() != nil {
			writer.WriteHeader(http.StatusBadRequest)

			return
		}

		sentClientID, sentClientSecret, isBasicAuth := request.BasicAuth()
		if !isBasicAuth {
			sentClientID = request.PostForm.Get("client_id")
			sentClientSecret = request.PostForm.Get("client_secret")
		}

		code := request.PostForm.Get("code")
		_, isFound := accounts.get(code)

		if !isFound || sentClientID != clientID || sentClientSecret != clientSecret {
			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(writer).Encode(map[string]string{"error": "invalid_grant"})

			return
		}

		writeJSON(writer, map[string]string{"access_token": code, "token_type": "bearer"})
	}
}

func writeJSON(writer http.ResponseWriter, body any) {
	writer.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(writer).Encode(body)
}
