package postgresql_logical

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/testing/containers"
)

// The engine tests live in internal/engines/postgresql/logical; this one shows the old model reaches
// its engine and writes the detected version back.
func Test_TestConnection_WhenServerIsReachable_PopulatesVersionThroughEngine(t *testing.T) {
	endpoint := containers.StartPostgres(t, "postgres:16")
	databaseName := containers.PostgresDatabase

	pgModel := &PostgresqlLogicalDatabase{
		Host:     endpoint.Host,
		Port:     endpoint.Port,
		Username: containers.PostgresUsername,
		Password: containers.PostgresPassword,
		Database: &databaseName,
		SslMode:  pgshared.SslModeDisable,
		CpuCount: 1,
	}

	err := pgModel.TestConnection(t.Context(), slog.New(slog.NewTextHandler(os.Stdout, nil)), nil)
	require.NoError(t, err)
	assert.Equal(t, pgshared.GetVersionEnum("16"), pgModel.Version)
}

func Test_Validate_WhenLocalhostAndDatabasus_ReturnsError(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		username string
		database string
	}{
		{
			name:     "localhost with databasus db",
			host:     "localhost",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "127.0.0.1 with databasus db",
			host:     "127.0.0.1",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "172.17.0.1 with databasus db",
			host:     "172.17.0.1",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "host.docker.internal with databasus db",
			host:     "host.docker.internal",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "LOCALHOST (uppercase) with DATABASUS db",
			host:     "LOCALHOST",
			username: "POSTGRES",
			database: "DATABASUS",
		},
		{
			name:     "LocalHost (mixed case) with DataBasus db",
			host:     "LocalHost",
			username: "anyuser",
			database: "DataBasus",
		},
		{
			name:     "localhost with databasus and any username",
			host:     "localhost",
			username: "myuser",
			database: "databasus",
		},
		{
			name:     "::1 (IPv6 loopback) with databasus db",
			host:     "::1",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     ":: (IPv6 all interfaces) with databasus db",
			host:     "::",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "::1 (uppercase) with DATABASUS db",
			host:     "::1",
			username: "POSTGRES",
			database: "DATABASUS",
		},
		{
			name:     "0.0.0.0 (all IPv4 interfaces) with databasus db",
			host:     "0.0.0.0",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "127.0.0.2 (loopback range) with databasus db",
			host:     "127.0.0.2",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "127.255.255.255 (end of loopback range) with databasus db",
			host:     "127.255.255.255",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "Unix socket with databasus db",
			host:     "/tmp",
			username: "postgres",
			database: "databasus",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgModel := &PostgresqlLogicalDatabase{
				Host:     tc.host,
				Port:     5437,
				Username: tc.username,
				Password: "somepassword",
				Database: &tc.database,
				CpuCount: 1,
			}

			err := pgModel.Validate()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "backing up Databasus internal database is not allowed")
			assert.Contains(t, err.Error(), "https://databasus.com/faq#backup-databasus")
		})
	}
}

func Test_Validate_WhenNotLocalhostOrNotDatabasus_ValidatesSuccessfully(t *testing.T) {
	testCases := []struct {
		name     string
		host     string
		username string
		database string
	}{
		{
			name:     "different host (remote server) with databasus db",
			host:     "192.168.1.100",
			username: "postgres",
			database: "databasus",
		},
		{
			name:     "different database name on localhost",
			host:     "localhost",
			username: "postgres",
			database: "myapp",
		},
		{
			name:     "all different",
			host:     "db.example.com",
			username: "appuser",
			database: "production",
		},
		{
			name:     "localhost with postgres database",
			host:     "localhost",
			username: "postgres",
			database: "postgres",
		},
		{
			name:     "remote host with databasus db name (allowed)",
			host:     "db.example.com",
			username: "postgres",
			database: "databasus",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pgModel := &PostgresqlLogicalDatabase{
				Host:     tc.host,
				Port:     5432,
				Username: tc.username,
				Password: "somepassword",
				Database: &tc.database,
				CpuCount: 1,
			}

			err := pgModel.Validate()
			assert.NoError(t, err)
		})
	}
}

