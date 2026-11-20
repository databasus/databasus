package e2e

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

const (
	renamedArchivePath = dataDirectory + "/recovery/renamed-copy.dbbackup"
	otherReleaseLabel  = "v0.0.1-recovery-cycle"
)

// TestRecoveryCycle loses one source installation and recovers it into new data in every way an
// owner can. The scenarios share the source's two files, its database server and its remote
// storage, which survive the lost server; each owns its own data and containers.
func TestRecoveryCycle(t *testing.T) {
	t.Parallel()

	cycle := newRecoveryCycle(t)
	lost := loseSourceInstallation(t, cycle)

	t.Run("recovers a fresh named volume from mounted files and skips them on every later start", func(t *testing.T) {
		t.Parallel()
		runFreshVolumeRecovery(t, cycle, lost)
	})

	t.Run("recovers a bind-mounted directory after an ordinary first start and an interrupted publication",
		func(t *testing.T) {
			t.Parallel()
			runInitializedDirectoryRecovery(t, cycle, lost)
		})

	t.Run("refuses mounted files after a root account was created and keeps that installation", func(t *testing.T) {
		t.Parallel()
		installation := cycle.newInstallation(t, "account", namedVolumeLayout)
		firstStart := installation.start(t)
		targetRoot := signUp(t, firstStart, newAccountEmail("target-root"))
		firstStart.stop(t)
		targetKey := installation.readFile(t, installedKeyPath)
		installation.putRecoveryFiles(t, lost.archive, lost.masterKey)

		refusedStart := installation.startUntilExit(t, getRecoveryEnvironment()...)

		requireRefusalLog(t, refusedStart, "EXISTING_ACCOUNTS",
			"a sole root account included", "root-confirmed import", "separate fresh installation")
		assert.NotContains(t, refusedStart.logs, string(lost.masterKey))
		assert.Equal(t, targetKey, installation.readFile(t, installedKeyPath))
		ordinaryStart := installation.start(t)
		signIn(t, ordinaryStart, targetRoot.Email)
		assert.NotEqual(t, http.StatusOK, requestSignIn(t, ordinaryStart, lost.rootEmail).StatusCode())
	})

	t.Run("refuses an archive of another release before changing the database or the key", func(t *testing.T) {
		t.Parallel()
		installation := cycle.newInstallation(t, "release", namedVolumeLayout)
		otherRelease := "APP_VERSION=" + otherReleaseLabel
		installation.start(t, otherRelease).stop(t)
		targetKey := installation.readFile(t, installedKeyPath)
		installation.putRecoveryFiles(t, lost.archive, lost.masterKey)

		refusedStart := installation.startUntilExit(t, append(getRecoveryEnvironment(), otherRelease)...)

		requireRefusalLog(t, refusedStart, "INCOMPATIBLE_RELEASE", lost.sourceRelease, otherReleaseLabel)
		assert.Equal(t, targetKey, installation.readFile(t, installedKeyPath))
		ordinaryStart := installation.start(t, otherRelease)
		assert.NotEqual(t, http.StatusOK, requestSignIn(t, ordinaryStart, lost.rootEmail).StatusCode())
	})

	t.Run("imports the archive file into a separately initialized installation with its own root admin",
		func(t *testing.T) {
			t.Parallel()
			runFileImport(t, cycle, lost)
		})
}

