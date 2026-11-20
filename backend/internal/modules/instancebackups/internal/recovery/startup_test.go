package recovery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/instancebackups/internal/archive"
)

func TestStartupRecovery(t *testing.T) {
	t.Parallel()

	t.Run("recovers a fresh target and installs the exact source key", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		freshTarget := newFreshTarget(t)

		err := freshTarget.openForServer(t, freshTarget.newStartup(t, source))

		require.NoError(t, err)
		journal := freshTarget.readJournal(t)
		require.NotNil(t, journal.LastCompleted)
		archiveSha256, err := hashFile(source.path)
		require.NoError(t, err)
		assert.Equal(t, []string{sourceWorkspaceName}, freshTarget.listWorkspaceNames(t))
		assert.Equal(t, source.keyContent, freshTarget.readKey(t))
		assert.Equal(t, &journal.LastCompleted.OperationID, freshTarget.getRecoveredByOperationID(t))
		assert.Equal(t, []string{archiveSha256}, journal.ConsumedStartupArchives)
		assert.Equal(t, OriginStartup, journal.LastCompleted.Origin)
		assert.Equal(t, testRelease, journal.LastCompleted.SourceRelease)
		assert.Nil(t, journal.Operation)
		assert.Empty(t, freshTarget.listRecoveryDatabases(t))
		assertNoOperationFiles(t, freshTarget)
	})

	t.Run("ends the lost instance's sessions and file work", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		freshTarget := newFreshTarget(t)

		err := freshTarget.openForServer(t, freshTarget.newStartup(t, source))

		require.NoError(t, err)
		passwordVersion := freshTarget.readPasswordVersion(t)
		var pendingDeletionCount int64
		withTestDatabase(t, freshTarget.dsn, func(db *gorm.DB) {
			require.NoError(t, db.Raw("SELECT count(*) FROM storages_pending_file_deletions").
				Scan(&pendingDeletionCount).Error)
		})
		assert.NotEqual(t, source.passwordVersion, passwordVersion)
		assert.Zero(t, pendingDeletionCount)
		assert.Equal(t, 1, freshTarget.readJournal(t).LastCompleted.DiscardedCleanupCount)
	})

	t.Run("replaces the generated key of an automatically initialized target", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		generatedKey := initializedTarget.readKey(t)

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, source))

		require.NoError(t, err)
		assert.NotEqual(t, generatedKey, source.keyContent)
		assert.Equal(t, source.keyContent, initializedTarget.readKey(t))
		assert.Equal(t, []string{sourceWorkspaceName}, initializedTarget.listWorkspaceNames(t))
		assert.Empty(t, initializedTarget.listRecoveryDatabases(t))
		info, err := os.Stat(initializedTarget.getDirectory().root)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
	})

	t.Run("starts without recovery and without a recovery log when none is configured", func(t *testing.T) {
		t.Parallel()
		initializedTarget := newInitializedTarget(t)

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, sourceArchive{}))

		require.NoError(t, err)
		assert.NotContains(t, initializedTarget.getLogs(), "startup recovery")
	})
}

