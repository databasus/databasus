package mongodb

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

	t.Run("feeds the archive into mongorestore and renames the source database", func(t *testing.T) {
		t.Parallel()
		clientTools := ClientTools{Root: t.TempDir()}
		mongorestore := toolstub.Write(t, clientTools.GetExecutable(ExecutableMongorestore), toolstub.Spec{
			IsReadingStdin: true,
			Script:         "",
		})
		source := newStubSource("orders_restored")

		err := source.Restore(t.Context(), RestoreSpec{
			ClientTools:        clientTools,
			SourceDatabaseName: "orders",
			InsertionWorkers:   4,
			Logger:             slog.New(slog.DiscardHandler),
		}, bytes.NewReader([]byte("mongodump-archive")))

		require.NoError(t, err)
		assert.Equal(t, "mongodump-archive", string(mongorestore.GetStdin(t)))
		assert.Equal(t, []string{
			"--uri=" + source.GetSettings().BuildRestoreURI(),
			"--archive", "--gzip", "--drop",
			"--nsFrom=orders.*", "--nsTo=orders_restored.*",
			"--numInsertionWorkersPerCollection=4",
		}, mongorestore.GetLastArgs(t))
	})

	t.Run("logs the mongorestore command without the password from the URI", func(t *testing.T) {
		t.Parallel()
		clientTools := ClientTools{Root: t.TempDir()}
		toolstub.Write(t, clientTools.GetExecutable(ExecutableMongorestore), toolstub.Spec{
			IsReadingStdin: true,
			Script:         "",
		})
		var restoreLog bytes.Buffer

		err := newStubSource("orders").Restore(t.Context(), RestoreSpec{
			ClientTools:        clientTools,
			SourceDatabaseName: "orders",
			InsertionWorkers:   1,
			Logger:             slog.New(slog.NewTextHandler(&restoreLog, nil)),
		}, bytes.NewReader([]byte("mongodump-archive")))

		require.NoError(t, err)
		assert.Contains(t, restoreLog.String(), "executing MongoDB restore command")
		assert.NotContains(t, restoreLog.String(), stubSourcePassword)
	})

	t.Run("includes only the target database when the names match", func(t *testing.T) {
		t.Parallel()
		clientTools := ClientTools{Root: t.TempDir()}
		mongorestore := toolstub.Write(t, clientTools.GetExecutable(ExecutableMongorestore), toolstub.Spec{
			IsReadingStdin: true,
			Script:         "",
		})

		err := newStubSource("orders").Restore(t.Context(), RestoreSpec{
			ClientTools:        clientTools,
			SourceDatabaseName: "orders",
			InsertionWorkers:   1,
			Logger:             slog.New(slog.DiscardHandler),
		}, bytes.NewReader(nil))

		require.NoError(t, err)
		assert.Equal(t, []string{"--nsInclude=orders.*"}, mongorestore.GetLastArgs(t)[4:])
	})

	for _, testCase := range []struct {
		name            string
		script          string
		expectedMessage string
	}{
		{
			name:            "reports a failed authentication when mongorestore says so",
			script:          "echo 'error connecting to host: authentication failed' >&2; exit 1",
			expectedMessage: "MongoDB authentication failed. Check username and password.",
		},
		{
			name: "reports the mongorestore error written after mongorestore exited",
			script: "(sleep 0.2; echo 'Failed: restore error: orders: error restoring from archive: unexpected EOF' >&2) &\n" +
				"exit 1",
			expectedMessage: "error restoring from archive",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			clientTools := ClientTools{Root: t.TempDir()}
			toolstub.Write(t, clientTools.GetExecutable(ExecutableMongorestore), toolstub.Spec{
				IsReadingStdin: true,
				Script:         testCase.script,
			})

			err := newStubSource("orders").Restore(t.Context(), RestoreSpec{
				ClientTools:        clientTools,
				SourceDatabaseName: "orders",
				InsertionWorkers:   1,
				Logger:             slog.New(slog.DiscardHandler),
			}, bytes.NewReader([]byte("archive")))

			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.expectedMessage)
		})
	}
}
