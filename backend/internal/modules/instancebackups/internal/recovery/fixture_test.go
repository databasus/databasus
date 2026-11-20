package recovery

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
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
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/instancebackups/internal/snapshot"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/migrations"
)

const (
	testRelease           = "v9.9.9"
	sourceWorkspaceName   = "Recovered workspace"
	sourceAccountEmail    = "source-owner@databasus.local"
	newerMigrationVersion = 99991231235959
)

// sourceArchive is a Databasus backup file of a source installation and the source's key file.
type sourceArchive struct {
	path       string
	keyPath    string
	keyContent []byte
	// passwordVersion is the source account's password version, which every session carries.
	passwordVersion string
}

type archiveSpec struct {
	release       string
	bundleDigest  string
	workspaceName string
}

func newSourceArchive(t *testing.T) sourceArchive {
	t.Helper()

	return newSourceArchiveWith(
		t,
		archiveSpec{release: testRelease, bundleDigest: "", workspaceName: sourceWorkspaceName},
	)
}

// newSourceArchiveWith produces a real archive of a source database holding one account and one
// workspace. A non-empty bundleDigest stands for a build with other migrations.
func newSourceArchiveWith(t *testing.T, spec archiveSpec) sourceArchive {
	t.Helper()

	sourceDsn := testdb.Open(t)
	sourceDb := openTestDatabase(t, sourceDsn)
	require.NoError(
		t,
		sourceDb.Exec(`INSERT INTO users (email, password_creation_time, role, name, status, is_root_admin)
		VALUES (?, now(), 'ADMIN', 'Source owner', 'ACTIVE', true)`, sourceAccountEmail).Error,
	)
	workspaceID := uuid.New()
	require.NoError(t, sourceDb.Exec(`INSERT INTO workspaces (id, name, created_at) VALUES (?, ?, now())`,
		workspaceID, spec.workspaceName).Error)
	storageID := uuid.New()
	require.NoError(
		t,
		sourceDb.Exec(`INSERT INTO storages (id, workspace_id, name, type) VALUES (?, ?, 'Local', 'LOCAL')`,
			storageID, workspaceID).Error,
	)
	require.NoError(t, sourceDb.Exec(`INSERT INTO storages_local (storage_id) VALUES (?)`, storageID).Error)
	require.NoError(t, sourceDb.Exec(`INSERT INTO storages_pending_file_deletions (storage_id, file_name)
		VALUES (?, 'written-after-the-snapshot.dump')`, storageID).Error)

	var passwordVersions []string
	require.NoError(t, sourceDb.Raw("SELECT password_version::text FROM users").Scan(&passwordVersions).Error)
	require.Len(t, passwordVersions, 1)

	connection, err := internaldb.ParseConnection(sourceDsn)
	require.NoError(t, err)

	bundle := getTestBundle(t)
	if spec.bundleDigest != "" {
		bundle.Digest = spec.bundleDigest
	}

	keyContent := []byte(" source key " + uuid.NewString() + uuid.NewString() + "\n")
	masterKey, err := crypto.NewMasterKey(keyContent)
	require.NoError(t, err)

	spooled, err := snapshot.NewProducer(snapshot.ProducerDeps{
		Repositories: snapshot.NewRepositories(sourceDb),
		Connection:   connection,
		ClientTools:  getClientTools(t),
		MasterKey:    masterKey,
		Release:      spec.release,
		Bundle:       bundle,
		SpoolDir:     filepath.Join(t.TempDir(), "spool"),
		Clock:        clock.NewManual(time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)),
		Logger:       slog.New(slog.DiscardHandler),
		InspectDisk:  diskusage.Get,
	}).Produce(t.Context(), uuid.New())
	require.NoError(t, err)

	inputDirectory := t.TempDir()
	keyPath := filepath.Join(inputDirectory, "master.key")
	require.NoError(t, os.WriteFile(keyPath, keyContent, 0o600))

	return sourceArchive{
		path: spooled.Path, keyPath: keyPath, keyContent: keyContent, passwordVersion: passwordVersions[0],
	}
}

// target is the installation a startup recovery would replace: its database, data directory and
// key file.
type target struct {
	dsn           string
	server        internaldb.Connection
	dataDirectory string
	secretKeyPath string
	logs          *lockedBuffer
}