func Test_Validate_WhenDatabaseIsNil_ValidatesSuccessfully(t *testing.T) {
	pgModel := &PostgresqlLogicalDatabase{
		Host:     "localhost",
		Port:     5437,
		Username: "postgres",
		Password: "somepassword",
		Database: nil,
		CpuCount: 1,
	}

	err := pgModel.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenDatabaseIsEmpty_ValidatesSuccessfully(t *testing.T) {
	emptyDb := ""
	pgModel := &PostgresqlLogicalDatabase{
		Host:     "localhost",
		Port:     5437,
		Username: "postgres",
		Password: "somepassword",
		Database: &emptyDb,
		CpuCount: 1,
	}

	err := pgModel.Validate()
	assert.NoError(t, err)
}

func Test_Validate_WhenRequiredFieldsMissing_ReturnsError(t *testing.T) {
	testCases := []struct {
		name          string
		model         *PostgresqlLogicalDatabase
		expectedError string
	}{
		{
			name: "missing host",
			model: &PostgresqlLogicalDatabase{
				Host:     "",
				Port:     5432,
				Username: "user",
				Password: "pass",
				CpuCount: 1,
			},
			expectedError: "host is required",
		},
		{
			name: "missing port",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     0,
				Username: "user",
				Password: "pass",
				CpuCount: 1,
			},
			expectedError: "port is required",
		},
		{
			name: "missing username",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     5432,
				Username: "",
				Password: "pass",
				CpuCount: 1,
			},
			expectedError: "username is required",
		},
		{
			name: "missing password",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "",
				CpuCount: 1,
			},
			expectedError: "password is required",
		},
		{
			name: "invalid cpu count",
			model: &PostgresqlLogicalDatabase{
				Host:     "localhost",
				Port:     5432,
				Username: "user",
				Password: "pass",
				CpuCount: 0,
			},
			expectedError: "cpu count must be greater than 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.model.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func Test_HideSensitiveData_WhenCalled_ClearsPasswordAndPreservesOtherFields(t *testing.T) {
	databaseName := "appdb"
	pgModel := &PostgresqlLogicalDatabase{
		Version:        pgshared.GetVersionEnum("16"),
		Host:           "db.example.com",
		Port:           5432,
		Username:       "appuser",
		Password:       "supersecret",
		Database:       &databaseName,
		SslMode:        pgshared.SslModeRequire,
		IncludeSchemas: []string{"public"},
		ExcludeTables:  []string{"audit_logs"},
		CpuCount:       4,
	}

	pgModel.HideSensitiveData()

	assert.Empty(t, pgModel.Password)
	assert.Equal(t, "db.example.com", pgModel.Host)
	assert.Equal(t, 5432, pgModel.Port)
	assert.Equal(t, "appuser", pgModel.Username)
	assert.Equal(t, &databaseName, pgModel.Database)
	assert.Equal(t, pgshared.SslModeRequire, pgModel.SslMode)
	assert.Equal(t, []string{"public"}, pgModel.IncludeSchemas)
	assert.Equal(t, []string{"audit_logs"}, pgModel.ExcludeTables)
	assert.Equal(t, 4, pgModel.CpuCount)
}

func Test_HideSensitiveData_WhenReceiverIsNil_DoesNotPanic(t *testing.T) {
	var pgModel *PostgresqlLogicalDatabase

	assert.NotPanics(t, func() {
		pgModel.HideSensitiveData()
	})
}

type stubEncryptor struct{}

func (stubEncryptor) Encrypt(value string) (string, error) {
	if value == "" || strings.HasPrefix(value, "enc:") {
		return value, nil
	}

	return "enc:" + value, nil
}

func (stubEncryptor) Decrypt(value string) (string, error) {
	return strings.TrimPrefix(value, "enc:"), nil
}

