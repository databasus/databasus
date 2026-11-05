package logical

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/toolstub"
)

const stubSourcePassword = "stub-secret-password"

func TestDump(t *testing.T) {
	t.Parallel()

	t.Run("writes the pg_dump archive into the output with the dump scope in the arguments", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		pgDump := writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgDump, toolstub.Spec{
			IsReadingStdin: false,
			Script:         "printf 'PGDMP-archive'",
		})
		var dumpedArchive bytes.Buffer

		err := newStubSource("orders").Dump(t.Context(), DumpSpec{
			ClientTools:      clientTools,
			Version:          pgshared.Version16,
			IncludeSchemas:   []string{"sales", "public", "sales"},
			ExcludeTables:    []string{"public.audit_log"},
			ExcludeTableData: []string{"public.events"},
			Logger:           slog.New(slog.DiscardHandler),
		}, &dumpedArchive)

		require.NoError(t, err)
		assert.Equal(t, "PGDMP-archive", dumpedArchive.String())
		assert.Equal(t, []string{
			"-Fc", "--no-password",
			"-h", "db.example.com", "-p", "5432", "-U", "backup_user",
			"-d", "dbname='orders'",
			"--verbose",
			"-n", "sales", "-n", "public",
			"--exclude-table=public.audit_log",
			"--exclude-table-data=public.events",
			"--compress=zstd:5",
		}, pgDump.GetLastArgs(t))
		assert.NotContains(t, pgDump.GetLastArgs(t), stubSourcePassword)
	})

	t.Run("compresses with gzip before PostgreSQL 16", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		pgDump := writeStubTool(t, clientTools, pgshared.Version15, pgshared.ExecutablePgDump, toolstub.Spec{})

		err := newStubSource(
			"orders",
		).Dump(t.Context(), newStubDumpSpec(clientTools, pgshared.Version15), &bytes.Buffer{})

		require.NoError(t, err)
		assert.Equal(t, []string{"-Z", "5"}, pgDump.GetLastArgs(t)[len(pgDump.GetLastArgs(t))-2:])
	})

	t.Run("points at the password file when the server asks for a password pg_dump lacks", func(t *testing.T) {
		t.Parallel()
		clientTools := pgshared.ClientTools{Root: t.TempDir()}
		writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgDump, toolstub.Spec{
			IsReadingStdin: false,
			Script:         `echo 'pg_dump: error: fe_sendauth: no password supplied' >&2; exit 1`,
		})

		err := newStubSource(
			"orders",
		).Dump(t.Context(), newStubDumpSpec(clientTools, pgshared.Version16), &bytes.Buffer{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "the password file given to pg_dump held none for this connection")
		assert.NotContains(t, err.Error(), "PGPASSWORD")
	})

	t.Run("refuses a source without a database name", func(t *testing.T) {
		t.Parallel()

		err := newStubSource("").Dump(
			t.Context(), newStubDumpSpec(pgshared.ClientTools{Root: t.TempDir()}, pgshared.Version16), &bytes.Buffer{},
		)

		require.ErrorIs(t, err, ErrDumpDatabaseNameRequired)
	})

	for _, testCase := range []struct {
		name            string
		script          string
		expectedMessage string
	}{
		{
			name:            "reports a rejected connection when pg_dump names pg_hba.conf",
			script:          `echo 'FATAL:  no pg_hba.conf entry for host "10.0.0.5"' >&2; exit 1`,
			expectedMessage: "PostgreSQL connection rejected by server configuration (pg_hba.conf).",
		},
		{
			name:            "reports a failed authentication when pg_dump names the password",
			script:          `echo 'FATAL:  password authentication failed for user "backup_user"' >&2; exit 1`,
			expectedMessage: "PostgreSQL authentication failed. Check username and password.",
		},
		{
			name:            "reports missing details when pg_dump exits with 1 and no error output",
			script:          "exit 1",
			expectedMessage: "pg_dump failed with exit status 1 but provided no error details.",
		},
		{
			name:            "quotes the exit status and error output for any other exit code",
			script:          "echo 'pg_dump: error: out of memory' >&2; exit 3",
			expectedMessage: "pg_dump failed: exit status 3 – stderr: pg_dump: error: out of memory",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			clientTools := pgshared.ClientTools{Root: t.TempDir()}
			writeStubTool(t, clientTools, pgshared.Version16, pgshared.ExecutablePgDump, toolstub.Spec{
				IsReadingStdin: false,
				Script:         testCase.script,
			})

			err := newStubSource(
				"orders",
			).Dump(t.Context(), newStubDumpSpec(clientTools, pgshared.Version16), &bytes.Buffer{})

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedMessage)
		})
	}
}

func newStubSource(databaseName string) *Source {
	return NewSource(Settings{
		Host:      "db.example.com",
		HostAddr:  "",
		Port:      5432,
		Username:  "backup_user",
		Password:  stubSourcePassword,
		Database:  databaseName,
		TLS:       pgshared.TLS{},
		SSHTunnel: nil,
	})
}

func newStubDumpSpec(clientTools pgshared.ClientTools, version pgshared.Version) DumpSpec {
	return DumpSpec{
		ClientTools:      clientTools,
		Version:          version,
		IncludeSchemas:   nil,
		ExcludeTables:    nil,
		ExcludeTableData: nil,
		Logger:           slog.New(slog.DiscardHandler),
	}
}

func writeStubTool(
	t *testing.T,
	clientTools pgshared.ClientTools,
	version pgshared.Version,
	executable pgshared.Executable,
	spec toolstub.Spec,
) *toolstub.Stub {
	t.Helper()

	return toolstub.Write(t, clientTools.GetExecutable(version, executable), spec)
}
