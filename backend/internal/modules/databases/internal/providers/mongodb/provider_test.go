package mongodb

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
			name:   "accepts an SRV connection without a port",
			change: func(s *Settings) { s.IsSrv = true; s.Port = 0 },
		},
		{
			name:         "refuses a standard connection without a port",
			change:       func(s *Settings) { s.Port = 0 },
			refusedField: "settings.port",
		},
		{
			name:         "refuses settings without a database",
			change:       func(s *Settings) { s.Database = "" },
			refusedField: "settings.database",
		},
		{
			name:         "refuses settings without a CPU",
			change:       func(s *Settings) { s.CPUCount = 0 },
			refusedField: "settings.cpuCount",
		},
		{
			name: "refuses an SSH tunnel with an SRV connection",
			change: func(s *Settings) {
				s.IsSrv = true
				s.SSHTunnel = providers.SSHTunnelSettings{
					IsEnabled: true, Host: "bastion", Port: 22, Username: "u",
					AuthType: providers.SSHAuthTypePassword, Password: "p",
				}
			},
			refusedField: "settings.sshTunnel.isEnabled",
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

	t.Run("keeps the stored password and version the incoming settings leave out", func(t *testing.T) {
		t.Parallel()
		storedSettings := newSettings()
		storedSettings.Version = "7.0"
		incomingSettings := newSettings()
		incomingSettings.Password = ""
		incomingSettings.ExcludeCollections = []string{"sessions"}

		mergedSettings, err := newProvider(t).MergeStoredSettings(providers.SettingsMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		mongodbSettings, isSettings := mergedSettings.(Settings)
		require.True(t, isSettings)
		assert.Equal(t, "stored-password", mongodbSettings.Password)
		assert.Equal(t, storedSettings.Version, mongodbSettings.Version)
		assert.Equal(t, []string{"sessions"}, mongodbSettings.ExcludeCollections)
	})
}

func newProvider(t *testing.T) *Provider {
	t.Helper()

	return New(
		Deps{Repositories: Repositories{settings: nil}, FieldEncryptor: providertest.NewFieldEncryptor(t), Logger: nil},
	)
}

func newSettings() Settings {
	return Settings{
		Host:         "db.example.com",
		Port:         27017,
		Username:     "backup",
		Password:     "stored-password",
		Database:     "shop",
		AuthDatabase: "admin",
		CPUCount:     1,
	}
}
