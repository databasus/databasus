package apptest

// Option changes the test application before it is built.
type Option func(*settings)

type settings struct {
	isMailServerMissing bool
	isTurnstileEnabled  bool
	isGitHubEnabled     bool
	isGoogleEnabled     bool
	publicURL           string
}

// WithoutMailServer builds an instance whose operator configured no mail server.
func WithoutMailServer() Option {
	return func(applied *settings) {
		applied.isMailServerMissing = true
	}
}

// WithTurnstile builds an instance with a human-verification challenge served by a fake Cloudflare
// Turnstile that accepts only TurnstileValidAnswer.
func WithTurnstile() Option {
	return func(applied *settings) {
		applied.isTurnstileEnabled = true
	}
}

// WithGitHub builds an instance that offers GitHub sign-in through a fake GitHub.
func WithGitHub() Option {
	return func(applied *settings) {
		applied.isGitHubEnabled = true
	}
}

// WithGoogle builds an instance that offers Google sign-in through a fake Google.
func WithGoogle() Option {
	return func(applied *settings) {
		applied.isGoogleEnabled = true
	}
}

// WithPublicURL builds an instance whose operator configured the public address that email links
// point to.
func WithPublicURL(publicURL string) Option {
	return func(applied *settings) {
		applied.publicURL = publicURL
	}
}
