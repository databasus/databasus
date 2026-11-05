package postgresqlphysical

import (
	"strings"
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
			name:         "refuses settings without a password",
			change:       func(s *Settings) { s.Password = "" },
			refusedField: "settings.password",
		},
		{
			name:         "refuses an unknown backup type",
			change:       func(s *Settings) { s.BackupType = "DIFFERENTIAL" },
			refusedField: "settings.backupType",
		},
		{
			name: "refuses the embedded internal cluster of Databasus",
			change: func(s *Settings) {
				s.Host = "127.0.0.1"
				s.Port = 5437
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

	t.Run("keeps the slots, the cluster identity and the password the incoming settings leave out", func(t *testing.T) {
		t.Parallel()
		storedSettings := newSettings()
		storedSettings.ReplicationSlotName = "databasus_slot_stored"
		storedSettings.BackupSlotName = "databasus_basebackup_stored"
		storedSettings.SystemIdentifier = new("7300000000000000001")
		storedSettings.WalSegmentSizeBytes = new(int64(16 * 1024 * 1024))
		incomingSettings := newSettings()
		incomingSettings.Password = ""
		incomingSettings.BackupType = providers.PhysicalBackupTypeFullAndIncremental

		mergedSettings, err := newProvider(t).MergeStoredSettings(providers.SettingsMerge{
			Incoming: incomingSettings,
			Stored:   storedSettings,
		})

		require.NoError(t, err)
		physicalSettings, isSettings := mergedSettings.(Settings)
		require.True(t, isSettings)
		assert.Equal(t, "stored-password", physicalSettings.Password)
		assert.Equal(t, "databasus_slot_stored", physicalSettings.ReplicationSlotName)
		assert.Equal(t, "databasus_basebackup_stored", physicalSettings.BackupSlotName)
		assert.Equal(t, "7300000000000000001", *physicalSettings.SystemIdentifier)
		assert.Equal(t, int64(16*1024*1024), *physicalSettings.WalSegmentSizeBytes)
		assert.Equal(t, providers.PhysicalBackupTypeFullAndIncremental, physicalSettings.BackupType)
	})
}

func TestCopyForNewDatabase(t *testing.T) {
	t.Parallel()

	t.Run("gives the copy its own slots and forgets the cluster identity", func(t *testing.T) {
		t.Parallel()
		originalSettings := newSettings()
		originalSettings.ReplicationSlotName = "databasus_slot_original"
		originalSettings.BackupSlotName = "databasus_basebackup_original"
		originalSettings.SystemIdentifier = new("7300000000000000001")

		copiedSettings, isSettings := newProvider(t).CopyForNewDatabase(originalSettings).(Settings)
		require.True(t, isSettings)

		assert.NotEqual(t, "databasus_slot_original", copiedSettings.ReplicationSlotName)
		assert.True(t, strings.HasPrefix(copiedSettings.ReplicationSlotName, "databasus_slot_"))
		assert.NotEqual(t, "databasus_basebackup_original", copiedSettings.BackupSlotName)
		assert.True(t, strings.HasPrefix(copiedSettings.BackupSlotName, "databasus_basebackup_"))
		assert.Nil(t, copiedSettings.SystemIdentifier)
		assert.Equal(t, "stored-password", copiedSettings.Password)
	})
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
		Host:       "db.example.com",
		Port:       5432,
		Username:   "replicator",
		Password:   "stored-password",
		TLS:        providers.PostgresqlTLSSettings{SslMode: pgshared.SslModeDisable},
		BackupType: providers.PhysicalBackupTypeFull,
	}
}
