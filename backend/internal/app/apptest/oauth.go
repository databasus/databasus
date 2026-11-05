package apptest

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

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
