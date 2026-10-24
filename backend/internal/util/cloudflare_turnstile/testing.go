//go:build testhelpers

package cloudflare_turnstile

// A test that has to see an endpoint refuse an unanswered challenge cannot get
// there through the environment, because the service reads it once at startup.
// The returned function puts the previous key back.
func EnableChallengeForTesting(secretKey string) func() {
	previousSecretKey := cloudflareTurnstileService.secretKey
	cloudflareTurnstileService.secretKey = secretKey

	return func() {
		cloudflareTurnstileService.secretKey = previousSecretKey
	}
}
