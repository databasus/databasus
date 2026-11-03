package config

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		t.Setenv("SMTP_HOST", "")
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
		t.Setenv("SMTP_HOST", "")
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

func TestApplicationVersion(t *testing.T) {
	for _, scenario := range []struct {
		name       string
		configured string
		reported   string
	}{
		{"keeps the development default when APP_VERSION is unset", "", "dev"},
		{"uses the configured version when APP_VERSION is set", "9.8.7", "9.8.7"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			setRequiredVariables(t)
			t.Setenv("APP_VERSION", scenario.configured)
			loadedConfig, _, err := Load(newIsolatedSpec(t, ""))
			require.NoError(t, err)
			assert.Equal(t, scenario.reported, loadedConfig.AppVersion)
		})
	}
}
