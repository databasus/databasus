package mariadb

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/toolstub"
)

func TestRestore(t *testing.T) {
	t.Parallel()

	t.Run("feeds the SQL into the mariadb client with Galera replication off", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version1011, toolstub.Spec{IsReadingStdin: true, Script: ""})

		err := newStubSource(clientTools, false).Restore(
			t.Context(), newStubRestoreSpec(), bytes.NewReader([]byte("INSERT INTO orders VALUES (1);")),
		)

		require.NoError(t, err)
		assert.Equal(t, "INSERT INTO orders VALUES (1);", string(stubs.mariadbClient.GetStdin(t)))
		assert.Equal(t, []string{
			"--host=db.example.com", "--port=3306", "--user=backup_user",
			"--verbose", "--init-command=SET SESSION wsrep_on=OFF",
			"--max-allowed-packet=1G", "--skip-ssl",
			"orders",
		}, stubs.mariadbClient.GetLastArgs(t)[1:])
	})

	t.Run("keeps Galera replication on when its disabling is skipped", func(t *testing.T) {
		t.Parallel()
		clientTools, stubs := writeStubClientTools(t, Version1011, toolstub.Spec{IsReadingStdin: true, Script: ""})
		spec := newStubRestoreSpec()
		spec.IsSkipGaleraDisable = true

		err := newStubSource(clientTools, false).Restore(t.Context(), spec, bytes.NewReader(nil))

		require.NoError(t, err)
		assert.NotContains(t, stubs.mariadbClient.GetLastArgs(t), "--init-command=SET SESSION wsrep_on=OFF")
	})

	t.Run("reports the Galera writeset limit", func(t *testing.T) {
		t.Parallel()
		clientTools, _ := writeStubClientTools(t, Version1011, toolstub.Spec{
			IsReadingStdin: true,
			Script:         "echo 'ERROR 1180 (HY000): Maximum writeset size exceeded' >&2; exit 1",
		})

		err := newStubSource(clientTools, false).Restore(t.Context(), newStubRestoreSpec(), bytes.NewReader(nil))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "MariaDB Galera Cluster writeset size limit exceeded.")
	})
}

func newStubRestoreSpec() RestoreSpec {
	return RestoreSpec{
		Version:             Version1011,
		SourceDatabaseName:  "",
		IsSkipGaleraDisable: false,
		Logger:              slog.New(slog.DiscardHandler),
	}
}