func Test_HideSensitiveData_WhenCertsSet_ClearsOnlyClientKey(t *testing.T) {
	databaseName := "appdb"
	pgModel := &PostgresqlLogicalDatabase{
		Host:          "db.example.com",
		Port:          5432,
		Username:      "appuser",
		Password:      "supersecret",
		Database:      &databaseName,
		SslMode:       pgshared.SslModeVerifyFull,
		SslClientCert: "client-cert-pem",
		SslClientKey:  "client-key-pem",
		SslRootCert:   "root-cert-pem",
		CpuCount:      1,
	}

	pgModel.HideSensitiveData()

	assert.Empty(t, pgModel.Password)
	assert.Empty(t, pgModel.SslClientKey)
	assert.Equal(t, "client-cert-pem", pgModel.SslClientCert)
	assert.Equal(t, "root-cert-pem", pgModel.SslRootCert)
	assert.Equal(t, pgshared.SslModeVerifyFull, pgModel.SslMode)
}

func Test_EncryptSensitiveFields_WhenCertsSet_EncryptsEverySecret(t *testing.T) {
	pgModel := &PostgresqlLogicalDatabase{
		Password:      "supersecret",
		SslMode:       pgshared.SslModeVerifyFull,
		SslClientCert: "client-cert-pem",
		SslClientKey:  "client-key-pem",
		SslRootCert:   "root-cert-pem",
	}

	err := pgModel.EncryptSensitiveFields(stubEncryptor{})
	assert.NoError(t, err)
	assert.Equal(t, "enc:supersecret", pgModel.Password)
	assert.Equal(t, "enc:client-cert-pem", pgModel.SslClientCert)
	assert.Equal(t, "enc:client-key-pem", pgModel.SslClientKey)
	assert.Equal(t, "enc:root-cert-pem", pgModel.SslRootCert)

	err = pgModel.EncryptSensitiveFields(stubEncryptor{})
	assert.NoError(t, err)
	assert.Equal(t, "enc:supersecret", pgModel.Password)
	assert.Equal(t, "enc:client-key-pem", pgModel.SslClientKey)
}

func Test_EncryptSensitiveFields_WhenCertsEmpty_LeavesThemEmpty(t *testing.T) {
	pgModel := &PostgresqlLogicalDatabase{Password: "supersecret"}

	err := pgModel.EncryptSensitiveFields(stubEncryptor{})
	assert.NoError(t, err)
	assert.Equal(t, "enc:supersecret", pgModel.Password)
	assert.Empty(t, pgModel.SslClientCert)
	assert.Empty(t, pgModel.SslClientKey)
	assert.Empty(t, pgModel.SslRootCert)
}

func Test_Update_WhenIncomingClientKeyEmpty_PreservesExistingKey(t *testing.T) {
	databaseName := "appdb"
	existing := &PostgresqlLogicalDatabase{
		Host:          "old-host",
		Port:          5432,
		Username:      "olduser",
		Password:      "enc:oldpass",
		Database:      &databaseName,
		SslMode:       pgshared.SslModeVerifyCA,
		SslClientCert: "enc:old-cert",
		SslClientKey:  "enc:old-key",
		SslRootCert:   "enc:old-root",
		CpuCount:      1,
	}

	incoming := &PostgresqlLogicalDatabase{
		Host:          "new-host",
		Port:          5433,
		Username:      "newuser",
		Database:      &databaseName,
		SslMode:       pgshared.SslModeRequire,
		SslClientCert: "new-cert",
		SslClientKey:  "",
		SslRootCert:   "new-root",
		CpuCount:      2,
	}

	existing.Update(incoming)

	assert.Equal(t, pgshared.SslModeRequire, existing.SslMode)
	assert.Equal(t, "new-cert", existing.SslClientCert)
	assert.Equal(t, "enc:old-key", existing.SslClientKey)
	assert.Equal(t, "new-root", existing.SslRootCert)
}

