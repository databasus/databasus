package mongodb

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/toolstub"
)

func TestDump(t *testing.T) {
	t.Parallel()

	t.Run("writes the archive into the output with the dump scope in the arguments", func(t *testing.T) {
		t.Parallel()
		clientTools := ClientTools{Root: t.TempDir()}
		mongodump := toolstub.Write(t, clientTools.GetExecutable(ExecutableMongodump), toolstub.Spec{
			IsReadingStdin: false,
			Script:         "printf 'mongodump-archive'",
		})
		source := newStubSource("orders")
		var dumpedArchive bytes.Buffer

		err := source.Dump(t.Context(), DumpSpec{
			ClientTools:         clientTools,
			ExcludeCollections:  []string{"audit_log", "audit_log"},
			ParallelCollections: 32,
			Logger:              slog.New(slog.DiscardHandler),
		}, &dumpedArchive)

		require.NoError(t, err)
		assert.Equal(t, "mongodump-archive", dumpedArchive.String())
		assert.Equal(t, []string{
			"--uri=" + source.GetSettings().BuildConnectionURI(),
			"--archive", "--gzip",
			"--excludeCollection=audit_log",
			"--numParallelCollections=16",
		}, mongodump.GetLastArgs(t))
	})

	t.Run("logs the mongodump command without the password from the URI", func(t *testing.T) {
		t.Parallel()
		clientTools := ClientTools{Root: t.TempDir()}
		toolstub.Write(t, clientTools.GetExecutable(ExecutableMongodump), toolstub.Spec{
			IsReadingStdin: false,
			Script:         "",
		})
		var dumpLog bytes.Buffer

		err := newStubSource("orders").Dump(t.Context(), DumpSpec{
			ClientTools:         clientTools,
			ExcludeCollections:  nil,
			ParallelCollections: 1,
			Logger:              slog.New(slog.NewTextHandler(&dumpLog, nil)),
		}, &bytes.Buffer{})

		require.NoError(t, err)
		assert.Contains(t, dumpLog.String(), "executing MongoDB backup command")
		assert.NotContains(t, dumpLog.String(), stubSourcePassword)
	})

	t.Run("quotes the exit status and error output when mongodump fails", func(t *testing.T) {
		t.Parallel()
		clientTools := ClientTools{Root: t.TempDir()}
		toolstub.Write(t, clientTools.GetExecutable(ExecutableMongodump), toolstub.Spec{
			IsReadingStdin: false,
			Script:         "echo 'Failed: error connecting to db server' >&2; exit 1",
		})

		err := newStubSource("orders").Dump(t.Context(), DumpSpec{
			ClientTools:         clientTools,
			ExcludeCollections:  nil,
			ParallelCollections: 1,
			Logger:              slog.New(slog.DiscardHandler),
		}, &bytes.Buffer{})

		require.EqualError(t, err, "mongodump failed: exit status 1\nstderr: Failed: error connecting to db server\n")
	})
}

const stubSourcePassword = "stub-secret-password"

func newStubSource(databaseName string) *Source {
	return NewSource(Settings{
		Host:               "mongo.example.com",
		Port:               27017,
		Username:           "backup_user",
		Password:           stubSourcePassword,
		Database:           databaseName,
		AuthDatabase:       "admin",
		IsTLS:              false,
		IsSrv:              false,
		IsDirectConnection: false,
		SSHTunnel:          nil,
	})
}
