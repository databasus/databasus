package mysql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/providers/providertest"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		change       func(settings *Settings)
		refusedField string
	}{
		{name: "accepts complete settings", change: func(*Settings) {}},
		{
			name:         "refuses settings without a host",
			change:       func(s *Settings) { s.Host = "" },
			refusedField: "settings.host",
		},
		{
			name:         "refuses settings without a port",
			change:       func(s *Settings) { s.Port = 0 },
			refusedField: "settings.port",
		},
		{
			name:         "refuses settings without a password",
			change:       func(s *Settings) { s.Password = "" },
			refusedField: "settings.password",
		},
		{
			name:         "refuses an enabled tunnel without a username",
			change:       func(s *Settings) { s.SSHTunnel = providers.SSHTunnelSettings{IsEnabled: true, Host: "b", Port: 22} },
			refusedField: "settings.sshTunnel.username",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			validatedSettings := newSettings()
			testCase.change(&validatedSettings)

			err := newProvider(t).Validate(validatedSettings)

			if testCase.refusedField == "" {
				require.NoError(t, err)

				return
			}

			assert.Equal(t, testCase.refusedField, providertest.GetRefusedField(t, err))
		})
	}
}

func TestMergeStoredSettings(t *testing.T) {
	t.Parallel()

	t.Run("keeps the stored password, version and privileges the incoming settings leave out", func(t *testing.T) {
		t.Parallel()
		storedSettings := newSettings()
		storedSettings.Version = "8.0"
		storedSettings.Privileges = []string{"SELECT", "SHOW VIEW"}
		incomingSettings := newSettings()
		incomingSettings.Password = ""
		incomingSettings.ExcludeTables = []string{"audit_log"}

		mergedSettings, err := newProvider(t).MergeStoredSettings(providers.SettingsMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		mysqlSettings, isSettings := mergedSettings.(Settings)
		require.True(t, isSettings)
		assert.Equal(t, "stored-password", mysqlSettings.Password)
		assert.Equal(t, storedSettings.Version, mysqlSettings.Version)
		assert.Equal(t, []string{"SELECT", "SHOW VIEW"}, mysqlSettings.Privileges)
		assert.Equal(t, []string{"audit_log"}, mysqlSettings.ExcludeTables)
	})
}

func newProvider(t *testing.T) *Provider {
	t.Helper()

	return New(Deps{
		Repositories:   Repositories{settings: nil},
		FieldEncryptor: providertest.NewFieldEncryptor(t),
		Logger:         nil,
		ClientToolsDir: "",
	})
}

func newSettings() Settings {
	return Settings{
		Host:     "db.example.com",
		Port:     3306,
		Username: "backup",
		Password: "stored-password",
		Database: "shop",
	}
}
