package apptest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TurnstileValidAnswer is the only answer the fake Cloudflare Turnstile accepts.
const TurnstileValidAnswer = "valid-turnstile-answer"

const turnstileSecretKey = "test-turnstile-secret"

// startFakeTurnstile serves the verification endpoint of Cloudflare Turnstile until the test ends.
func startFakeTurnstile(tb testing.TB) string {
	tb.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		isValid := request.ParseForm() == nil &&
			request.PostForm.Get("secret") == turnstileSecretKey &&
			request.PostForm.Get("response") == TurnstileValidAnswer

		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]bool{"success": isValid})
	}))
	tb.Cleanup(server.Close)

	return server.URL
}
