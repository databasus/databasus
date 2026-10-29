package apptest

import (
	"time"

	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/storages"
)

// Option changes the test application before it is built.
type Option func(*settings)

type settings struct {
	isPhysicalBackupLoopsEnabled bool
	physicalBackupStopTimeout    time.Duration
	isMailServerMissing          bool
	isTurnstileEnabled           bool
	isGitHubEnabled              bool
	isGoogleEnabled              bool
	publicURL                    string
	clientToolsDir               string
	// notificationDeliveryTimeout is zero for the timeout the server uses.
	notificationDeliveryTimeout time.Duration
	notifierEndpoints           NotifierEndpoints
	// fileTimings is nil for the timings the server uses.
	fileTimings      *storages.FileTimings
	walStreamTimings *backups.WalStreamTimings
}

// NotifierEndpoints are the addresses a test points the Slack and Telegram notifiers at.
type NotifierEndpoints struct {
	SlackPostMessageURL string
	TelegramBotAPIURL   string
}

func WithClientToolsDir(directory string) Option {
	return func(applied *settings) {
		applied.clientToolsDir = directory
	}
}

// WithNotificationDeliveryTimeout builds an instance that gives up on a send after timeout, so a
// test of an endpoint that never answers ends quickly.
func WithNotificationDeliveryTimeout(timeout time.Duration) Option {
	return func(applied *settings) {
		applied.notificationDeliveryTimeout = timeout
	}
}

// WithNotifierEndpoints builds an instance whose Slack notifiers post to endpoints.SlackPostMessageURL
// and whose Telegram notifiers without a Bot API URL of their own use endpoints.TelegramBotAPIURL. An
// empty address keeps the one that does not resolve.
func WithNotifierEndpoints(endpoints NotifierEndpoints) Option {
	return func(applied *settings) {
		if endpoints.SlackPostMessageURL != "" {
			applied.notifierEndpoints.SlackPostMessageURL = endpoints.SlackPostMessageURL
		}

		if endpoints.TelegramBotAPIURL != "" {
			applied.notifierEndpoints.TelegramBotAPIURL = endpoints.TelegramBotAPIURL
		}
	}
}

// WithFileTimings builds an instance whose stored files and their cleanup follow timings, so a test
// can make a retry due or a commit window pass by moving the clock a short way.
func WithFileTimings(timings storages.FileTimings) Option {
	return func(applied *settings) {
		applied.fileTimings = &timings
	}
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

func WithPhysicalBackupLoops() Option {
	return func(applied *settings) {
		applied.isPhysicalBackupLoopsEnabled = true
	}
}

func WithWalStreamTimings(timings backups.WalStreamTimings) Option {
	return func(applied *settings) {
		applied.walStreamTimings = &timings
	}
}