func runFreshVolumeRecovery(t *testing.T, cycle *recoveryCycle, lost lostInstallation) {
	t.Helper()

	// Every recovered start streams WAL from the source server at once.
	cycle.physicalSourceTurn.Lock()
	defer cycle.physicalSourceTurn.Unlock()

	installation := cycle.newInstallation(t, "fresh", namedVolumeLayout)
	installation.putRecoveryFiles(t, lost.archive, lost.masterKey)

	recoveredStart := installation.start(t, getRecoveryEnvironment()...)

	assert.Contains(t, recoveredStart.getLogs(t), "recovery published")
	assert.NotContains(t, recoveredStart.getLogs(t), string(lost.masterKey))
	assert.Equal(t, lost.masterKey, installation.readFile(t, installedKeyPath))
	_, startupRecovery := requireRecoveredSnapshot(t, recoveredStart, lost)
	assert.Equal(t, apiclient.InstanceRecoveryOriginSTARTUP, startupRecovery.Origin)
	recoveredStart.stop(t)

	restartWithSameFiles := installation.start(t, getRecoveryEnvironment()...)
	requireSkipLog(t, restartWithSameFiles)
	root := signIn(t, restartWithSameFiles, lost.rootEmail)
	assert.Equal(t, startupRecovery.Id, root.getInstanceBackupStatus(t).Recovery.Id)
	restartWithSameFiles.stop(t)

	installation.putFile(t, renamedArchivePath, lost.archive)
	restartWithRenamedFile := installation.start(t,
		"RECOVERY_DUMP_PATH="+renamedArchivePath, "RECOVERY_MASTER_KEY_PATH="+recoveryKeyPath)
	requireSkipLog(t, restartWithRenamedFile)
	restartWithRenamedFile.stop(t)

	installation.putFile(t, recoveryArchivePath, append(bytes.Clone(lost.archive), 0))
	refusedStart := installation.startUntilExit(t, getRecoveryEnvironment()...)
	requireRefusalLog(t, refusedStart, "EXISTING_ACCOUNTS", "root-confirmed import")
	installation.putFile(t, recoveryArchivePath, lost.archive)

	uploadedRecovery := runImportAfterStartupRecovery(t, installation, lost)

	restartAfterImport := installation.start(t, getRecoveryEnvironment()...)
	requireSkipLog(t, restartAfterImport)
	root = signIn(t, restartAfterImport, lost.rootEmail)
	assert.Equal(t, uploadedRecovery.Id, root.getInstanceBackupStatus(t).Recovery.Id,
		"the imported state stays current and the startup archive is not imported again")

	upgradedStart := runOrdinaryUpgradeWithRetainedFiles(t, installation, restartAfterImport)
	root = signIn(t, upgradedStart, lost.rootEmail)
	assert.Equal(t, uploadedRecovery.Id, root.getInstanceBackupStatus(t).Recovery.Id)
	assert.ElementsMatch(t, []string{productionWorkspaceName, archiveWorkspaceName}, root.listWorkspaceNames(t))
	upgradedStart.stop(t)
}

// runImportAfterStartupRecovery deliberately imports the same archive bytes again through the file
// import, after work was done on the startup-recovered installation. The publication finishes in
// the same container, which no restart policy restarts. The recovered installation starts its full
// physical backup at once, so the checks of its work run before the container stops; a stop would
// interrupt that backup, and the next one waits for the schedule.
func runImportAfterStartupRecovery(
	t *testing.T,
	installation *installation,
	lost lostInstallation,
) apiclient.InstanceRecovery {
	t.Helper()

	instance := installation.start(t, getRecoveryEnvironment()...)
	startedAt := instance.getState(t).StartedAt
	root := signIn(t, instance, lost.rootEmail)
	root.createWorkspace(t, "Created after the startup recovery")

	preparedRecovery := prepareImport(t, root, lost.archive, lost.masterKey)
	assert.Equal(t, lost.archiveSha256, preparedRecovery.ArchiveSha256)
	applyResponse := applyImport(t, root, preparedRecovery.Id)
	require.NotNil(t, applyResponse, "the in-place publication answers the apply before it stops")
	require.Equal(t, http.StatusAccepted, applyResponse.StatusCode(), string(applyResponse.Body))

	restoredRoot := waitForRestoredSignIn(t, instance, root.Token, lost.rootEmail)

	assert.Equal(t, startedAt, instance.getState(t).StartedAt, "the same container serves the recovered installation")
	assert.ElementsMatch(t, []string{productionWorkspaceName, archiveWorkspaceName}, restoredRoot.listWorkspaceNames(t))
	status := restoredRoot.getInstanceBackupStatus(t)
	require.NotNil(t, status.Recovery)
	assert.Equal(t, apiclient.InstanceRecoveryStatusCOMPLETED, status.Recovery.Status)
	assert.Equal(t, preparedRecovery.Id, status.Recovery.Id)
	assert.Equal(t, apiclient.InstanceRecoveryOriginUPLOAD, status.Recovery.Origin)
	requireRunningRecovery(t, instance, restoredRoot, *status.Recovery, lost)
	instance.stop(t)

	return *status.Recovery
}

