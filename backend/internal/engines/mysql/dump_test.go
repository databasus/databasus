package mysql

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/toolstub"
)

const stubSourcePassword = `stub"secret\password`

// rejectZstdScript makes a stub fail every start that asks for zstd the way a client fails a
// compression the endpoint cannot negotiate.
const rejectZstdScript = `case "$*" in *--compression-algorithms=zstd*)
  echo 'mysqldump: Got error: 2066: Connection failed due to wrongly configured compression algorithm' >&2
  exit 2;;
esac`

func TestDump(t *testing.T) {
	t.Parallel()

	t.Run(
		"writes the dump into the output with the scope and the negotiated compression in the arguments",
		func(t *testing.T) {
			t.Parallel()
			clientTools, stubs := writeStubClientTools(t, Version80, toolstub.Spec{
				IsReadingStdin: false,
				Script:         `case "$*" in *--no-data*) exit 0;; esac; printf 'CREATE TABLE orders (id INT);'`,
			})
			var dumpedSQL bytes.Buffer

			err := newStubSource(clientTools, false).Dump(t.Context(), DumpSpec{
				Version:       Version80,
				ExcludeTables: []string{"audit_log", "audit_log"},
				Privileges:    []string{"SELECT", "TRIGGER", "EVENT"},
				Logger:        slog.New(slog.DiscardHandler),
			}, &dumpedSQL)

			require.NoError(t, err)
			assert.Equal(t, "CREATE TABLE orders (id INT);", dumpedSQL.String())
			dumpArgs := stubs.mysqldump.GetLastArgs(t)
			assert.Equal(t, []string{
				"--compression-algorithms=zstd", "--zstd-compression-level=5",
				"--host=db.example.com", "--port=3306", "--user=backup_user",
				"--single-transaction", "--routines", "--set-gtid-purged=OFF", "--quick", "--skip-add-locks",
				"--verbose", "--no-tablespaces",
				"--triggers", "--events",
				"--ignore-table=orders.audit_log",
				"--max-allowed-packet=1G", "--ssl-mode=DISABLED",
				"orders",
			}, dumpArgs[1:])
			assert.NotContains(t, dumpArgs, stubSourcePassword)
		},
	)

	t.Run("passes the credentials and the TLS mode through the defaults file", func(t *testing.T) {
		t.Parallel()
		copiedConfigPath := filepath.Join(t.TempDir(), "my.cnf")
		clientTools, _ := writeStubClientTools(t, Version80, toolstub.Spec{
			IsReadingStdin: false,
			Script:         `cp "${1#--defaults-file=}" '` + copiedConfigPath + `'`,
		})

		err := newStubSource(clientTools, true).Dump(t.Context(), newStubDumpSpec(Version80), &bytes.Buffer{})

		require.NoError(t, err)
		copiedConfig, err := os.ReadFile(copiedConfigPath)
		require.NoError(t, err)
		assert.Equal(t,
			"[client]\nuser=backup_user\npassword=\"stub\\\"secret\\\\password\"\nhost=db.example.com\nport=3306\n"+
				"ssl-mode=REQUIRED\n",
			string(copiedConfig),
		)
	})

	t.Run("skips triggers without the TRIGGER privilege", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version80, toolstub.Spec{})

		err := newStubSource(clientTools, false).Dump(t.Context(), newStubDumpSpec(Version80), &bytes.Buffer{})

		require.NoError(t, err)
		assert.Contains(t, stubs.mysqldump.GetLastArgs(t), "--skip-triggers")
		assert.NotContains(t, stubs.mysqldump.GetLastArgs(t), "--events")
	})

	t.Run("falls back to zlib when the server rejects zstd", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version80, toolstub.Spec{
			IsReadingStdin: false,
			Script:         rejectZstdScript,
		})

		err := newStubSource(clientTools, false).Dump(t.Context(), newStubDumpSpec(Version80), &bytes.Buffer{})

		require.NoError(t, err)
		assert.Equal(t, "--compression-algorithms=zlib", stubs.mysqldump.GetLastArgs(t)[1])
	})

	t.Run("offers only the legacy compress flag to MySQL 5.7", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version57, toolstub.Spec{})

		err := newStubSource(clientTools, false).Dump(t.Context(), newStubDumpSpec(Version57), &bytes.Buffer{})

		require.NoError(t, err)
		for _, invocation := range stubs.mysqldump.GetInvocations(t) {
			assert.NotContains(t, invocation[1], "--compression-algorithms")
		}
		assert.Equal(t, "--compress", stubs.mysqldump.GetLastArgs(t)[1])
	})

	t.Run("refuses a source without a database name", func(t *testing.T) {
		t.Parallel()
		clientTools, _ := writeStubClientTools(t, Version80, toolstub.Spec{})
		source := NewSource(Settings{Host: "db.example.com", Port: 3306, Username: "backup_user"}, clientTools)

		err := source.Dump(t.Context(), newStubDumpSpec(Version80), &bytes.Buffer{})

		require.ErrorIs(t, err, ErrDumpDatabaseNameRequired)
	})

	for _, testCase := range []struct {
		name            string
		script          string
		expectedMessage string
	}{
		{
			name:            "reports denied access when mysqldump says so",
			script:          `case "$*" in *--no-data*) exit 0;; esac; echo "mysqldump: Got error: 1045: Access denied for user" >&2; exit 2`,
			expectedMessage: "MySQL access denied. Check username and password.",
		},
		{
			name: "reports a rejected compression when the dump fails on it after the probe passed",
			script: `case "$*" in *--no-data*) exit 0;; esac
echo 'mysqldump: Got error: 2066: wrongly configured compression algorithm' >&2; exit 2`,
			expectedMessage: "MySQL rejected the network compression algorithm that had just passed the pre-flight handshake probe.",
		},
		{
			name:            "quotes the exit status and error output for any other exit code",
			script:          `case "$*" in *--no-data*) exit 0;; esac; echo 'mysqldump: out of memory' >&2; exit 5`,
			expectedMessage: "mysqldump failed: exit status 5 – stderr: mysqldump: out of memory",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			clientTools, _ := writeStubClientTools(t, Version80, toolstub.Spec{
				IsReadingStdin: false,
				Script:         testCase.script,
			})

			err := newStubSource(clientTools, false).Dump(t.Context(), newStubDumpSpec(Version80), &bytes.Buffer{})

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedMessage)
		})
	}
}

