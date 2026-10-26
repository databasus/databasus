package postgresql_physical

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/testing/containers"
)

func Test_Validate_EmbeddedPhysicalDatabase_ReturnsError(t *testing.T) {
	physicalDatabase := validModel()
	physicalDatabase.Host = "/tmp"
	physicalDatabase.Port = 5437

	err := physicalDatabase.Validate()

	require.ErrorContains(t, err, "backing up Databasus internal PostgreSQL cluster is not allowed")
}

func validModel() *PostgresqlPhysicalDatabase {
	return &PostgresqlPhysicalDatabase{
		Version:    pgshared.Version17,
		Host:       "127.0.0.1",
		Port:       5432,
		Username:   "user",
		Password:   "pass",
		SslMode:    pgshared.SslModeDisable,
		BackupType: BackupTypeFullOnly,
	}
}

func Test_Validate_RejectsBlankHost(t *testing.T) {
	m := validModel()
	m.Host = ""

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsZeroPort(t *testing.T) {
	m := validModel()
	m.Port = 0

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsBlankUsername(t *testing.T) {
	m := validModel()
	m.Username = ""

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsBlankPassword(t *testing.T) {
	m := validModel()
	m.Password = ""

	assert.Error(t, m.Validate())
}

func Test_Validate_DelegatesSslConfigToShared(t *testing.T) {
	m := validModel()
	m.SslMode = "totally-invalid"

	assert.Error(t, m.Validate())
}

func Test_Validate_RejectsUnknownBackupType(t *testing.T) {
	m := validModel()
	m.BackupType = "GIBBERISH"

	assert.Error(t, m.Validate())
}

func Test_Validate_AcceptsAllThreeBackupTypes(t *testing.T) {
	for _, bt := range []BackupType{
		BackupTypeFullOnly,
		BackupTypeFullAndIncremental,
		BackupTypeFullIncrementalAndWalStream,
	} {
		m := validModel()
		m.BackupType = bt
		assert.NoErrorf(t, m.Validate(), "backup type %s", bt)
	}
}

func Test_BackupType_RequiresWalSummary(t *testing.T) {
	cases := map[BackupType]bool{
		BackupTypeFullOnly:                    false,
		BackupTypeFullAndIncremental:          true,
		BackupTypeFullIncrementalAndWalStream: true,
	}

	for backupType, expectsWalSummary := range cases {
		assert.Equal(
			t,
			expectsWalSummary,
			backupType.IsRequireWalSummary(),
			"backup type %s",
			backupType,
		)
	}
}

func Test_ValidateUpdate_RejectsSystemIdentifierChange(t *testing.T) {
	x := "111"
	y := "222"
	old := &PostgresqlPhysicalDatabase{SystemIdentifier: &x}
	next := &PostgresqlPhysicalDatabase{SystemIdentifier: &y}

	err := next.ValidateUpdate(old)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "immutable")
}

func Test_ValidateUpdate_AllowsSystemIdentifierFirstSet(t *testing.T) {
	first := "111"
	old := &PostgresqlPhysicalDatabase{SystemIdentifier: nil}
	next := &PostgresqlPhysicalDatabase{SystemIdentifier: &first}

	assert.NoError(t, next.ValidateUpdate(old))
}

func Test_ValidateUpdate_AllowsUnchangedSystemIdentifier(t *testing.T) {
	same := "111"
	old := &PostgresqlPhysicalDatabase{SystemIdentifier: &same}
	next := &PostgresqlPhysicalDatabase{SystemIdentifier: &same}

	assert.NoError(t, next.ValidateUpdate(old))
}

func Test_SystemIdentifierUint64(t *testing.T) {
	positiveSysID := "7361234567890123456" // fits int64, parses directly
	highBitSysID := "-1"                   // uint64 max stored as a signed bigint
	invalidSysID := "not-a-number"

	scenarios := []struct {
		name     string
		database *PostgresqlPhysicalDatabase
		systemID uint64
	}{
		{"unset yields zero", &PostgresqlPhysicalDatabase{SystemIdentifier: nil}, 0},
		{"positive value", &PostgresqlPhysicalDatabase{SystemIdentifier: &positiveSysID}, 7361234567890123456},
		{
			"high-bit value stored as negative bigint",
			&PostgresqlPhysicalDatabase{SystemIdentifier: &highBitSysID},
			^uint64(0),
		},
		{"unparseable yields zero", &PostgresqlPhysicalDatabase{SystemIdentifier: &invalidSysID}, 0},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			assert.Equal(t, scenario.systemID, scenario.database.SystemIdentifierUint64())
		})
	}
}

func Test_Update_PreservesPasswordWhenIncomingBlank(t *testing.T) {
	existing := validModel()
	existing.Password = "kept"

	incoming := validModel()
	incoming.Password = ""

	existing.Update(incoming)

	assert.Equal(t, "kept", existing.Password)
}

func Test_Update_OverwritesPasswordWhenIncomingNonBlank(t *testing.T) {
	existing := validModel()
	existing.Password = "old"

	incoming := validModel()
	incoming.Password = "new"

	existing.Update(incoming)

	assert.Equal(t, "new", existing.Password)
}

