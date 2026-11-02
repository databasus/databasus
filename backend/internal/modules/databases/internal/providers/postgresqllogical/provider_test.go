package postgresqllogical

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
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
			name:         "refuses settings without a username",
			change:       func(s *Settings) { s.Username = "" },
			refusedField: "settings.username",
		},
		{
			name:         "refuses settings without a password",
			change:       func(s *Settings) { s.Password = "" },
			refusedField: "settings.password",
		},
		{
			name:         "refuses settings without a CPU",
			change:       func(s *Settings) { s.CPUCount = 0 },
			refusedField: "settings.cpuCount",
		},
		{
			name:         "refuses a client certificate without SSL",
			change:       func(s *Settings) { s.TLS.ClientCert = "cert"; s.TLS.ClientKey = "key" },
			refusedField: "settings.sslMode",
		},
		{
			name:         "refuses an enabled tunnel without a host",
			change:       func(s *Settings) { s.SSHTunnel = providers.SSHTunnelSettings{IsEnabled: true} },
			refusedField: "settings.sshTunnel.host",
		},
		{
			name: "refuses the embedded internal database of Databasus",
			change: func(s *Settings) {
				s.Host = "localhost"
				s.Database = "databasus"
			},
			refusedField: "settings.host",
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

	t.Run("keeps the stored secrets and version when the incoming settings leave them out", func(t *testing.T) {
		t.Parallel()
		storedSettings := newSettings()
		storedSettings.Version = pgshared.Version("16")
		storedSettings.TLS = providers.PostgresqlTLSSettings{
			SslMode:    pgshared.SslModeRequire,
			ClientCert: "stored-cert",
			ClientKey:  "stored-key",
		}
		incomingSettings := storedSettings
		incomingSettings.Version = ""
		incomingSettings.Password = ""
		incomingSettings.TLS.ClientKey = ""
		incomingSettings.Username = "new-user"

		mergedSettings, err := newProvider(t).MergeStoredSettings(providers.SettingsMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		logicalSettings, isSettings := mergedSettings.(Settings)
		require.True(t, isSettings)
		assert.Equal(t, "stored-password", logicalSettings.Password)
		assert.Equal(t, "stored-key", logicalSettings.TLS.ClientKey)
		assert.Equal(t, pgshared.Version("16"), logicalSettings.Version)
		assert.Equal(t, "new-user", logicalSettings.Username)
	})

	t.Run("leaves the secrets empty when the port changes", func(t *testing.T) {
		t.Parallel()
		storedSettings := newSettings()
		storedSettings.TLS = providers.PostgresqlTLSSettings{
			SslMode:    pgshared.SslModeRequire,
			ClientCert: "stored-cert",
			ClientKey:  "stored-key",
		}
		incomingSettings := storedSettings
		incomingSettings.Password = ""
		incomingSettings.TLS.ClientKey = ""
		incomingSettings.Port = 6543

		mergedSettings, err := newProvider(t).MergeStoredSettings(providers.SettingsMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		logicalSettings, isSettings := mergedSettings.(Settings)
		require.True(t, isSettings)
		assert.Empty(t, logicalSettings.Password)
		assert.Empty(t, logicalSettings.TLS.ClientKey)
	})

	t.Run("refuses settings of another type", func(t *testing.T) {
		t.Parallel()

		_, err := newProvider(t).MergeStoredSettings(providers.SettingsMerge{
			Incoming: otherTypeSettings{},
			Stored:   newSettings(),
		})

		require.ErrorIs(t, err, providers.ErrSettingsOfAnotherType)
	})
}

type otherTypeSettings struct{}

func (otherTypeSettings) GetType() providers.Type {
	return providers.TypeMongodb
}

func newProvider(t *testing.T) *Provider {
	t.Helper()

	return New(Deps{
		Repositories:   Repositories{settings: nil},
		FieldEncryptor: providertest.NewFieldEncryptor(t),
		Logger:         nil,
	})
}

func newSettings() Settings {
	return Settings{
		Host:     "db.example.com",
		Port:     5432,
		Username: "backup",
		Password: "stored-password",
		Database: "shop",
		TLS:      providers.PostgresqlTLSSettings{SslMode: pgshared.SslModeDisable},
		CPUCount: 1,
	}
}
