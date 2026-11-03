// Package config reads the application settings once, from the process environment, an optional
// .env file and the connection string published by container startup. It never changes the process
// environment: the database clients the application starts inherit that environment, and values
// the application loaded for itself must not reach them (spec runtime-configuration).
package config

import (
	"strings"

	"databasus-backend/internal/lib/mailer"
)

type Config struct {
	EnvMode EnvMode `env:"ENV_MODE,required"`

	DatabaseDsn string `env:"DATABASE_DSN"`

	ShowDbInstallationVerificationLogs bool `env:"SHOW_DB_INSTALLATION_VERIFICATION_LOGS" envDefault:"true"`

	IsDisableAnonymousTelemetry bool `env:"IS_DISABLE_ANONYMOUS_TELEMETRY"`

	GitHubClientID     string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`

	CloudflareTurnstileSecretKey string `env:"CLOUDFLARE_TURNSTILE_SECRET_KEY"`
	CloudflareTurnstileSiteKey   string `env:"CLOUDFLARE_TURNSTILE_SITE_KEY"`

	SMTPHost               string          `env:"SMTP_HOST"`
	SMTPPort               int             `env:"SMTP_PORT"`
	SMTPUser               string          `env:"SMTP_USER"`
	SMTPPassword           string          `env:"SMTP_PASSWORD"`
	SMTPFrom               string          `env:"SMTP_FROM"`
	SMTPSecurity           mailer.Security `env:"SMTP_SECURITY"`
	SMTPHeloName           string          `env:"SMTP_HELO_NAME"`
	SMTPInsecureSkipVerify bool            `env:"SMTP_INSECURE_SKIP_VERIFY"`

	// DatabasusURL is the address email links point to.
	DatabasusURL string `env:"DATABASUS_URL"`

	LogLevel             string `env:"LOG_LEVEL"`
	LogFileIsEnabled     string `env:"LOG_FILE_IS_ENABLED"`
	OpenTelemetryURL     string `env:"OPEN_TELEMETRY_URL"`
	OpenTelemetryHeaders string `env:"OPEN_TELEMETRY_HEADERS"`
	AppVersion           string `env:"APP_VERSION"            envDefault:"dev"`

	// The paths below all live in DataDirectory, the one data volume.
	DataDirectory         string `env:"-"`
	DataFolder            string `env:"-"`
	TempFolder            string `env:"-"`
	SecretKeyPath         string `env:"-"`
	TelemetryInstancePath string `env:"-"`
	LogFilePath           string `env:"-"`
}

// IsLogFileEnabled turns the file off only on an explicit false: the file is the forensic copy an
// operator reaches for when stdout is already gone, so a typo keeps it on.
func (c Config) IsLogFileEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(c.LogFileIsEnabled), "false")
}

// GetInstanceMailSettings fills an unset security mode from the port and an unset greeting name from
// the public address, so the instance's own mail works with only SMTP_HOST and SMTP_PORT set.
func (c Config) GetInstanceMailSettings() mailer.Settings {
	if c.SMTPHost == "" {
		return mailer.Settings{}
	}

	security := c.SMTPSecurity
	if security == "" {
		security = mailer.GetDefaultSecurityForPort(c.SMTPPort)
	}

	heloName := c.SMTPHeloName
	if heloName == "" {
		heloName = mailer.GetDefaultHeloName(c.DatabasusURL)
	}

	return mailer.Settings{
		Host:                 c.SMTPHost,
		Port:                 c.SMTPPort,
		Security:             security,
		Username:             c.SMTPUser,
		Password:             c.SMTPPassword,
		From:                 c.SMTPFrom,
		HeloName:             heloName,
		IsInsecureSkipVerify: c.SMTPInsecureSkipVerify,
	}
}

type EnvMode string

const (
	EnvModeDevelopment EnvMode = "development"
	EnvModeProduction  EnvMode = "production"
)