type stubClientTools struct {
	mysqldump   *toolstub.Stub
	mysqlClient *toolstub.Stub
}

// writeStubClientTools writes spec as both mysqldump and mysql, because the bundle of a version
// counts only when it holds both.
func writeStubClientTools(t *testing.T, version Version, spec toolstub.Spec) (ClientTools, stubClientTools) {
	t.Helper()

	clientTools := ClientTools{Root: t.TempDir()}
	binDir := clientTools.getBinDir(version)

	return clientTools, stubClientTools{
		mysqldump:   toolstub.Write(t, filepath.Join(binDir, string(ExecutableMysqldump)), spec),
		mysqlClient: toolstub.Write(t, filepath.Join(binDir, string(ExecutableMysql)), spec),
	}
}

func newStubSource(clientTools ClientTools, isHTTPS bool) *Source {
	return NewSource(Settings{
		Host:      "db.example.com",
		Port:      3306,
		Username:  "backup_user",
		Password:  stubSourcePassword,
		Database:  "orders",
		IsHTTPS:   isHTTPS,
		SSHTunnel: nil,
	}, clientTools)
}

func newStubDumpSpec(version Version) DumpSpec {
	return DumpSpec{
		Version:       version,
		ExcludeTables: nil,
		Privileges:    nil,
		Logger:        slog.New(slog.DiscardHandler),
	}
}
