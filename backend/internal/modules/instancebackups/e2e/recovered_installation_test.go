package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

const keptBackupFile = dataDirectory + "/backups/kept-on-the-new-host.dump"

// requireRecoveredSnapshot checks what a recovery publishes: the snapshot's accounts with root
// admin among them, its workspaces, no session of the lost installation and a durable pause. It
// returns root admin's new session and the recovery that paused the installation.
func requireRecoveredSnapshot(
	t *testing.T,
	instance runningInstance,
	lost lostInstallation,
) (accountSession, apiclient.InstanceRecovery) {
	t.Helper()

	root := signIn(t, instance, lost.rootEmail)
	rootAccount := root.getOwnAccount(t)
	require.Equal(t, http.StatusOK, rootAccount.StatusCode(), string(rootAccount.Body))
	assert.True(t, rootAccount.JSON200.IsRootAdmin)
	signIn(t, instance, lost.memberEmail)
	assert.ElementsMatch(t, []string{productionWorkspaceName, archiveWorkspaceName}, root.listWorkspaceNames(t))

	copiedSession := accountSession{
		Client: instance.newSessionClient(t, lost.rootToken),
		UserID: root.UserID,
		Email:  lost.rootEmail,
		Token:  lost.rootToken,
	}
	assert.Equal(t, http.StatusUnauthorized, copiedSession.getOwnAccount(t).StatusCode())

	status := root.getInstanceBackupStatus(t)
	require.True(t, status.IsRecoveryPaused)
	require.NotNil(t, status.Recovery)
	assert.Equal(t, apiclient.InstanceRecoveryStatusPAUSED, status.Recovery.Status)
	assert.Equal(t, lost.archiveSha256, status.Recovery.ArchiveSha256)

	assert.Equal(t, "1", instance.queryInternalDatabase(t,
		"SELECT count(*) FROM goose_db_version WHERE version_id = "+historicalMigration+" AND is_applied"),
		"recovery keeps the applied migration no current file carries")

	return root, *status.Recovery
}

// requireNoRemoteEffectsWhilePaused tries the manual actions that write to a storage and checks
// that the remote storage holds exactly the objects it held before the recovery.
func requireNoRemoteEffectsWhilePaused(
	t *testing.T,
	cycle *recoveryCycle,
	root accountSession,
	lost lostInstallation,
	storedObjectCount int,
) {
	t.Helper()

	backupResponse, err := root.Client.StartLogicalBackupWithResponse(t.Context(),
		apiclient.LogicalBackupStart{DatabaseId: lost.remoteDatabaseID})
	require.NoError(t, err)
	requireProblem(t, backupResponse.StatusCode(), backupResponse.Body,
		http.StatusConflict, apiclient.ProblemCodeRecoveryPaused)

	settingsResponse, err := root.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(),
		apiclient.InstanceBackupSettings{
			RetentionDays: 30,
			Schedule:      apiclient.Schedule{Type: apiclient.ScheduleTypeHOURLY},
			StorageIds:    []uuid.UUID{},
		})
	require.NoError(t, err)
	requireProblem(t, settingsResponse.StatusCode(), settingsResponse.Body,
		http.StatusConflict, apiclient.ProblemCodeRecoveryPaused)

	assert.Equal(t, storedObjectCount, cycle.countStoredObjects(t))
}

// requireActivatedRecovery activates the paused installation and checks the work it resumes: a
// new physical FULL, restores of the database backups the snapshot catalogs, and a recovered local
// storage that stays unavailable until an editor validates its actual backup directory. The caller
// holds the source server's turn, because an activated installation streams its WAL.
func requireActivatedRecovery(
	t *testing.T,
	instance runningInstance,
	root accountSession,
	recoveryID uuid.UUID,
	lost lostInstallation,
) {
	t.Helper()

	cycle := instance.cycle

	activatedAt := time.Now().Add(-time.Minute)
	activatedRecovery := root.activateRecovery(t, recoveryID)
	assert.Equal(t, apiclient.InstanceRecoveryStatusACTIVE, activatedRecovery.Status)
	assert.False(t, root.getInstanceBackupStatus(t).IsRecoveryPaused)

	waitForPhysicalBackup(t, root, lost.physical.databaseID, physicalBackupQuery{
		kind: apiclient.PhysicalBackupListRowTypeFULL, createdAfter: activatedAt,
	})
	requireLocalStorageValidation(t, instance, root, lost)

	recoveredBackupIDs := listLogicalBackupIDs(t, root, lost.remoteDatabaseID)
	assert.True(t, containsBackup(recoveredBackupIDs, lost.remoteBackupID))
	assert.False(t, containsBackup(recoveredBackupIDs, lost.laterBackupID),
		"a backup stored after the snapshot is not claimed as recovered history")
	requireLogicalRestoreOfSnapshot(t, cycle, root, lost.remoteBackupID)
	requirePointInTimeRestoreOfSnapshot(t, cycle, instance, root, lost.physical)
}

