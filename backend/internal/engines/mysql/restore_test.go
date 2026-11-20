package mysql

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/mysqlfamily/mysqlfamilytest"
	"databasus-backend/internal/lib/toolstub"
)

func TestRestore(t *testing.T) {
	t.Parallel()

	t.Run("feeds the SQL into the mysql client", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version80, toolstub.Spec{IsReadingStdin: true, Script: ""})

		err := newStubSource(clientTools, true).Restore(
			t.Context(), newStubRestoreSpec(), bytes.NewReader([]byte("INSERT INTO orders VALUES (1);")),
		)

		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO orders VALUES (1);", string(stubs.mysqlClient.GetStdin(t)))
		assert.Equal(t, []string{
			"--host=db.example.com", "--port=3306", "--user=backup_user",
			"--verbose", "--max-allowed-packet=1G", "--ssl-mode=REQUIRED",
			"--database=orders",
		}, stubs.mysqlClient.GetLastArgs(t)[1:])
	})

	t.Run("reports the client error written after the client exited", func(t *testing.T) {
		t.Parallel()
		clientTools, _ := writeStubClientTools(t, Version80, toolstub.Spec{
			IsReadingStdin: true,
			Script: "(sleep 0.2; echo 'ERROR 1064 (42000) at line 53465: You have an error in your SQL syntax' >&2) &\n" +
				"exit 1",
		})

		err := newStubSource(clientTools, false).Restore(
			t.Context(), newStubRestoreSpec(), bytes.NewReader([]byte("INSERT INTO orders VALUES (1);")),
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "SQL syntax")
	})

	t.Run("names the backed up database when the target database does not exist", func(t *testing.T) {
		t.Parallel()
		clientTools, _ := writeStubClientTools(t, Version80, toolstub.Spec{
			IsReadingStdin: true,
			Script:         "echo \"ERROR 1049 (42000): Unknown database 'orders'\" >&2; exit 1",
		})
		spec := newStubRestoreSpec()
		spec.SourceDatabaseName = "shop"

		err := newStubSource(clientTools, false).Restore(t.Context(), spec, bytes.NewReader(nil))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "target database does not exist (backup db shop).")
	})

	t.Run("keeps a dash-led database name out of the options with every shipped mysql client", func(t *testing.T) {
		t.Parallel()

		for _, version := range getSupportedVersions() {
			t.Run("version "+string(version), func(t *testing.T) {
				t.Parallel()
				clientTools := getClientTools(t)
				if clientTools.RequireBundle(version) != nil {
					t.Skip("this architecture ships no client for the version")
				}
				teePath := filepath.Join(t.TempDir(), "restore.log")
				source := newClosedPortSource(t, clientTools, "--tee="+teePath)
				mysqlClient, err := clientTools.GetExecutable(version, ExecutableMysql)
				require.NoError(t, err)

				toolOutput := mysqlfamilytest.RunClientTool(
					t, mysqlClient, append([]string{"--no-defaults"}, source.buildRestoreArgs()...),
				)

				assert.NoFileExists(t, teePath)
				assert.True(t, mysqlfamilytest.IsConnectionFailure(toolOutput), toolOutput)
			})
		}
	})
}

func newStubRestoreSpec() RestoreSpec {
	return RestoreSpec{
		Version:            Version80,
		SourceDatabaseName: "",
		Logger:             slog.New(slog.DiscardHandler),
	}
}
