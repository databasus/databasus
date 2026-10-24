// Package config reads the application settings once, from the process environment, an optional
// .env file and the connection string published by container startup. It never changes the process
// environment: the database clients the application starts inherit that environment, and values
// the application loaded for itself must not reach them (spec runtime-configuration).
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"

	env_utils "databasus-backend/internal/util/env"
	"databasus-backend/internal/util/smtp_transport"
)

const databaseDsnVariable = "DATABASE_DSN"

// A process started with `docker exec` does not inherit PID 1's environment, so this file is the
// only way it learns the internal password, which is rotated on every container start. The
// location is memory-backed, so the value dies with the container instead of going stale against
// the next rotation.
const defaultPublishedDatabaseDsnPath = "/dev/shm/databasus-database-dsn"

type Config struct {
	EnvMode env_utils.EnvMode `env:"ENV_MODE,required"`

	DatabaseDsn string `env:"DATABASE_DSN"`

	ShowDbInstallationVerificationLogs bool `env:"SHOW_DB_INSTALLATION_VERIFICATION_LOGS" envDefault:"true"`

	IsDisableAnonymousTelemetry bool `env:"IS_DISABLE_ANONYMOUS_TELEMETRY"`

	GitHubClientID     string `env:"GITHUB_CLIENT_ID"`
	GitHubClientSecret string `env:"GITHUB_CLIENT_SECRET"`
	GoogleClientID     string `env:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret string `env:"GOOGLE_CLIENT_SECRET"`

	CloudflareTurnstileSecretKey string `env:"CLOUDFLARE_TURNSTILE_SECRET_KEY"`
	CloudflareTurnstileSiteKey   string `env:"CLOUDFLARE_TURNSTILE_SITE_KEY"`

	SMTPHost               string                  `env:"SMTP_HOST"`
	SMTPPort               int                     `env:"SMTP_PORT"`
	SMTPUser               string                  `env:"SMTP_USER"`
	SMTPPassword           string                  `env:"SMTP_PASSWORD"`
	SMTPFrom               string                  `env:"SMTP_FROM"`
	SMTPSecurity           smtp_transport.Security `env:"SMTP_SECURITY"`
	SMTPHeloName           string                  `env:"SMTP_HELO_NAME"`
	SMTPInsecureSkipVerify bool                    `env:"SMTP_INSECURE_SKIP_VERIFY"`

	// DatabasusURL is the address email links point to.
	DatabasusURL string `env:"DATABASUS_URL"`

	LogLevel             string `env:"LOG_LEVEL"`
	LogFileIsEnabled     string `env:"LOG_FILE_IS_ENABLED"`
	OpenTelemetryURL     string `env:"OPEN_TELEMETRY_URL"`
	OpenTelemetryHeaders string `env:"OPEN_TELEMETRY_HEADERS"`
	AppVersion           string `env:"APP_VERSION"`

	// The paths below all live in DataDirectory, the one data volume.
	DataDirectory         string `env:"-"`
	DataFolder            string `env:"-"`
	TempFolder            string `env:"-"`
	SecretKeyPath         string `env:"-"`
	TelemetryInstancePath string `env:"-"`
	LogFilePath           string `env:"-"`
}

type LoadSpec struct {
	// The storage probe runs before the embedded database exists, so it starts without a
	// connection string.
	IsDatabaseRequired bool

	// RootDirectory holds .env and the databasus-data directory. Empty means the parent of the
	// backend module found upward from the working directory, or / in the image.
	RootDirectory string

	// PublishedDatabaseDsnPath is empty in production, which means the path container startup
	// writes to.
	PublishedDatabaseDsnPath string
}

// LoadReport tells the caller what Load found, so it can log it once a logger exists.
type LoadReport struct {
	EnvFilePath                string
	IsEnvFileLoaded            bool
	IsPublishedDatabaseDsnUsed bool
	// PublishedDatabaseDsnReadError is set when the published file exists but cannot be read.
	// Startup goes on, because the connection string may come from elsewhere.
	PublishedDatabaseDsnReadError error
}

// IsLogFileEnabled turns the file off only on an explicit false: the file is the forensic copy an
// operator reaches for when stdout is already gone, so a typo keeps it on.
func (c Config) IsLogFileEnabled() bool {
	return !strings.EqualFold(strings.TrimSpace(c.LogFileIsEnabled), "false")
}

func Load(spec LoadSpec) (Config, LoadReport, error) {
	var loadedConfig Config

	variables, report, err := ReadVariables(spec)
	if err != nil {
		return Config{}, report, err
	}

	err = env.ParseWithOptions(&loadedConfig, env.Options{Environment: variables})
	if err != nil {
		return Config{}, report, fmt.Errorf("parse settings: %w", err)
	}

	err = validate(loadedConfig, spec)
	if err != nil {
		return Config{}, report, err
	}

	setDataPaths(&loadedConfig, getRootDirectory(spec))

	return loadedConfig, report, nil
}

// ReadVariables merges the setting sources into one map: the process environment first, then .env
// and the published connection string for whatever is still unset. An empty value counts as unset,
// so an empty DATABASE_DSN does not hide the published one.
func ReadVariables(spec LoadSpec) (map[string]string, LoadReport, error) {
	report := LoadReport{EnvFilePath: filepath.Join(getRootDirectory(spec), ".env")}

	variables := map[string]string{}

	for _, variable := range os.Environ() {
		name, value, isPair := strings.Cut(variable, "=")
		if isPair && value != "" {
			variables[name] = value
		}
	}

	fileVariables, isEnvFileLoaded, err := readEnvFile(report.EnvFilePath)
	if err != nil {
		return nil, report, fmt.Errorf("read %s: %w", report.EnvFilePath, err)
	}

	report.IsEnvFileLoaded = isEnvFileLoaded

	for name, value := range fileVariables {
		if _, isSet := variables[name]; !isSet && value != "" {
			variables[name] = value
		}
	}

	if _, isSet := variables[databaseDsnVariable]; !isSet {
		publishedDsn, err := readPublishedDatabaseDsn(getPublishedDatabaseDsnPath(spec))
		if err != nil {
			report.PublishedDatabaseDsnReadError = err
		}

		if publishedDsn != "" {
			variables[databaseDsnVariable] = publishedDsn
			report.IsPublishedDatabaseDsnUsed = true
		}
	}

	return variables, report, nil
}

// The Docker image carries no .env, so every setting there comes from the operator or from a code
// default. A development checkout still reads its own .env.
func readEnvFile(envFilePath string) (map[string]string, bool, error) {
	_, err := os.Stat(envFilePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}

	if err != nil {
		return nil, false, err
	}

	fileVariables, err := godotenv.Read(envFilePath)
	if err != nil {
		return nil, false, err
	}

	return fileVariables, true, nil
}

func readPublishedDatabaseDsn(publishedDsnPath string) (string, error) {
	publishedDsn, err := os.ReadFile(publishedDsnPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(publishedDsn)), nil
}

func validate(loadedConfig Config, spec LoadSpec) error {
	if spec.IsDatabaseRequired && loadedConfig.DatabaseDsn == "" {
		return fmt.Errorf("%w: %s", ErrRequiredVariableNotSet, databaseDsnVariable)
	}

	if loadedConfig.EnvMode != env_utils.EnvModeDevelopment && loadedConfig.EnvMode != env_utils.EnvModeProduction {
		return fmt.Errorf(
			"%w: ENV_MODE %q is neither development nor production",
			ErrInvalidVariable,
			loadedConfig.EnvMode,
		)
	}

	return validateSMTPSettings(loadedConfig)
}

func validateSMTPSettings(loadedConfig Config) error {
	if loadedConfig.SMTPHost != "" && loadedConfig.SMTPPort <= 0 {
		return fmt.Errorf(
			"%w: SMTP_PORT must be a positive integer when SMTP_HOST is set, got %d",
			ErrInvalidVariable, loadedConfig.SMTPPort,
		)
	}

	if loadedConfig.SMTPSecurity != "" && !loadedConfig.SMTPSecurity.IsValid() {
		return fmt.Errorf("SMTP_SECURITY %q: %w", loadedConfig.SMTPSecurity, smtp_transport.ErrUnknownSecurity)
	}

	if loadedConfig.SMTPHeloName != "" {
		if err := smtp_transport.ValidateHeloName(loadedConfig.SMTPHeloName); err != nil {
			return fmt.Errorf("SMTP_HELO_NAME %q: %w", loadedConfig.SMTPHeloName, err)
		}
	}

	if loadedConfig.SMTPFrom != "" {
		if _, err := smtp_transport.ParseSender(loadedConfig.SMTPFrom); err != nil {
			return fmt.Errorf("SMTP_FROM %q: %w", loadedConfig.SMTPFrom, err)
		}
	}

	return nil
}

func setDataPaths(loadedConfig *Config, rootDirectory string) {
	loadedConfig.DataDirectory = filepath.Join(rootDirectory, "databasus-data")
	loadedConfig.DataFolder = filepath.Join(loadedConfig.DataDirectory, "backups")
	loadedConfig.TempFolder = filepath.Join(loadedConfig.DataDirectory, "temp")
	loadedConfig.SecretKeyPath = filepath.Join(loadedConfig.DataDirectory, "secret.key")
	loadedConfig.TelemetryInstancePath = filepath.Join(loadedConfig.DataDirectory, "instance.json")
	loadedConfig.LogFilePath = filepath.Join(loadedConfig.DataDirectory, "databasus.log")
}

func getRootDirectory(spec LoadSpec) string {
	if spec.RootDirectory != "" {
		return spec.RootDirectory
	}

	return filepath.Dir(findBackendRoot())
}

func getPublishedDatabaseDsnPath(spec LoadSpec) string {
	if spec.PublishedDatabaseDsnPath != "" {
		return spec.PublishedDatabaseDsnPath
	}

	return defaultPublishedDatabaseDsnPath
}

// findBackendRoot returns the directory holding go.mod, or the filesystem root when there is none,
// as in the image.
func findBackendRoot() string {
	backendRoot, err := os.Getwd()
	if err != nil {
		return "."
	}

	for {
		if _, err := os.Stat(filepath.Join(backendRoot, "go.mod")); err == nil {
			return backendRoot
		}

		parent := filepath.Dir(backendRoot)
		if parent == backendRoot {
			return backendRoot
		}

		backendRoot = parent
	}
}
