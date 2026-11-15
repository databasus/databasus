package snapshot

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/engines/clienttools"
	"databasus-backend/internal/engines/postgresql/internaldb"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/diskusage"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
	"databasus-backend/migrations"
)

func TestProduceAndImport(t *testing.T) {
	t.Parallel()

	t.Run("imports the snapshot with only the file and the source key, keeping exact key bytes", func(t *testing.T) {
		t.Parallel()
		source := newTestSource(t, diskusage.Get)
		require.NoError(t, source.db.Exec(
			`INSERT INTO workspaces (id, name, created_at) VALUES (?, 'Recovered workspace', now())`, uuid.New(),
		).Error)

		spooled, err := source.producer.Produce(t.Context(), uuid.New())
		require.NoError(t, err)
		info, err := os.Stat(spooled.Path)
		require.NoError(t, err)
		sourceKeyCopy, err := crypto.NewMasterKey([]byte(source.keyText))
		require.NoError(t, err)
		importer := NewImporter(ImporterDeps{
			Server:      source.connection,
			ClientTools: getClientTools(t),
			Logger:      slog.New(slog.DiscardHandler),
		})
		manifest, err := importer.Validate(spooled.Path, sourceKeyCopy)
		require.NoError(t, err)
		stagingName := newStagingName(t, source.connection)
		require.NoError(t, importer.Import(t.Context(), spooled.Path, sourceKeyCopy, stagingName))

		var workspaceNames []string
		stagingDb := openDatabase(t, source.connection.WithDatabase(stagingName))
		require.NoError(t, stagingDb.Raw("SELECT name FROM workspaces").Scan(&workspaceNames).Error)
		stagingVersions, err := database.ReadAppliedMigrationVersions(t.Context(), stagingDb)
		require.NoError(t, err)

		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		assert.Equal(t, info.Size(), spooled.SizeBytes)
		assert.Equal(t, "v9.9.9", manifest.SourceRelease)
		assert.Equal(t, source.bundle.Digest, manifest.SchemaIdentity.MigrationBundleDigest)
		require.NoError(t, source.bundle.CheckCompatibility(manifest.SchemaIdentity))
		assert.Equal(t, manifest.SchemaIdentity.AppliedMigrationVersions, stagingVersions)
		assert.Equal(t, []string{"Recovered workspace"}, workspaceNames)
	})

	t.Run("refuses a key that differs only by a trailing line ending", func(t *testing.T) {
		t.Parallel()
		source := newTestSource(t, diskusage.Get)
		spooled, err := source.producer.Produce(t.Context(), uuid.New())
		require.NoError(t, err)
		changedKey, err := crypto.NewMasterKey([]byte(source.keyText + "\n"))
		require.NoError(t, err)

		_, err = NewImporter(ImporterDeps{
			Server: source.connection, ClientTools: getClientTools(t), Logger: slog.New(slog.DiscardHandler),
		}).Validate(spooled.Path, changedKey)

		require.ErrorIs(t, err, archive.ErrMasterKeyMismatch)
	})

	t.Run("writes nothing when the disk has too little free space", func(t *testing.T) {
		t.Parallel()
		source := newTestSource(t, func(string) (diskusage.Usage, error) {
			return diskusage.Usage{TotalBytes: 1 << 30, UsedBytes: 1 << 30, FreeBytes: 0}, nil
		})

		_, err := source.producer.Produce(t.Context(), uuid.New())

		require.ErrorIs(t, err, ErrInsufficientDiskSpace)
		entries, err := os.ReadDir(source.spoolDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("removes the partial spool file when the dump fails", func(t *testing.T) {
		t.Parallel()
		source := newTestSource(t, diskusage.Get)
		source.producer.connection.Password = "wrong-password"

		_, err := source.producer.Produce(t.Context(), uuid.New())

		require.ErrorIs(t, err, internaldb.ErrDumpFailed)
		entries, err := os.ReadDir(source.spoolDir)
		require.NoError(t, err)
		assert.Empty(t, entries)
	})
}

type testSource struct {
	db         *gorm.DB
	connection internaldb.Connection
	producer   *Producer
	bundle     database.MigrationBundle
	keyText    string
	spoolDir   string
}

func newTestSource(t *testing.T, inspectDisk func(string) (diskusage.Usage, error)) testSource {
	t.Helper()

	dsn := testdb.Open(t)
	db, err := database.Open(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	connection, err := internaldb.ParseConnection(dsn)
	require.NoError(t, err)
	bundle, err := database.ReadMigrationBundle(migrations.Files)
	require.NoError(t, err)

	keyText := " key with spaces " + uuid.NewString() + uuid.NewString()
	masterKey, err := crypto.NewMasterKey([]byte(keyText))
	require.NoError(t, err)

	spoolDir := filepath.Join(t.TempDir(), "spool")

	return testSource{
		db:         db,
		connection: connection,
		bundle:     bundle,
		keyText:    keyText,
		spoolDir:   spoolDir,
		producer: NewProducer(ProducerDeps{
			Repositories: NewRepositories(db),
			Connection:   connection,
			ClientTools:  getClientTools(t),
			MasterKey:    masterKey,
			Release:      "v9.9.9",
			Bundle:       bundle,
			SpoolDir:     spoolDir,
			Clock:        clock.NewManual(time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)),
			Logger:       slog.New(slog.DiscardHandler),
			InspectDisk:  inspectDisk,
		}),
	}
}

func newStagingName(t *testing.T, server internaldb.Connection) string {
	t.Helper()

	name := internaldb.StagingPrefix + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	t.Cleanup(func() { _ = internaldb.DropStagingDatabase(context.WithoutCancel(t.Context()), server, name) })

	return name
}

func openDatabase(t *testing.T, connection internaldb.Connection) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), connection.ConnString())
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func getClientTools(t *testing.T) pgshared.ClientTools {
	t.Helper()

	workingDir, err := os.Getwd()
	require.NoError(t, err)
	root, err := clienttools.FindBundleRoot(workingDir)
	require.NoError(t, err)

	return pgshared.ClientTools{Root: root}
}
