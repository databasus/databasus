package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	operatorDsn  = "host=operator user=postgres password=operator dbname=databasus port=5432 sslmode=disable"
	publishedDsn = "host=localhost user=postgres password=published dbname=databasus port=5437 sslmode=disable"
	fileDsn      = "host=file user=postgres password=file dbname=databasus port=5433 sslmode=disable"
)

// The tests change the process environment through t.Setenv, so none of them runs in parallel.
func TestLoad(t *testing.T) {
	t.Run("names the variable when a required one is empty", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		t.Setenv("ENV_MODE", "development")
		t.Setenv(databaseDsnVariable, "")

		_, _, err := Load(spec)

		require.ErrorIs(t, err, ErrRequiredVariableNotSet)
		assert.Contains(t, err.Error(), databaseDsnVariable)
	})

	t.Run("names the variable when a required one is missing", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		unsetVariable(t, "ENV_MODE")
		t.Setenv(databaseDsnVariable, operatorDsn)

		_, _, err := Load(spec)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "ENV_MODE")
	})

	t.Run("keeps the process value when .env sets another one", func(t *testing.T) {
		spec := newIsolatedSpec(t, "ENV_MODE=development\nDATABASE_DSN="+fileDsn+"\n")
		t.Setenv(databaseDsnVariable, operatorDsn)

		loadedConfig, report, err := Load(spec)

		require.NoError(t, err)
		assert.True(t, report.IsEnvFileLoaded)
		assert.Equal(t, operatorDsn, loadedConfig.DatabaseDsn)
	})

	t.Run("loads with an empty SMTP_PORT", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		setRequiredVariables(t)
		t.Setenv("SMTP_PORT", "")

		loadedConfig, _, err := Load(spec)

		require.NoError(t, err)
		assert.Zero(t, loadedConfig.SMTPPort)
	})

	t.Run("uses the published connection string when DATABASE_DSN is empty", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		writePublishedDsn(t, spec, publishedDsn+"\n")
		t.Setenv("ENV_MODE", "production")
		t.Setenv(databaseDsnVariable, "")

		loadedConfig, report, err := Load(spec)

		require.NoError(t, err)
		assert.True(t, report.IsPublishedDatabaseDsnUsed)
		assert.Equal(t, publishedDsn, loadedConfig.DatabaseDsn)
	})

	t.Run("keeps the operator connection string over the published one", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		writePublishedDsn(t, spec, publishedDsn)
		setRequiredVariables(t)

		loadedConfig, report, err := Load(spec)

		require.NoError(t, err)
		assert.False(t, report.IsPublishedDatabaseDsnUsed)
		assert.Equal(t, operatorDsn, loadedConfig.DatabaseDsn)
	})

	t.Run("fills a variable set empty in the process from .env", func(t *testing.T) {
		spec := newIsolatedSpec(t, "ENV_MODE=development\nDATABASE_DSN="+fileDsn+"\n")
		t.Setenv("ENV_MODE", "")
		t.Setenv(databaseDsnVariable, "")

		loadedConfig, _, err := Load(spec)

		require.NoError(t, err)
		assert.Equal(t, fileDsn, loadedConfig.DatabaseDsn)
	})

	t.Run("leaves the process environment unchanged after loading", func(t *testing.T) {
		spec := newIsolatedSpec(t, "ENV_MODE=development\nDATABASUS_ONLY_IN_ENV_FILE=from-file\n")
		writePublishedDsn(t, spec, publishedDsn)
		unsetVariable(t, "ENV_MODE")
		t.Setenv(databaseDsnVariable, "")
		t.Setenv("SMTP_PORT", "")
		environmentBeforeLoad := getSortedEnvironment()

		loadedConfig, _, err := Load(spec)

		require.NoError(t, err)
		assert.Equal(t, publishedDsn, loadedConfig.DatabaseDsn)
		assert.Equal(t, environmentBeforeLoad, getSortedEnvironment())
	})

	t.Run("loads without a connection string when the database is not required", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		spec.IsDatabaseRequired = false
		t.Setenv("ENV_MODE", "production")
		unsetVariable(t, databaseDsnVariable)

		loadedConfig, report, err := Load(spec)

		require.NoError(t, err)
		assert.False(t, report.IsEnvFileLoaded)
		assert.Empty(t, loadedConfig.DatabaseDsn)
	})

	t.Run("derives every data path from the data directory", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		setRequiredVariables(t)

		loadedConfig, _, err := Load(spec)

		require.NoError(t, err)

		dataDirectory := filepath.Join(spec.RootDirectory, "databasus-data")
		assert.Equal(t, dataDirectory, loadedConfig.DataDirectory)
		assert.Equal(t, filepath.Join(dataDirectory, "backups"), loadedConfig.DataFolder)
		assert.Equal(t, filepath.Join(dataDirectory, "temp"), loadedConfig.TempFolder)
		assert.Equal(t, filepath.Join(dataDirectory, "secret.key"), loadedConfig.SecretKeyPath)
		assert.Equal(t, filepath.Join(dataDirectory, "databasus.log"), loadedConfig.LogFilePath)
	})

	t.Run("rejects an unknown ENV_MODE", func(t *testing.T) {
		spec := newIsolatedSpec(t, "")
		setRequiredVariables(t)
		t.Setenv("ENV_MODE", "staging")

		_, _, err := Load(spec)

		require.ErrorIs(t, err, ErrInvalidVariable)
		assert.Contains(t, err.Error(), "ENV_MODE")
	})

	t.Run("returns an error with a malformed .env", func(t *testing.T) {
		spec := newIsolatedSpec(t, "not a variable assignment\n")
		setRequiredVariables(t)

		_, _, err := Load(spec)

		require.Error(t, err)
	})
}

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

func TestValidateSMTPSettings(t *testing.T) {
	t.Parallel()

	rejectedCases := []struct {
		name             string
		smtpConfig       Config
		expectedVariable string
	}{
		{
			name:             "rejects a host without a port",
			smtpConfig:       Config{SMTPHost: "smtp.example.com"},
			expectedVariable: "SMTP_PORT",
		},
		{
			name:             "rejects an unknown security mode",
			smtpConfig:       Config{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPSecurity: "ssl"},
			expectedVariable: "SMTP_SECURITY",
		},
		{
			name:             "rejects a greeting name with a space",
			smtpConfig:       Config{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPHeloName: "my relay"},
			expectedVariable: "SMTP_HELO_NAME",
		},
		{
			name:             "rejects an unreadable sender",
			smtpConfig:       Config{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPFrom: "noreply at example.com"},
			expectedVariable: "SMTP_FROM",
		},
	}

	for _, rejectedCase := range rejectedCases {
		t.Run(rejectedCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateSMTPSettings(rejectedCase.smtpConfig)

			require.Error(t, err)
			assert.Contains(t, err.Error(), rejectedCase.expectedVariable)
		})
	}

	t.Run("accepts a sender with a display name", func(t *testing.T) {
		t.Parallel()

		err := validateSMTPSettings(Config{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPFrom:     "Acme Backups <noreply@example.com>",
			SMTPSecurity: "none",
			SMTPHeloName: "[IPv6:2001:db8::1]",
		})

		require.NoError(t, err)
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