func TestStartupRecoveryRefusesUserState(t *testing.T) {
	t.Parallel()

	for _, refusalCase := range []struct {
		name           string
		createState    func(t *testing.T, tg target)
		expectedReason RefusalReason
		expectedDetail string
	}{
		{
			name: "refuses a target whose only account is the root account",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				tg.execute(t, `INSERT INTO users (email, password_creation_time, role, name, status, is_root_admin)
					VALUES ('root@target.local', now(), 'ADMIN', 'Root', 'ACTIVE', true)`)
			},
			expectedReason: RefusalExistingAccounts,
			expectedDetail: "user accounts",
		},
		{
			name: "refuses a target without accounts whose instance settings were changed",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				tg.execute(t, `UPDATE users_settings SET is_allow_external_registrations = false`)
			},
			expectedReason: RefusalChangedSettings,
			expectedDetail: "settings",
		},
		{
			name: "refuses a target without accounts whose Databasus backup settings were changed",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				tg.execute(t, `UPDATE instancebackups_configs SET retention_days = 7`)
			},
			expectedReason: RefusalChangedSettings,
			expectedDetail: "settings",
		},
		{
			name: "refuses a target without accounts that holds other user data",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				tg.execute(t, `INSERT INTO workspaces (id, name, created_at) VALUES (gen_random_uuid(), 'Kept', now())`)
			},
			expectedReason: RefusalUserData,
			expectedDetail: "public.workspaces",
		},
		{
			name: "refuses a target whose local backup folder holds files",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				require.NoError(t, os.WriteFile(filepath.Join(tg.dataDirectory, "backups", "kept.dump"),
					[]byte("kept"), 0o600))
			},
			expectedReason: RefusalUserData,
			expectedDetail: "local backup folder",
		},
		{
			name: "refuses a target holding rows outside the application schema",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				tg.execute(t, `CREATE SCHEMA operator_notes`)
				tg.execute(t, `CREATE TABLE operator_notes.notes (text TEXT)`)
				tg.execute(t, `INSERT INTO operator_notes.notes VALUES ('kept')`)
			},
			expectedReason: RefusalUnknownState,
			expectedDetail: "could not be verified",
		},
		{
			name: "refuses a target migrated by a newer version and keeps the newer-schema guidance",
			createState: func(t *testing.T, tg target) {
				t.Helper()
				tg.execute(t, `INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, true)`,
					newerMigrationVersion)
			},
			expectedReason: RefusalNewerSchema,
			expectedDetail: "Start the image of the newer version instead",
		},
	} {
		t.Run(refusalCase.name, func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			initializedTarget := newInitializedTarget(t)
			refusalCase.createState(t, initializedTarget)
			generatedKey := initializedTarget.readKey(t)

			err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, source))

			refusal := requireRefusal(t, err, refusalCase.expectedReason)
			assertTargetUnchanged(t, initializedTarget, generatedKey)
			logs := initializedTarget.getLogs()
			assert.Contains(t, logs, "startup recovery did not run")
			assert.Contains(t, logs, `"reason":"`+string(refusalCase.expectedReason)+`"`)
			assert.Contains(t, logs, `"current_installation":"unchanged"`)
			assert.Contains(t, refusal.Explanation+refusal.NextStep+logs, refusalCase.expectedDetail)
			assert.NotContains(t, logs, string(source.keyContent))
			if refusalCase.expectedReason != RefusalNewerSchema {
				assert.Contains(t, refusal.NextStep, "root-confirmed import")
				assert.Contains(t, refusal.NextStep, "separate fresh installation")
			}
		})
	}
}

