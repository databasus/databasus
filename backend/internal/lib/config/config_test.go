package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/mailer"
)

const (
	operatorDsn  = "host=operator user=postgres password=operator dbname=databasus port=5432 sslmode=disable"
	publishedDsn = "host=localhost user=postgres password=published dbname=databasus port=5437 sslmode=disable"
	fileDsn      = "host=file user=postgres password=file dbname=databasus port=5433 sslmode=disable"
)

func TestIsLogFileEnabled(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		logFileIsEnabled string
		isEnabled        bool
	}{
		{name: "enables the file when the variable is unset", logFileIsEnabled: "", isEnabled: true},
		{name: "disables the file with false in any case", logFileIsEnabled: " FALSE ", isEnabled: false},
		{name: "keeps the file with an unrecognised value", logFileIsEnabled: "nope", isEnabled: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			loadedConfig := Config{LogFileIsEnabled: testCase.logFileIsEnabled}

			assert.Equal(t, testCase.isEnabled, loadedConfig.IsLogFileEnabled())
		})
	}
}

func TestGetInstanceMailSettings(t *testing.T) {
	t.Parallel()

	t.Run("picks tls for port 465 and starttls for other ports without a security mode", func(t *testing.T) {
		t.Parallel()

		implicitTLSSettings := Config{SMTPHost: "smtp.example.com", SMTPPort: 465}.GetInstanceMailSettings()
		upgradeSettings := Config{SMTPHost: "smtp.example.com", SMTPPort: 587}.GetInstanceMailSettings()

		assert.Equal(t, mailer.SecurityTLS, implicitTLSSettings.Security)
		assert.Equal(t, mailer.SecurityStartTLS, upgradeSettings.Security)
	})

	t.Run("uses starttls without credentials for a relay without credentials", func(t *testing.T) {
		t.Parallel()

		relaySettings := Config{SMTPHost: "relay.example.com", SMTPPort: 25}.GetInstanceMailSettings()

		assert.Equal(t, mailer.SecurityStartTLS, relaySettings.Security)
		assert.Empty(t, relaySettings.Username)
		assert.Empty(t, relaySettings.Password)
	})

	t.Run("greets with the host of the public address without a greeting name", func(t *testing.T) {
		t.Parallel()

		mailSettings := Config{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			DatabasusURL: "https://backup.example.com",
		}.GetInstanceMailSettings()

		assert.Equal(t, "backup.example.com", mailSettings.HeloName)
	})

	t.Run("returns empty settings without a host", func(t *testing.T) {
		t.Parallel()

		mailSettings := Config{SMTPPort: 587, SMTPSecurity: mailer.SecurityTLS}.GetInstanceMailSettings()

		assert.Equal(t, mailer.Settings{}, mailSettings)
	})
}

func newIsolatedSpec(t *testing.T, envFileContents string) LoadSpec {
	t.Helper()

	rootDirectory := t.TempDir()
	if envFileContents != "" {
		envFilePath := filepath.Join(rootDirectory, ".env")
		require.NoError(t, os.WriteFile(envFilePath, []byte(envFileContents), 0o600))
	}

	return LoadSpec{
		IsDatabaseRequired:       true,
		RootDirectory:            rootDirectory,
		PublishedDatabaseDsnPath: filepath.Join(rootDirectory, "never-published"),
	}
}

func writePublishedDsn(t *testing.T, spec LoadSpec, contents string) {
	t.Helper()

	require.NoError(t, os.WriteFile(spec.PublishedDatabaseDsnPath, []byte(contents), 0o600))
}

func setRequiredVariables(t *testing.T) {
	t.Helper()

	t.Setenv("ENV_MODE", "development")
	t.Setenv(databaseDsnVariable, operatorDsn)
}

func unsetVariable(t *testing.T, name string) {
	t.Helper()

	t.Setenv(name, "")
	require.NoError(t, os.Unsetenv(name))
}

func getSortedEnvironment() []string {
	environment := os.Environ()
	slices.Sort(environment)

	return environment
}