// runOrdinaryUpgradeWithRetainedFiles starts a newer release with the consumed archive and both
// paths still configured. The newer release is this image under another release label whose newest
// migration the database does not have yet: the migration is rolled back before the start.
func runOrdinaryUpgradeWithRetainedFiles(
	t *testing.T,
	installation *installation,
	instance runningInstance,
) runningInstance {
	t.Helper()

	newestMigration := readNewestMigration(t)
	instance.queryInternalDatabase(t, newestMigration.downStatements)
	instance.queryInternalDatabase(t, "DELETE FROM goose_db_version WHERE version_id = "+newestMigration.version)
	instance.stop(t)

	upgradedStart := installation.start(t, append(getRecoveryEnvironment(), "APP_VERSION="+otherReleaseLabel)...)

	requireSkipLog(t, upgradedStart)
	assert.Equal(t, "1", upgradedStart.queryInternalDatabase(t,
		"SELECT count(*) FROM goose_db_version WHERE version_id = "+newestMigration.version+" AND is_applied"),
		"the ordinary start applies the pending migration")

	return upgradedStart
}

type migrationFile struct {
	version        string
	downStatements string
}

func readNewestMigration(t *testing.T) migrationFile {
	t.Helper()

	migrationPaths, err := filepath.Glob(filepath.Join(getRepositoryRoot(t), "backend", "migrations", "*.sql"))
	require.NoError(t, err)
	require.NotEmpty(t, migrationPaths)
	slices.Sort(migrationPaths)
	newestPath := migrationPaths[len(migrationPaths)-1]

	content, err := os.ReadFile(newestPath)
	require.NoError(t, err)

	_, downSection, hasDown := strings.Cut(string(content), "-- +goose Down")
	require.True(t, hasDown, "%s has a down migration", newestPath)

	version, _, _ := strings.Cut(filepath.Base(newestPath), "_")

	return migrationFile{version: version, downStatements: strings.TrimSpace(downSection)}
}

func runInitializedDirectoryRecovery(t *testing.T, cycle *recoveryCycle, lost lostInstallation) {
	t.Helper()

	// Every recovered start streams WAL from the source server at once.
	cycle.physicalSourceTurn.Lock()
	defer cycle.physicalSourceTurn.Unlock()

	installation := cycle.newInstallation(t, "initialized", bindMountLayout)
	installation.start(t).stop(t)
	generatedKey := installation.readFile(t, installedKeyPath)
	require.NotEqual(t, lost.masterKey, generatedKey)

	installation.putRecoveryFiles(t, cutAtChunkBoundary(t, lost.archive), lost.masterKey)
	truncatedStart := installation.startUntilExit(t, getRecoveryEnvironment()...)
	requireRefusalLog(t, truncatedStart, "INVALID_ARCHIVE")
	assert.Equal(t, generatedKey, installation.readFile(t, installedKeyPath))

	installation.putRecoveryFiles(t, lost.archive, generatedKey)
	wrongKeyStart := installation.startUntilExit(t, getRecoveryEnvironment()...)
	requireRefusalLog(t, wrongKeyStart, "INVALID_MASTER_KEY")
	assert.NotContains(t, wrongKeyStart.logs, string(generatedKey))
	assert.Equal(t, generatedKey, installation.readFile(t, installedKeyPath))

	preservedStart := installation.start(t)
	assert.NotEqual(t, http.StatusOK, requestSignIn(t, preservedStart, lost.rootEmail).StatusCode())
	preservedStart.stop(t)

	// A directory where the key's temporary file goes stops the publication after the databases
	// were swapped and before the key is installed, as a crash at that moment does.
	installation.putRecoveryFiles(t, lost.archive, lost.masterKey)
	blockedKeyPath := installation.getHostPath(t, installedKeyPath+".tmp")
	require.NoError(t, os.Mkdir(blockedKeyPath, 0o700))

	interruptedStart := installation.startUntilExit(t, getRecoveryEnvironment()...)
	assert.NotEqual(t, 0, interruptedStart.exitCode)
	assert.Contains(t, interruptedStart.logs, "startup recovery publication did not finish")
	assert.NotContains(t, interruptedStart.logs, "recovery published")
	require.NoError(t, os.Remove(blockedKeyPath))

	resumedStart := installation.start(t, getRecoveryEnvironment()...)
	resumedLogs := resumedStart.getLogs(t)
	assert.Contains(t, resumedLogs, "finishing an interrupted recovery publication")
	assert.Contains(t, resumedLogs, "recovery published")
	assert.Contains(t, resumedLogs, "startup recovery skipped an archive it already recovered",
		"the finished publication consumed the archive, so it is not imported again")
	assert.Equal(t, lost.masterKey, installation.readFile(t, installedKeyPath))
	requireRecoveredSnapshot(t, resumedStart, lost)
	resumedStart.stop(t)

	restartedStart := installation.start(t, getRecoveryEnvironment()...)
	requireSkipLog(t, restartedStart)
	assert.NotContains(t, restartedStart.getLogs(t), "recovery published")
	restartedStart.stop(t)
}