func TestStartupRecoveryRefusesUnusableInputs(t *testing.T) {
	t.Parallel()

	for _, refusalCase := range []struct {
		name           string
		changeInputs   func(t *testing.T, source sourceArchive) startupSpec
		expectedReason RefusalReason
		expectedError  error
	}{
		{
			name: "refuses a missing Databasus backup file",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()
				source.path = filepath.Join(t.TempDir(), "missing.dbbackup")

				return startupSpec{archive: source, release: testRelease}
			},
			expectedReason: RefusalInputUnreadable,
			expectedError:  os.ErrNotExist,
		},
		{
			name: "refuses a missing master key file",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()
				source.keyPath = filepath.Join(t.TempDir(), "missing.key")

				return startupSpec{archive: source, release: testRelease}
			},
			expectedReason: RefusalInputUnreadable,
			expectedError:  os.ErrNotExist,
		},
		{
			name: "refuses a master key shorter than any valid key",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()
				source.keyPath = writeInput(t, "short.key", []byte("short"))

				return startupSpec{archive: source, release: testRelease}
			},
			expectedReason: RefusalInvalidMasterKey,
			expectedError:  crypto.ErrMasterKeyTooShort,
		},
		{
			name: "refuses another installation's master key",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()
				source.keyPath = writeInput(t, "other.key", []byte(uuid.NewString()+uuid.NewString()))

				return startupSpec{archive: source, release: testRelease}
			},
			expectedReason: RefusalInvalidMasterKey,
			expectedError:  archive.ErrMasterKeyMismatch,
		},
		{
			name: "refuses a truncated Databasus backup file",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()
				content, err := os.ReadFile(source.path)
				require.NoError(t, err)
				source.path = writeInput(t, "truncated.dbbackup", content[:len(content)-10])

				return startupSpec{archive: source, release: testRelease}
			},
			expectedReason: RefusalInvalidArchive,
			expectedError:  archive.ErrArchiveIncomplete,
		},
		{
			name: "refuses a file of another format",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()
				source.path = writeInput(t, "plain.dump", []byte("PGDMP not a Databasus backup"))

				return startupSpec{archive: source, release: testRelease}
			},
			expectedReason: RefusalInvalidArchive,
			expectedError:  archive.ErrNotDatabasusBackup,
		},
		{
			name: "refuses a Databasus backup of another patch release",
			changeInputs: func(t *testing.T, source sourceArchive) startupSpec {
				t.Helper()

				return startupSpec{archive: source, release: "v9.9.10"}
			},
			expectedReason: RefusalIncompatibleRelease,
			expectedError:  nil,
		},
	} {
		t.Run(refusalCase.name, func(t *testing.T) {
			t.Parallel()
			source := newSourceArchive(t)
			initializedTarget := newInitializedTarget(t)
			generatedKey := initializedTarget.readKey(t)

			err := initializedTarget.openForServer(t,
				initializedTarget.newStartupWith(t, refusalCase.changeInputs(t, source)))

			_ = requireRefusal(t, err, refusalCase.expectedReason)
			if refusalCase.expectedError != nil {
				require.ErrorIs(t, err, refusalCase.expectedError)
			}
			assertTargetUnchanged(t, initializedTarget, generatedKey)
			assert.Empty(t, initializedTarget.readJournal(t).ConsumedStartupArchives)
			assert.Contains(t, initializedTarget.getLogs(), `"current_installation":"unchanged"`)
			assert.NotContains(t, initializedTarget.getLogs(), string(source.keyContent))
		})
	}

	t.Run("names both releases when the backup comes from another release", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchiveWith(t, archiveSpec{release: "v9.9.8", bundleDigest: "", workspaceName: "w"})
		initializedTarget := newInitializedTarget(t)

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, source))

		refusal := requireRefusal(t, err, RefusalIncompatibleRelease)
		assert.Contains(t, refusal.Explanation, "v9.9.8")
		assert.Contains(t, refusal.Explanation, testRelease)
		assert.Contains(t, refusal.NextStep, "Start Databasus v9.9.8")
	})

	t.Run("refuses a build of the same release label with other migrations", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchiveWith(t, archiveSpec{release: testRelease, bundleDigest: "other", workspaceName: "w"})
		freshTarget := newFreshTarget(t)

		err := freshTarget.openForServer(t, freshTarget.newStartup(t, source))

		_ = requireRefusal(t, err, RefusalIncompatibleSchema)
		assertTargetUnchanged(t, freshTarget, nil)
	})

	t.Run("recovers after a failed attempt, which was not marked consumed", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		wrongKeySource := source
		wrongKeySource.keyPath = writeInput(t, "other.key", []byte(uuid.NewString()+uuid.NewString()))
		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, wrongKeySource))
		_ = requireRefusal(t, err, RefusalInvalidMasterKey)

		err = initializedTarget.openForServer(t, initializedTarget.newStartup(t, source))

		require.NoError(t, err)
		assert.Equal(t, []string{sourceWorkspaceName}, initializedTarget.listWorkspaceNames(t))
		assert.Equal(t, source.keyContent, initializedTarget.readKey(t))
	})
}

func TestStartupRecoveryConsumption(t *testing.T) {
	t.Parallel()

	t.Run("skips a consumed archive on restart, renamed, and after an upgrade", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		recoveredTarget := newFreshTarget(t)
		require.NoError(t, recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, source)))
		recoveredTarget.execute(
			t,
			`INSERT INTO workspaces (id, name, created_at) VALUES (?, 'Created after recovery', now())`,
			uuid.New(),
		)
		content, err := os.ReadFile(source.path)
		require.NoError(t, err)
		renamedSource := source
		renamedSource.path = writeInput(t, "renamed.dbbackup", content)

		for _, spec := range []startupSpec{
			{archive: source, release: testRelease},
			{archive: renamedSource, release: testRelease},
			{archive: source, release: "v9.9.10"},
		} {
			err := recoveredTarget.openForServer(t, recoveredTarget.newStartupWith(t, spec))

			require.NoError(t, err)
		}

		assert.Equal(t, []string{"Created after recovery", sourceWorkspaceName}, recoveredTarget.listWorkspaceNames(t))
		assert.Equal(t, source.keyContent, recoveredTarget.readKey(t))
		assert.Contains(t, recoveredTarget.getLogs(), "startup recovery skipped an archive it already recovered")
		assert.Contains(t, recoveredTarget.getLogs(), `"level":"INFO","msg":"startup recovery skipped`)
		assert.Len(t, recoveredTarget.readJournal(t).ConsumedStartupArchives, 1)
	})

	t.Run("refuses other bytes at the configured path of a consumed archive", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		otherSource := newSourceArchiveWith(
			t,
			archiveSpec{release: testRelease, bundleDigest: "", workspaceName: "Other"},
		)
		recoveredTarget := newFreshTarget(t)
		require.NoError(t, recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, source)))
		otherContent, err := os.ReadFile(otherSource.path)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(source.path, otherContent, 0o600))
		source.keyPath = otherSource.keyPath

		err = recoveredTarget.openForServer(t, recoveredTarget.newStartup(t, source))

		_ = requireRefusal(t, err, RefusalExistingAccounts)
		assert.Equal(t, []string{sourceWorkspaceName}, recoveredTarget.listWorkspaceNames(t))
	})
}

