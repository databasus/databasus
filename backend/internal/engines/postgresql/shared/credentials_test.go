package shared

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildConnConfig(t *testing.T) {
	t.Parallel()

	t.Run("treats credential fields as literal values when they hold conninfo syntax", func(t *testing.T) {
		t.Parallel()
		spec := CredentialSpec{
			Host:     "db host=attacker\\socket'one",
			HostAddr: "",
			Port:     5432,
			Username: "user name=postgres\\role'one",
			TLS:      TLS{Mode: SslModeDisable},
		}
		databaseName := "postgres dbname=databasus\\archive'one"
		password := "secret password=overridden\\value'one"

		connConfig, err := BuildConnConfig(spec, password, databaseName, nil)

		require.NoError(t, err)
		assert.Equal(t, spec.Host, connConfig.Host)
		assert.Equal(t, spec.Username, connConfig.User)
		assert.Equal(t, databaseName, connConfig.Database)
		assert.Equal(t, password, connConfig.Password)
	})
}

func TestBuildDatabaseNameConninfo(t *testing.T) {
	t.Parallel()

	t.Run("treats a duplicate parameter as a literal database name", func(t *testing.T) {
		t.Parallel()
		databaseName := "postgres dbname=databasus"

		connConfig, err := pgx.ParseConfig(BuildDatabaseNameConninfo(databaseName))

		require.NoError(t, err)
		assert.Equal(t, databaseName, connConfig.Database)
	})
}

func TestWriteCredentialFilesToTempDir(t *testing.T) {
	t.Parallel()

	t.Run("escapes the pgpass separators when the password holds them", func(t *testing.T) {
		t.Parallel()
		spec := CredentialSpec{Host: "db.example.com", Port: 5432, Username: "backup:user", TLS: TLS{}}

		files, err := WriteCredentialFilesToTempDir(spec, "pass\\word:x\ny")
		require.NoError(t, err)
		t.Cleanup(files.Remove)

		assert.FileExists(t, files.PgpassPath)
		assert.Empty(t, files.ClientCertPath)
		pgpass := readFile(t, files.PgpassPath)
		assert.Equal(t, `db.example.com:5432:*:backup\:user:pass\\word\:xy`, pgpass)
	})

	t.Run("writes the certificates in clear text when SSL is enabled", func(t *testing.T) {
		t.Parallel()
		spec := CredentialSpec{
			Host:     "db.example.com",
			Port:     5432,
			Username: "backup",
			TLS:      TLS{Mode: SslModeVerifyFull, ClientCert: "cert", ClientKey: "key", RootCert: "root"},
		}

		files, err := WriteCredentialFilesToTempDir(spec, "password")
		require.NoError(t, err)
		t.Cleanup(files.Remove)

		assert.Equal(t, "cert", readFile(t, files.ClientCertPath))
		assert.Equal(t, "key", readFile(t, files.ClientKeyPath))
		assert.Equal(t, "root", readFile(t, files.RootCertPath))
	})
}