func Test_Update_PreservesSslClientKeyWhenIncomingBlank(t *testing.T) {
	existing := validModel()
	existing.SslMode = pgshared.SslModeRequire
	existing.SslClientKey = "kept-key"

	incoming := validModel()
	incoming.SslMode = pgshared.SslModeRequire
	incoming.SslClientKey = ""

	existing.Update(incoming)

	assert.Equal(t, "kept-key", existing.SslClientKey)
}

func Test_Update_NeverOverwritesSystemIdentifierFromIncoming(t *testing.T) {
	original := "111"
	existing := validModel()
	existing.SystemIdentifier = &original

	hostile := "222"
	incoming := validModel()
	incoming.SystemIdentifier = &hostile

	existing.Update(incoming)

	require.NotNil(t, existing.SystemIdentifier)
	assert.Equal(t, "111", *existing.SystemIdentifier)
}

func Test_HideSensitiveData_ZerosPasswordAndSslClientKey(t *testing.T) {
	m := validModel()
	m.Password = "secret"
	m.SslClientKey = "key"
	m.SslClientCert = "visible"

	m.HideSensitiveData()

	assert.Empty(t, m.Password)
	assert.Empty(t, m.SslClientKey)
	assert.Equal(t, "visible", m.SslClientCert)
}

func Test_HideSensitiveData_NilReceiver_DoesNotPanic(t *testing.T) {
	var m *PostgresqlPhysicalDatabase
	assert.NotPanics(t, func() {
		m.HideSensitiveData()
	})
}

type noopEncryptor struct{}

func (noopEncryptor) Encrypt(s string) (string, error) { return "enc:" + s, nil }
func (noopEncryptor) Decrypt(s string) (string, error) {
	if len(s) > 4 && s[:4] == "enc:" {
		return s[4:], nil
	}
	return s, nil
}

func Test_EncryptSensitiveFields_EncryptsAllFourFields(t *testing.T) {
	m := validModel()
	m.Password = "p"
	m.SslClientCert = "c"
	m.SslClientKey = "k"
	m.SslRootCert = "r"

	require.NoError(t, m.EncryptSensitiveFields(noopEncryptor{}))

	assert.Equal(t, "enc:p", m.Password)
	assert.Equal(t, "enc:c", m.SslClientCert)
	assert.Equal(t, "enc:k", m.SslClientKey)
	assert.Equal(t, "enc:r", m.SslRootCert)
}

func Test_EncryptSensitiveFields_SkipsBlankFields(t *testing.T) {
	m := validModel()
	m.Password = "p"
	m.SslClientCert = ""
	m.SslClientKey = ""
	m.SslRootCert = ""

	require.NoError(t, m.EncryptSensitiveFields(noopEncryptor{}))

	assert.Equal(t, "enc:p", m.Password)
	assert.Empty(t, m.SslClientCert)
	assert.Empty(t, m.SslClientKey)
	assert.Empty(t, m.SslRootCert)
}

func Test_BeforeCreate_FillsReplicationSlotNameIfBlank(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{}

	require.NoError(t, m.BeforeCreate(nil))

	assert.True(t,
		len(m.ReplicationSlotName) > len("databasus_slot_") &&
			m.ReplicationSlotName[:len("databasus_slot_")] == "databasus_slot_",
		"got %q", m.ReplicationSlotName,
	)
}

func Test_BeforeCreate_PreservesExistingReplicationSlotName(t *testing.T) {
	m := &PostgresqlPhysicalDatabase{ReplicationSlotName: "preserved_slot"}

	require.NoError(t, m.BeforeCreate(nil))

	assert.Equal(t, "preserved_slot", m.ReplicationSlotName)
}

var _ encryption.FieldEncryptor = noopEncryptor{}

// The old model reaches the cluster only through the engine; this shows that it writes what the
// engine detected into itself and keeps a system identifier it already holds.
func Test_PopulateDbData_WritesClusterIdentityFromTheEngine(t *testing.T) {
	source := containers.StartPhysicalPostgres(t, "postgres:17")

	newSourceModel := func() *PostgresqlPhysicalDatabase {
		m := validModel()
		m.Version = ""
		m.Host = source.Host
		m.Port = source.Port
		m.Username = containers.PostgresUsername
		m.Password = containers.PostgresPassword

		return m
	}

	t.Run("fills version, system identifier and wal segment size on first call", func(t *testing.T) {
		m := newSourceModel()

		require.NoError(t, m.PopulateDbData(t.Context(), slog.New(slog.NewTextHandler(t.Output(), nil)), nil))

		assert.Equal(t, pgshared.Version17, m.Version)
		require.NotNil(t, m.SystemIdentifier)
		assert.NotEmpty(t, *m.SystemIdentifier)
		require.NotNil(t, m.WalSegmentSizeBytes)
		assert.Equal(t, int64(16*1024*1024), *m.WalSegmentSizeBytes)
	})

	t.Run("keeps the stored system identifier", func(t *testing.T) {
		m := newSourceModel()
		storedSystemIdentifier := "9999999999999999999"
		m.SystemIdentifier = &storedSystemIdentifier

		require.NoError(t, m.PopulateDbData(t.Context(), slog.New(slog.NewTextHandler(t.Output(), nil)), nil))

		require.NotNil(t, m.SystemIdentifier)
		assert.Equal(t, storedSystemIdentifier, *m.SystemIdentifier)
	})
}