func runFileImport(t *testing.T, cycle *recoveryCycle, lost lostInstallation) {
	t.Helper()

	// Every recovered start streams WAL from the source server at once.
	cycle.physicalSourceTurn.Lock()
	defer cycle.physicalSourceTurn.Unlock()

	installation := cycle.newInstallation(t, "import", bindMountLayout)
	instance := installation.start(t)
	targetRoot := signUp(t, instance, newAccountEmail("target-root"))
	targetKey := installation.readFile(t, installedKeyPath)

	failedRecovery := waitForPreparation(t, targetRoot, uploadArchive(t, targetRoot, lost.archive, targetKey).Id)
	assert.Equal(t, apiclient.InstanceRecoveryStatusFAILED, failedRecovery.Status)
	require.NotNil(t, failedRecovery.Failure)
	assert.Equal(t, http.StatusOK, targetRoot.getOwnAccount(t).StatusCode())
	assert.Equal(t, targetKey, installation.readFile(t, installedKeyPath))

	cancelledRecovery := prepareImport(t, targetRoot, lost.archive, lost.masterKey)
	assert.Equal(t, lost.sourceRelease, *cancelledRecovery.SourceRelease)
	assert.NotNil(t, cancelledRecovery.SnapshotAt)
	cancelResponse, err := targetRoot.Client.CancelInstanceRecoveryWithResponse(t.Context(), cancelledRecovery.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, cancelResponse.StatusCode(), string(cancelResponse.Body))
	assert.Equal(t, apiclient.InstanceRecoveryStatusCANCELLED, cancelResponse.JSON200.Status)
	assert.Equal(t, http.StatusOK, targetRoot.getOwnAccount(t).StatusCode())

	preparedRecovery := prepareImport(t, targetRoot, lost.archive, lost.masterKey)
	blockedKeyPath := installation.getHostPath(t, installedKeyPath+".tmp")
	require.NoError(t, os.Mkdir(blockedKeyPath, 0o700))

	// The blocked key file stops the publication and the process at once, which can take the apply's
	// answer with it: an answer lost to a crash, which the next start resolves from the journal.
	applyImport(t, targetRoot, preparedRecovery.Id)

	interruptedPublication := instance.waitForExit(t)
	assert.NotEqual(t, 0, interruptedPublication.exitCode)
	assert.Contains(t, interruptedPublication.logs, "failed to publish the accepted recovery")
	require.NoError(t, os.Remove(blockedKeyPath))

	resumedStart := installation.start(t)
	assert.Contains(t, resumedStart.getLogs(t), "finishing an interrupted recovery publication")
	assert.Equal(t, lost.masterKey, installation.readFile(t, installedKeyPath))
	replacedSession := accountSession{
		Client: resumedStart.newSessionClient(t, targetRoot.Token),
		UserID: targetRoot.UserID,
		Email:  targetRoot.Email,
		Token:  targetRoot.Token,
	}
	assert.Equal(t, http.StatusUnauthorized, replacedSession.getOwnAccount(t).StatusCode())
	assert.NotEqual(t, http.StatusOK, requestSignIn(t, resumedStart, targetRoot.Email).StatusCode())

	root, importedRecovery := requireRecoveredSnapshot(t, resumedStart, lost)
	assert.Equal(t, preparedRecovery.Id, importedRecovery.Id)
	assert.Equal(t, apiclient.InstanceRecoveryOriginUPLOAD, importedRecovery.Origin)

	repeatedApply := applyImport(t, root, preparedRecovery.Id)
	require.NotNil(t, repeatedApply)
	require.Equal(t, http.StatusAccepted, repeatedApply.StatusCode(), string(repeatedApply.Body))
	assert.Equal(t, preparedRecovery.Id, repeatedApply.JSON202.Id)
	assert.Equal(t, apiclient.InstanceRecoveryStatusCOMPLETED, repeatedApply.JSON202.Status)
	assert.Equal(t, 1, strings.Count(resumedStart.getLogs(t), "recovery published"),
		"repeating the apply of a published recovery publishes nothing again")
	assert.Equal(t, preparedRecovery.Id, root.getInstanceBackupStatus(t).Recovery.Id)

	requireRunningRecovery(t, resumedStart, root, importedRecovery, lost)
	resumedStart.stop(t)
}