func Test_Validate_SslConfig_EnforcesCertRules(t *testing.T) {
	databaseName := "appdb"
	baseModel := func() *PostgresqlLogicalDatabase {
		return &PostgresqlLogicalDatabase{
			Host:     "db.example.com",
			Port:     5432,
			Username: "appuser",
			Password: "secret",
			Database: &databaseName,
			CpuCount: 1,
		}
	}

	testCases := []struct {
		name          string
		mutate        func(*PostgresqlLogicalDatabase)
		expectedError string
	}{
		{
			name: "client cert without key",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = pgshared.SslModeRequire
				p.SslClientCert = "cert"
			},
			expectedError: "client certificate and client key must be provided together",
		},
		{
			name: "client key without cert",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = pgshared.SslModeRequire
				p.SslClientKey = "key"
			},
			expectedError: "client certificate and client key must be provided together",
		},
		{
			name: "certificates with SSL disabled",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = pgshared.SslModeDisable
				p.SslClientCert = "cert"
				p.SslClientKey = "key"
			},
			expectedError: "SSL certificates require SSL to be enabled",
		},
		{
			name: "unknown SSL mode",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = "bogus"
			},
			expectedError: "invalid SSL mode",
		},
		{
			name: "client certificate pair with verify-full",
			mutate: func(p *PostgresqlLogicalDatabase) {
				p.SslMode = pgshared.SslModeVerifyFull
				p.SslClientCert = "cert"
				p.SslClientKey = "key"
				p.SslRootCert = "root"
			},
			expectedError: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			model := baseModel()
			tc.mutate(model)

			err := model.Validate()

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}
		})
	}
}

func Test_Validate_WhenSslModeEmpty_DefaultsToDisable(t *testing.T) {
	databaseName := "appdb"
	model := &PostgresqlLogicalDatabase{
		Host:     "db.example.com",
		Port:     5432,
		Username: "appuser",
		Password: "secret",
		Database: &databaseName,
		CpuCount: 1,
	}

	err := model.Validate()

	assert.NoError(t, err)
	assert.Equal(t, pgshared.SslModeDisable, model.SslMode)
}

func databasusOnLoopback() *PostgresqlLogicalDatabase {
	databaseName := "databasus"

	return &PostgresqlLogicalDatabase{
		Host:     "127.0.0.1",
		Port:     5432,
		Username: "postgres",
		Password: "password",
		Database: &databaseName,
		CpuCount: 1,
	}
}

func Test_Validate_WhenPointedAtDatabasusOnLoopback_IsRejected(t *testing.T) {
	assert.Error(t, databasusOnLoopback().Validate())
}

func Test_Validate_WhenDatabasusOnLoopbackIsBehindARemoteBastion_IsAllowed(t *testing.T) {
	database := databasusOnLoopback()
	database.SshTunnel = sshtunnel.Config{
		IsEnabled: true,
		Host:      "bastion.example.com",
		Port:      22,
		Username:  "tunneluser",
		AuthType:  sshtunnel.AuthTypePassword,
		Password:  "tunnelpassword",
	}

	assert.NoError(t, database.Validate())
}

// A bastion on this machine forwards straight back to the instance the guard protects, so enabling
// a tunnel must not be a way around it.
func Test_Validate_WhenDatabasusOnLoopbackIsBehindALocalBastion_IsRejected(t *testing.T) {
	for _, bastionHost := range []string{"127.0.0.1", "localhost", "host.docker.internal", "127.5.5.5"} {
		database := databasusOnLoopback()
		database.SshTunnel = sshtunnel.Config{
			IsEnabled: true,
			Host:      bastionHost,
			Port:      22,
			Username:  "tunneluser",
			AuthType:  sshtunnel.AuthTypePassword,
			Password:  "tunnelpassword",
		}

		assert.Error(t, database.Validate(), "bastion host %q must not bypass the guard", bastionHost)
	}
}