// newFreshTarget is a container's first start: an empty database and no key file yet.
func newFreshTarget(t *testing.T) target {
	t.Helper()

	return newTargetOn(t, testdb.OpenEmpty(t))
}

// newInitializedTarget is a target after an ordinary first start: migrated, with the default
// instance settings row created on the first read and a generated key file, logs and telemetry
// identity beside it.
func newInitializedTarget(t *testing.T) target {
	t.Helper()

	initializedTarget := newTargetOn(t, testdb.Open(t))
	initializedTarget.execute(t, `INSERT INTO users_settings (id, is_allow_external_registrations,
		is_allow_member_invitations, is_member_allowed_to_create_workspaces, is_two_factor_auth_required)
		VALUES (gen_random_uuid(), true, true, true, false)`)

	_, err := crypto.OpenOrCreateMasterKey(initializedTarget.secretKeyPath)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(initializedTarget.dataDirectory, "temp"), 0o700))
	require.NoError(t, os.MkdirAll(filepath.Join(initializedTarget.dataDirectory, "backups"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(initializedTarget.dataDirectory, "databasus.log"),
		[]byte("{}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(initializedTarget.dataDirectory, "instance.json"),
		[]byte(`{"id":"generated"}`), 0o600))

	return initializedTarget
}

func newTargetOn(t *testing.T, dsn string) target {
	t.Helper()

	server, err := internaldb.ParseConnection(dsn)
	require.NoError(t, err)

	dataDirectory := filepath.Join(t.TempDir(), "databasus-data")
	require.NoError(t, os.MkdirAll(dataDirectory, 0o700))

	return target{
		dsn:           dsn,
		server:        server,
		dataDirectory: dataDirectory,
		secretKeyPath: filepath.Join(dataDirectory, "secret.key"),
		logs:          &lockedBuffer{mutex: sync.Mutex{}, buffer: bytes.Buffer{}},
	}
}

type startupSpec struct {
	archive sourceArchive
	release string
}

// newStartup configures startup recovery with the archive's files, or none for a zero archive.
func (tg target) newStartup(t *testing.T, archive sourceArchive) *Startup {
	t.Helper()

	return tg.newStartupWith(t, startupSpec{archive: archive, release: testRelease})
}

func (tg target) newStartupWith(t *testing.T, spec startupSpec) *Startup {
	t.Helper()

	return NewStartup(StartupDeps{
		Directory:           tg.getDirectory(),
		Server:              tg.server,
		SecretKeyPath:       tg.secretKeyPath,
		DataFolder:          filepath.Join(tg.dataDirectory, "backups"),
		ClientTools:         getClientTools(t),
		Release:             spec.release,
		Bundle:              getTestBundle(t),
		DumpPath:            spec.archive.path,
		MasterKeyPath:       spec.archive.keyPath,
		NewerSchemaGuidance: "Start the image of the newer version instead",
		Normalizers:         getModuleNormalizers(),
		Clock:               clock.NewManual(time.Date(2026, 11, 14, 13, 0, 0, 0, time.UTC)),
		Logger:              slog.New(slog.NewJSONHandler(tg.logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		InspectDisk:         diskusage.Get,
	})
}

// openForServer runs a server start and then releases its lock, as a process that stops does, so
// the next start of the test is a restart.
func (tg target) openForServer(t *testing.T, startup *Startup) error {
	t.Helper()

	lock, err := startup.OpenForServer(t.Context())
	if err == nil {
		require.NoError(t, lock.Close())
	}

	return err
}

func (tg target) getDirectory() Directory {
	return NewDirectory(filepath.Join(tg.dataDirectory, "instance-recovery"))
}

func (tg target) readJournal(t *testing.T) Journal {
	t.Helper()

	journal, err := NewJournalStore(tg.getDirectory()).Read()
	require.NoError(t, err)

	return journal
}

func (tg target) readKey(t *testing.T) []byte {
	t.Helper()

	content, err := os.ReadFile(tg.secretKeyPath)
	if os.IsNotExist(err) {
		return nil
	}

	require.NoError(t, err)

	return content
}

func (tg target) execute(t *testing.T, statement string, values ...any) {
	t.Helper()

	withTestDatabase(t, tg.dsn, func(db *gorm.DB) {
		require.NoError(t, db.Exec(statement, values...).Error)
	})
}

func (tg target) listWorkspaceNames(t *testing.T) []string {
	t.Helper()

	names := []string{}

	withTestDatabase(t, tg.dsn, func(db *gorm.DB) {
		isPresent, err := database.HasTable(t.Context(), db, "public.workspaces")
		require.NoError(t, err)

		if isPresent {
			require.NoError(t, db.Raw("SELECT name FROM workspaces ORDER BY name").Scan(&names).Error)
		}
	})

	return names
}

func (tg target) getRecoveredByOperationID(t *testing.T) *uuid.UUID {
	t.Helper()

	var operationIDs []*uuid.UUID

	withTestDatabase(t, tg.dsn, func(db *gorm.DB) {
		require.NoError(t, db.Raw("SELECT recovered_by_operation_id FROM instancebackups_configs").
			Scan(&operationIDs).Error)
	})
	require.Len(t, operationIDs, 1)

	return operationIDs[0]
}

// listRecoveryDatabases returns the staging and replaced databases of this target left on the
// server.
func (tg target) listRecoveryDatabases(t *testing.T) []string {
	t.Helper()

	maintenance := tg.server.WithDatabase(maintenanceDatabase)
	scope := getDatabaseScope(tg.server.Database)

	stagingNames, err := internaldb.ListDatabases(t.Context(), maintenance, internaldb.StagingPrefix+scope)
	require.NoError(t, err)
	previousNames, err := internaldb.ListDatabases(t.Context(), maintenance, internaldb.PreviousPrefix+scope)
	require.NoError(t, err)

	return append(stagingNames, previousNames...)
}

func (tg target) getLogs() string {
	return tg.logs.String()
}

// lockedBuffer collects the startup logs of one target.
type lockedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return b.buffer.String()
}

// withTestDatabase closes its pool before returning, because publication cannot rename a database
// that still has sessions.
func withTestDatabase(t *testing.T, dsn string, use func(db *gorm.DB)) {
	t.Helper()

	db, err := database.Open(t.Context(), dsn)
	require.NoError(t, err)
	defer closeDatabase(db)

	use(db.WithContext(t.Context()))
}

func openTestDatabase(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	db, err := database.Open(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { closeDatabase(db) })

	return db.WithContext(t.Context())
}

func getTestBundle(t *testing.T) database.MigrationBundle {
	t.Helper()

	bundle, err := database.ReadMigrationBundle(migrations.Files)
	require.NoError(t, err)

	return bundle
}

func getClientTools(t *testing.T) pgshared.ClientTools {
	t.Helper()

	workingDir, err := os.Getwd()
	require.NoError(t, err)
	root, err := clienttools.FindBundleRoot(workingDir)
	require.NoError(t, err)

	return pgshared.ClientTools{Root: root}
}

// getModuleNormalizers are the normalizations the application wires in, so a test recovery leaves
// the state a real one does.
func getModuleNormalizers() []StateNormalizer {
	return []StateNormalizer{
		func(ctx context.Context, db *gorm.DB, _ time.Time) (int, error) {
			return 0, users.NormalizeRecoveredState(ctx, db)
		},
		func(ctx context.Context, db *gorm.DB, _ time.Time) (int, error) {
			return storages.NormalizeRecoveredState(ctx, db)
		},
		func(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
			return 0, backups.NormalizeRecoveredState(ctx, db, now)
		},
		func(ctx context.Context, db *gorm.DB, now time.Time) (int, error) {
			return 0, verification.NormalizeRecoveredState(ctx, db, now)
		},
	}
}

func (tg target) readPasswordVersion(t *testing.T) string {
	t.Helper()

	var passwordVersions []string

	withTestDatabase(t, tg.dsn, func(db *gorm.DB) {
		require.NoError(t, db.Raw("SELECT password_version::text FROM users WHERE email = ?", sourceAccountEmail).
			Scan(&passwordVersions).Error)
	})
	require.Len(t, passwordVersions, 1)

	return passwordVersions[0]
}
