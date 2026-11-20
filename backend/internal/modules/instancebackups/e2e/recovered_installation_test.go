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

// requireRecoveredSnapshot checks what a recovery publishes: the snapshot's accounts with root
// admin among them, its workspaces, no session of the lost installation and a completed recovery.
// It returns root admin's new session and that recovery.
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
	require.NotNil(t, status.Recovery)
	assert.Equal(t, apiclient.InstanceRecoveryStatusCOMPLETED, status.Recovery.Status)
	assert.Equal(t, lost.archiveSha256, status.Recovery.ArchiveSha256)

	assert.Equal(t, "1", instance.queryInternalDatabase(t,
		"SELECT count(*) FROM goose_db_version WHERE version_id = "+historicalMigration+" AND is_applied"),
		"recovery keeps the applied migration no current file carries")

	return root, *status.Recovery
}

// requireRunningRecovery checks the work a recovered installation does without any confirmation:
// a new physical FULL after the recovery, restores of the database backups the snapshot catalogs,
// and a recovered local storage used from the first start without any test. The caller holds the source server's turn, because a recovered installation streams
// its WAL from its first start.
func requireRunningRecovery(
	t *testing.T,
	instance runningInstance,
	root accountSession,
	recovery apiclient.InstanceRecovery,
	lost lostInstallation,
) {
	t.Helper()

	cycle := instance.cycle

	require.NotNil(t, recovery.CompletedAt)
	waitForPhysicalBackup(t, root, lost.physical.databaseID, physicalBackupQuery{
		kind: apiclient.PhysicalBackupListRowTypeFULL, createdAfter: recovery.CompletedAt.Add(-time.Minute),
	})
	requireRecoveredLocalStorageInUse(t, instance, root, lost)

	recoveredBackupIDs := listLogicalBackupIDs(t, root, lost.remoteDatabaseID)
	assert.True(t, containsBackup(recoveredBackupIDs, lost.remoteBackupID))
	assert.False(t, containsBackup(recoveredBackupIDs, lost.laterBackupID),
		"a backup stored after the snapshot is not claimed as recovered history")
	requireLogicalRestoreOfSnapshot(t, cycle, root, lost.remoteBackupID)
	requirePointInTimeRestoreOfSnapshot(t, cycle, instance, root, lost.physical)
}

// requireRecoveredLocalStorageInUse checks that a recovered local storage works without any test:
// a new backup is stored in it and downloads, while a backup whose file stayed on the lost server
// answers that the file is missing.
func requireRecoveredLocalStorageInUse(
	t *testing.T,
	instance runningInstance,
	root accountSession,
	lost lostInstallation,
) {
	t.Helper()

	newBackup := backUpLogically(t, root, lost.localDatabaseID)
	newDownload := downloadLogicalBackup(t, instance, root, newBackup.Id)
	missingDownload := downloadLogicalBackup(t, instance, root, lost.localBackupID)

	assert.Equal(t, http.StatusOK, newDownload.StatusCode(), string(newDownload.Body))
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
