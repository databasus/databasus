package mariadb

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/toolstub"
)

func TestDump(t *testing.T) {
	t.Parallel()

	t.Run(
		"writes the dump into the output with the scope and the negotiated compression in the arguments",
		func(t *testing.T) {
			t.Parallel()
			clientTools, stubs := writeStubClientTools(t, Version1011, toolstub.Spec{
				IsReadingStdin: false,
				Script:         `case "$*" in *--no-data*) exit 0;; esac; printf 'CREATE TABLE orders (id INT);'`,
			})
			var dumpedSQL bytes.Buffer

			err := newStubSource(clientTools, true).Dump(t.Context(), DumpSpec{
				Version:         Version1011,
				ExcludeTables:   []string{"audit_log"},
				Privileges:      []string{"SELECT", "TRIGGER", "EVENT"},
				IsExcludeEvents: false,
				Logger:          slog.New(slog.DiscardHandler),
			}, &dumpedSQL)

			require.NoError(t, err)
			assert.Equal(t, "CREATE TABLE orders (id INT);", dumpedSQL.String())
			assert.Equal(t, []string{
				"--compress",
				"--host=db.example.com", "--port=3306", "--user=backup_user",
				"--single-transaction", "--routines", "--quick", "--skip-add-locks",
				"--verbose", "--no-tablespaces",
				"--triggers", "--events",
				"--ignore-table=orders.audit_log",
				"--max-allowed-packet=1G", "--ssl", "--skip-ssl-verify-server-cert",
				"orders",
			}, stubs.mariadbDump.GetLastArgs(t)[1:])
		},
	)

	t.Run("probes with the certificate opt-out over TLS", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version1011, toolstub.Spec{})

		err := newStubSource(clientTools, true).Dump(t.Context(), newStubDumpSpec(), &bytes.Buffer{})

		require.NoError(t, err)
		probeArgs := stubs.mariadbDump.GetInvocations(t)[0]
		assert.Equal(t, []string{
			"--compress", "--skip-ssl-verify-server-cert", "--no-data", "orders", "__databasus_compression_probe__",
		}, probeArgs[1:])
	})

	t.Run("leaves events out when they are excluded", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version1011, toolstub.Spec{})
		spec := newStubDumpSpec()
		spec.Privileges = []string{"EVENT"}
		spec.IsExcludeEvents = true

		err := newStubSource(clientTools, false).Dump(t.Context(), spec, &bytes.Buffer{})

		require.NoError(t, err)
		assert.NotContains(t, stubs.mariadbDump.GetLastArgs(t), "--events")
		assert.Contains(t, stubs.mariadbDump.GetLastArgs(t), "--skip-ssl")
	})

	t.Run("reports denied access when mariadb-dump says so", func(t *testing.T) {
		t.Parallel()
		clientTools, _ := writeStubClientTools(t, Version1011, toolstub.Spec{
			IsReadingStdin: false,
			Script:         `case "$*" in *--no-data*) exit 0;; esac; echo 'Got error: 1045: Access denied' >&2; exit 2`,
		})

		err := newStubSource(clientTools, false).Dump(t.Context(), newStubDumpSpec(), &bytes.Buffer{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MariaDB access denied. Check username and password.")
	})
}

type stubClientTools struct {
	mariadbDump   *toolstub.Stub
	mariadbClient *toolstub.Stub
}

// writeStubClientTools writes spec as both mariadb-dump and mariadb, because the bundle of a
// version counts only when it holds both.
func writeStubClientTools(t *testing.T, version Version, spec toolstub.Spec) (ClientTools, stubClientTools) {
	t.Helper()

	clientTools := ClientTools{Root: t.TempDir()}
	binDir := clientTools.getBinDir(GetClientVersionForServer(version))

	return clientTools, stubClientTools{
		mariadbDump:   toolstub.Write(t, filepath.Join(binDir, string(ExecutableMariadbDump)), spec),
		mariadbClient: toolstub.Write(t, filepath.Join(binDir, string(ExecutableMariadb)), spec),
	}
}

func newStubSource(clientTools ClientTools, isHTTPS bool) *Source {
	return NewSource(Settings{
		Host:      "db.example.com",
		Port:      3306,
		Username:  "backup_user",
		Password:  "stub-secret-password",
		Database:  "orders",
		IsHTTPS:   isHTTPS,
		SSHTunnel: nil,
	}, clientTools)
}

func newStubDumpSpec() DumpSpec {
	return DumpSpec{
		Version:         Version1011,
		ExcludeTables:   nil,
		Privileges:      nil,
		IsExcludeEvents: false,
		Logger:          slog.New(slog.DiscardHandler),
	}
}