func requireLocalStorageValidation(t *testing.T, instance runningInstance, root accountSession, lost lostInstallation) {
	t.Helper()

	assert.Equal(t, apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
		getRecoveryValidationStatus(t, root, lost.localStorageID), "activation alone does not validate it")

	unvalidatedDownload := downloadLogicalBackup(t, instance, root, lost.localBackupID)
	requireProblem(t, unvalidatedDownload.StatusCode(), unvalidatedDownload.Body,
		http.StatusConflict, apiclient.ProblemCodeStorageUnavailable)

	instance.executeAsRuntimeUser(t, "printf 'kept backup' > "+keptBackupFile)
	instance.executeAsRoot(t, "chmod 0000 "+dataDirectory+"/backups")

	failedTest := testStorage(t, root, lost.localStorageID)
	assert.False(t, failedTest.IsConnected, "a writable temporary folder does not validate the backup folder")
	assert.Equal(t, apiclient.StorageRecoveryValidationStatusREQUIRESVALIDATION,
		getRecoveryValidationStatus(t, root, lost.localStorageID))

	instance.executeAsRoot(t, "chmod 0770 "+dataDirectory+"/backups")

	passedTest := testStorage(t, root, lost.localStorageID)
	require.True(t, passedTest.IsConnected, "%v", passedTest.Error)
	assert.Equal(t, apiclient.StorageRecoveryValidationStatusAVAILABLE,
		getRecoveryValidationStatus(t, root, lost.localStorageID))

	keptContent, err := instance.cycle.docker.Execute(t.Context(), instance.containerName,
		[]string{"cat", keptBackupFile})
	require.NoError(t, err)
	assert.Equal(t, "kept backup", keptContent, "the probe leaves existing backup files unchanged")

	missingDownload := downloadLogicalBackup(t, instance, root, lost.localBackupID)
	requireProblem(t, missingDownload.StatusCode(), missingDownload.Body,
		http.StatusNotFound, apiclient.ProblemCodeNotFound)
}

// downloadLogicalBackup downloads a backup whose file lived on the lost server's local storage.
func downloadLogicalBackup(
	t *testing.T,
	instance runningInstance,
	root accountSession,
	backupID uuid.UUID,
) *apiclient.DownloadLogicalBackupResponse {
	t.Helper()

	tokenResponse, err := root.Client.CreateDownloadTokenWithResponse(t.Context(),
		apiclient.DownloadTokenCreate{BackupId: backupID})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, tokenResponse.StatusCode(), string(tokenResponse.Body))

	response, err := instance.newClient(t).DownloadLogicalBackupWithResponse(t.Context(), backupID,
		&apiclient.DownloadLogicalBackupParams{Token: tokenResponse.JSON201.Token})
	require.NoError(t, err)

	return response
}

func testStorage(t *testing.T, root accountSession, storageID uuid.UUID) apiclient.StorageTestResult {
	t.Helper()

	response, err := root.Client.TestStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return *response.JSON200
}

func getRecoveryValidationStatus(
	t *testing.T,
	root accountSession,
	storageID uuid.UUID,
) apiclient.StorageRecoveryValidationStatus {
	t.Helper()

	response, err := root.Client.GetStorageWithResponse(t.Context(), storageID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))

	return response.JSON200.RecoveryValidationStatus
}

// requireRefusalLog checks the operator-facing record of a startup recovery that did not run.
func requireRefusalLog(t *testing.T, exited exitedInstance, reason string, guidance ...string) {
	t.Helper()

	assert.NotEqual(t, 0, exited.exitCode, exited.logs)
	assert.Contains(t, exited.logs, "startup recovery did not run")
	assert.Contains(t, exited.logs, "reason="+reason)
	assert.Contains(t, exited.logs, "current_installation=unchanged")
	assert.NotContains(t, exited.logs, "recovery published")
	assert.NotContains(t, exited.logs, "http server listening")

	for _, expectedText := range guidance {
		assert.Contains(t, exited.logs, expectedText)
	}
}

func requireSkipLog(t *testing.T, instance runningInstance) {
	t.Helper()

	logs := instance.getLogs(t)
	assert.Contains(t, logs, "startup recovery skipped an archive it already recovered")
	assert.Contains(t, logs, `level=INFO msg="startup recovery skipped`)
	assert.NotContains(t, logs, "startup recovery did not run")
}

// cutAtChunkBoundary cuts the archive exactly after an encrypted chunk before its last one, which
// the stream format alone accepts as a complete stream. The archive's outer header and the stream
// header precede the length-prefixed chunks.
func cutAtChunkBoundary(t *testing.T, archive []byte) []byte {
	t.Helper()

	const encryptedStreamStart = 26 + 64

	boundaries := []int{}

	for offset := encryptedStreamStart; offset < len(archive); {
		chunkLength := int(uint32(archive[offset])<<24 | uint32(archive[offset+1])<<16 |
			uint32(archive[offset+2])<<8 | uint32(archive[offset+3]))
		offset += 4 + chunkLength
		boundaries = append(boundaries, offset)
	}

	require.Equal(t, len(archive), boundaries[len(boundaries)-1], "the archive ends after its last chunk")
	require.GreaterOrEqual(t, len(boundaries), 2, "the archive spans several encrypted chunks")

	return archive[:boundaries[len(boundaries)-2]]
}
