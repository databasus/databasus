package apptest

import (
	"log/slog"
	"time"

	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/storages"
)

// Option changes the test application before it is built.
type Option func(*settings)

type settings struct {
	isTelemetryEnabled           bool
	telemetryReceiverURL         string
	telemetryIdentityPath        string
	isPhysicalBackupLoopsEnabled bool
	physicalBackupStopTimeout    time.Duration
	isMailServerMissing          bool
	// instanceMailServer is nil for the recorder that stands in for the operator's mail server.
	instanceMailServer  *mailer.Settings
	isTurnstileEnabled  bool
	isGitHubEnabled     bool
	isGoogleEnabled     bool
	isRecoveryPublished bool
	// recoveryMaxArchiveBytes is zero for the configured limit.
	recoveryMaxArchiveBytes int64
	publicURL               string
	clientToolsDir          string
	agentBinariesDir        string
	logSink                 slog.Handler
	// notificationDeliveryTimeout is zero for the timeout the server uses.
	notificationDeliveryTimeout time.Duration
	notifierEndpoints           NotifierEndpoints
	walStreamTimings            *backups.WalStreamTimings
	storageFileTimings          *storages.FileTimings
	// inspectDisk is nil for the real free space of the disk.
	inspectDisk func(path string) (diskusage.Usage, error)
}

// NotifierEndpoints are the addresses a test points the Slack and Telegram notifiers at.
type NotifierEndpoints struct {
	SlackPostMessageURL string
	TelegramBotAPIURL   string
}

// WithInstanceMailServer sends the instance's own mail through a real SMTP server, such as Mailpit,
// instead of the recorder in App.Mail, which then stays empty.
func WithInstanceMailServer(mailServer mailer.Settings) Option {
	return func(applied *settings) {
		applied.instanceMailServer = &mailServer
	}
}

// WithRecoveryMaxArchiveBytes lowers the size limit of a recovery upload, so a test reaches it
// with a small file.
func WithRecoveryMaxArchiveBytes(maxBytes int64) Option {
	return func(applied *settings) {
		applied.recoveryMaxArchiveBytes = maxBytes
	}
}

func WithClientToolsDir(directory string) Option {
	return func(applied *settings) {
		applied.clientToolsDir = directory
	}
}

func WithLogSink(sink slog.Handler) Option {
	return func(applied *settings) {
		applied.logSink = sink
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

func WithStorageFileTimings(timings storages.FileTimings) Option {
	return func(applied *settings) {
		applied.storageFileTimings = &timings
	}
}

// WithDiskUsage replaces the free-space reading of the disk a Databasus backup is staged on, so a
// test reaches a full disk without filling one.
func WithDiskUsage(inspectDisk func(path string) (diskusage.Usage, error)) Option {
	return func(applied *settings) { applied.inspectDisk = inspectDisk }
}

// The real agent keeps its own time, so serving its binaries also selects the system clock.
func WithAgentBinariesDir(directory string) Option {
	return func(applied *settings) { applied.agentBinariesDir = directory }
}