func TestStartupRecoveryExclusiveAccess(t *testing.T) {
	t.Parallel()

	t.Run("refuses an unconsumed archive while another process holds the lifecycle lock", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		generatedKey := initializedTarget.readKey(t)
		otherProcess := holdSharedLock(t, initializedTarget)
		defer func() { _ = otherProcess.Close() }()

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, source))

		_ = requireRefusal(t, err, RefusalNoExclusiveAccess)
		assertTargetUnchanged(t, initializedTarget, generatedKey)
	})

	t.Run("starts beside another process when no recovery is configured", func(t *testing.T) {
		t.Parallel()
		initializedTarget := newInitializedTarget(t)
		otherProcess := holdSharedLock(t, initializedTarget)
		defer func() { _ = otherProcess.Close() }()

		err := initializedTarget.openForServer(t, initializedTarget.newStartup(t, sourceArchive{}))

		require.NoError(t, err)
	})

	t.Run("refuses user state created while the archive was imported", func(t *testing.T) {
		t.Parallel()
		source := newSourceArchive(t)
		initializedTarget := newInitializedTarget(t)
		generatedKey := initializedTarget.readKey(t)
		startup := initializedTarget.newStartup(t, source)
		journal := prepareStartupOperation(t, startup, source)

		initializedTarget.execute(
			t,
			`INSERT INTO users (email, password_creation_time, role, name, status, is_root_admin)
			VALUES ('root@target.local', now(), 'ADMIN', 'Root', 'ACTIVE', true)`,
		)
		_, err := startup.publishPrepared(t.Context(), journal)

		_ = requireRefusal(t, err, RefusalExistingAccounts)
		assertTargetUnchanged(t, initializedTarget, generatedKey)
		var emails []string
		withTestDatabase(t, initializedTarget.dsn, func(db *gorm.DB) {
			require.NoError(t, db.Raw("SELECT email FROM users").Scan(&emails).Error)
		})
		assert.Equal(t, []string{"root@target.local"}, emails)
	})
}

func requireRefusal(t *testing.T, err error, reason RefusalReason) *RefusalError {
	t.Helper()

	var refusal *RefusalError
	require.ErrorAs(t, err, &refusal)
	require.Equal(t, reason, refusal.Reason, err.Error())

	return refusal
}

// assertTargetUnchanged checks that the target kept its key and database and that nothing staged
// is left.
func assertTargetUnchanged(t *testing.T, tg target, keyBefore []byte) {
	t.Helper()

	assert.Equal(t, keyBefore, tg.readKey(t))
	assert.NotContains(t, tg.listWorkspaceNames(t), sourceWorkspaceName)
	assert.Empty(t, tg.listRecoveryDatabases(t))
	assert.Nil(t, tg.readJournal(t).Operation)
	assertNoOperationFiles(t, tg)
}

func assertNoOperationFiles(t *testing.T, tg target) {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(tg.getDirectory().root, "operations"))
	if os.IsNotExist(err) {
		return
	}

	require.NoError(t, err)
	assert.Empty(t, entries)
}

func holdSharedLock(t *testing.T, tg target) *LifecycleLock {
	t.Helper()

	require.NoError(t, tg.getDirectory().Ensure())
	lock, err := OpenLifecycleLock(tg.getDirectory())
	require.NoError(t, err)
	isShared, err := lock.TryShared()
	require.NoError(t, err)
	require.True(t, isShared)

	return lock
}

// prepareStartupOperation runs a startup recovery up to IMPORTED, as a server does before its
// final check under the exclusive lock.
func prepareStartupOperation(t *testing.T, startup *Startup, source sourceArchive) Journal {
	t.Helper()

	require.NoError(t, startup.directory.Ensure())
	archiveSha256, err := hashFile(source.path)
	require.NoError(t, err)
	masterKey, err := crypto.NewMasterKey(source.keyContent)
	require.NoError(t, err)
	journal, err := startup.journal.Read()
	require.NoError(t, err)

	journal, err = startup.prepare(t.Context(), journal, archiveSha256, source.keyContent, masterKey)
	require.NoError(t, err)
	require.Equal(t, PhaseImported, journal.Operation.Phase)

	return journal
}

func writeInput(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, content, 0o600))

	return path
}
