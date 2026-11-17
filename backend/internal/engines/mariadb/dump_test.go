package mariadb

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/mysqlfamily"
	"databasus-backend/internal/engines/mysqlfamily/mysqlfamilytest"
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
				"--", "orders",
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
			"--compress",
			"--skip-ssl-verify-server-cert",
			"--no-data",
			"--",
			"orders",
			"__databasus_compression_probe__",
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

	t.Run("refuses a value the tools could read as an option before any tool starts", func(t *testing.T) {
		t.Parallel()

		for _, testCase := range []struct {
			name            string
			withValue       func(source *Source, resultPath string)
			expectedSetting string
		}{
			{
				name:            "dash-led database name",
				withValue:       func(source *Source, resultPath string) { source.settings.Database = "--result-file=" + resultPath },
				expectedSetting: "database",
			},
			{
				name:            "line break in the user name",
				withValue:       func(source *Source, resultPath string) { source.settings.Username = "x\nresult-file=" + resultPath },
				expectedSetting: "username",
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()
				resultPath := filepath.Join(t.TempDir(), "result.sql")
				source := newClosedPortSource(t, getClientTools(t), "orders")
				testCase.withValue(source, resultPath)

				err := source.Dump(t.Context(), newStubDumpSpec(), io.Discard)

				invalidSettingError, isInvalidSetting := errors.AsType[*mysqlfamily.InvalidSettingError](err)
				require.True(t, isInvalidSetting, err)
				assert.Equal(t, testCase.expectedSetting, invalidSettingError.Setting)
				assert.NoFileExists(t, resultPath)
			})
		}
	})

	t.Run("keeps a dash-led database name out of the options with every shipped mariadb-dump", func(t *testing.T) {
		t.Parallel()

		for _, serverVersion := range getServerVersionPerShippedClient(t) {
			t.Run("client "+string(GetClientVersionForServer(serverVersion)), func(t *testing.T) {
				t.Parallel()
				clientTools := getClientTools(t)
				resultPath := filepath.Join(t.TempDir(), "result.sql")
				source := newClosedPortSource(t, clientTools, "--result-file="+resultPath)
				mariadbDump, err := clientTools.GetExecutable(serverVersion, ExecutableMariadbDump)
				require.NoError(t, err)
				dumpSpec := newStubDumpSpec()
				dumpSpec.Version = serverVersion

				toolOutput := mysqlfamilytest.RunClientTool(
					t, mariadbDump, append([]string{"--no-defaults"}, source.buildDumpArgs(dumpSpec)...),
				)

				assert.NoFileExists(t, resultPath)
				assert.True(t, mysqlfamilytest.IsConnectionFailure(toolOutput), toolOutput)
			})
		}
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

// newClosedPortSource points at a local port where nothing listens, so a real client tool fails at
// the connection after it has read every argument.
func newClosedPortSource(t *testing.T, clientTools ClientTools, databaseName string) *Source {
	t.Helper()

	return NewSource(Settings{
		Host:      "127.0.0.1",
		Port:      mysqlfamilytest.GetClosedPort(t),
		Username:  "backup_user",
		Password:  "stub-secret-password",
		Database:  databaseName,
		IsHTTPS:   false,
		SSHTunnel: nil,
	}, clientTools)
}

// getServerVersionPerShippedClient picks one server version for each shipped client, and fails when
// a client is added without a server version here, so every shipped client stays covered.
func getServerVersionPerShippedClient(t *testing.T) []Version {
	t.Helper()

	serverVersions := []Version{Version101, Version1011, Version130}
	coveredClientVersions := make([]ClientVersion, 0, len(serverVersions))
	for _, serverVersion := range serverVersions {
		coveredClientVersions = append(coveredClientVersions, GetClientVersionForServer(serverVersion))
	}

	require.True(t, slices.Equal(getShippedClientVersions(), coveredClientVersions), coveredClientVersions)

	return serverVersions
}
