package e2e

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const (
	productionWorkspaceName = "Production"
	archiveWorkspaceName    = "Local archive"
	unreachableBucket       = "databasus-recovery-unreachable"
	// historicalMigration is an applied migration that no current migration file carries, like one
	// a release shipped and a later release deleted. Recovery must keep its history row.
	historicalMigration = "20200101000000"
	// notifierHeaderBytes makes the encrypted notifier secret large enough that the Databasus backup
	// spans several encrypted chunks, which an exact chunk-boundary cut needs.
	notifierHeaderBytes = 2 << 20
)

// lostInstallation is what remains of the source installation once its server is gone: the two
// files the owner kept, and what the scenarios compare a recovered installation with.
type lostInstallation struct {
	archive       []byte
	archiveSha256 string
	sourceRelease string
	masterKey     []byte
	rootEmail     string
	memberEmail   string
	// rootToken is a session the lost installation issued; the snapshot carries it.
	rootToken string
	// remoteBackupID is an encrypted logical backup in the remote storage at the snapshot.
	remoteDatabaseID uuid.UUID
	remoteBackupID   uuid.UUID
	// laterBackupID was stored after the snapshot, so a recovered catalog does not know it.
	laterBackupID uuid.UUID
	// localBackupID was stored in a local storage of the lost server.
	localDatabaseID uuid.UUID
	localBackupID   uuid.UUID
	localStorageID  uuid.UUID
	physical        physicalChain
}

// loseSourceInstallation runs a source installation with accounts, workspaces, encrypted storage,
// notifier and database credentials and their backups, saves a Databasus backup to the remote
// storage, keeps the archive and the master key apart from it and stops the installation for good.
func loseSourceInstallation(t *testing.T, cycle *recoveryCycle) lostInstallation {
	t.Helper()

	source := cycle.newInstallation(t, "source", namedVolumeLayout)
	instance := source.start(t)

	root := signUp(t, instance, newAccountEmail("root"))
	openRegistrations(t, root)
	member := signUp(t, instance, newAccountEmail("member"))

	production := root.createWorkspace(t, productionWorkspaceName)
	localArchive := root.createWorkspace(t, archiveWorkspaceName)
	remoteStorageID := createRemoteStorage(t, root, production.Id, backupBucket)
	unreachableStorageID := createUnreachableStorage(t, cycle, root, production.Id)
	localStorageID := createLocalStorage(t, root, localArchive.Id)
	notifierID := createWebhookNotifier(t, root, production.Id)

	createLogicalRows(t, cycle.openSourceDatabase(t, sourceDatabaseName))
	remoteDatabase := createDatabase(t, root, production.Id, newSourceDatabaseSettings(t), []uuid.UUID{notifierID})
	enableLogicalBackups(t, root, remoteDatabase.Id, remoteStorageID)
	remoteBackup := backUpLogically(t, root, remoteDatabase.Id)
	localDatabase := createDatabase(t, root, localArchive.Id, newSourceDatabaseSettings(t), []uuid.UUID{})
	enableLogicalBackups(t, root, localDatabase.Id, localStorageID)
	localBackup := backUpLogically(t, root, localDatabase.Id)
	physical := buildPhysicalChain(t, cycle, root, production.Id, remoteStorageID)

	instance.queryInternalDatabase(t,
		"INSERT INTO goose_db_version (version_id, is_applied) VALUES ("+historicalMigration+", true)")

	archive, sourceRelease := saveDatabasusBackup(t, instance, root, remoteStorageID, unreachableStorageID)
	masterKey := exportMasterKey(t, instance, root)
	laterBackup := backUpLogically(t, root, remoteDatabase.Id)

	instance.stop(t)

	archiveSha256 := sha256.Sum256(archive)

	return lostInstallation{
		archive:          archive,
		archiveSha256:    hex.EncodeToString(archiveSha256[:]),
		sourceRelease:    sourceRelease,
		masterKey:        masterKey,
		rootEmail:        root.Email,
		memberEmail:      member.Email,
		rootToken:        root.Token,
		remoteDatabaseID: remoteDatabase.Id,
		remoteBackupID:   remoteBackup.Id,
		laterBackupID:    laterBackup.Id,
		localDatabaseID:  localDatabase.Id,
		localBackupID:    localBackup.Id,
		localStorageID:   localStorageID,
		physical:         physical,
	}
}

func openRegistrations(t *testing.T, root accountSession) {
	t.Helper()

	response, err := root.Client.UpdateInstanceSettingsWithResponse(t.Context(), apiclient.InstanceSettingsUpdate{
		IsAllowExternalRegistrations:      true,
		IsAllowMemberInvitations:          true,
		IsMemberAllowedToCreateWorkspaces: true,
		TwoFactorRequirement:              apiclient.TwoFactorRequirementOFF,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), string(response.Body))
}

func createStorage(
	t *testing.T,
	owner accountSession,
	workspaceID uuid.UUID,
	settings apiclient.StorageSettings,
) uuid.UUID {
	t.Helper()

	response, err := owner.Client.CreateStorageWithResponse(t.Context(), apiclient.StorageCreate{
		WorkspaceId: workspaceID,
		Name:        "Storage " + uuid.NewString()[:8],
		Settings:    settings,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return response.JSON201.Id
}

func createRemoteStorage(t *testing.T, owner accountSession, workspaceID uuid.UUID, bucket string) uuid.UUID {
	t.Helper()

	var settings apiclient.StorageSettings
	require.NoError(t, settings.FromS3StorageSettings(apiclient.S3StorageSettings{
		Type:        apiclient.S3StorageSettingsTypeS3,
		S3Bucket:    bucket,
		S3Region:    new(containers.MinioRegion),
		S3Endpoint:  new(fmt.Sprintf("http://%s:9000", minioNetworkAlias)),
		S3AccessKey: new(containers.MinioRootUser),
		S3SecretKey: new(containers.MinioRootPassword),
	}))

	return createStorage(t, owner, workspaceID, settings)
}

// createUnreachableStorage is a remote storage whose bucket disappears after it was saved, so every
// copy to it fails while copies to the other destination succeed.
func createUnreachableStorage(
	t *testing.T,
	cycle *recoveryCycle,
	owner accountSession,
	workspaceID uuid.UUID,
) uuid.UUID {
	t.Helper()

	require.NoError(t, cycle.storageClient.MakeBucket(t.Context(), unreachableBucket,
		minio.MakeBucketOptions{Region: containers.MinioRegion}))
	storageID := createRemoteStorage(t, owner, workspaceID, unreachableBucket)
	require.NoError(t, cycle.storageClient.RemoveBucket(t.Context(), unreachableBucket))

	return storageID
}

func createLocalStorage(t *testing.T, owner accountSession, workspaceID uuid.UUID) uuid.UUID {
	t.Helper()

	var settings apiclient.StorageSettings
	require.NoError(t, settings.FromLocalStorageSettings(apiclient.LocalStorageSettings{
		Type: apiclient.LocalStorageSettingsTypeLOCAL,
	}))

	return createStorage(t, owner, workspaceID, settings)
}

func createWebhookNotifier(t *testing.T, owner accountSession, workspaceID uuid.UUID) uuid.UUID {
	t.Helper()

	headerValue := make([]byte, notifierHeaderBytes/2)
	_, err := rand.Read(headerValue)
	require.NoError(t, err)

	var settings apiclient.NotifierSettings
	require.NoError(t, settings.FromWebhookNotifierSettings(apiclient.WebhookNotifierSettings{
		Type:               apiclient.WebhookNotifierSettingsTypeWEBHOOK,
		WebhookUrl:         "http://recovery-webhook.invalid/hook",
		WebhookMethod:      apiclient.WebhookNotifierSettingsWebhookMethodPOST,
		AcceptedCategories: []apiclient.NotificationCategory{apiclient.NotificationCategoryALL},
		Headers: []apiclient.WebhookHeader{
			{Name: "X-Recovery-Secret", Value: new(hex.EncodeToString(headerValue))},
		},
	}))

	response, err := owner.Client.CreateNotifierWithResponse(t.Context(), apiclient.NotifierCreate{
		WorkspaceId: workspaceID,
		Name:        "Recovery webhook",
		Settings:    settings,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, response.StatusCode(), string(response.Body))

	return response.JSON201.Id
}

// saveDatabasusBackup selects both destinations, which starts a Databasus backup at once, and
// downloads the copy stored in the reachable one while the other keeps failing.
func saveDatabasusBackup(
	t *testing.T,
	instance runningInstance,
	root accountSession,
	remoteStorageID, unreachableStorageID uuid.UUID,
) ([]byte, string) {
	t.Helper()

	settingsResponse, err := root.Client.GetInstanceBackupSettingsWithResponse(t.Context())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, settingsResponse.StatusCode(), string(settingsResponse.Body))
	settings := *settingsResponse.JSON200
	settings.StorageIds = []uuid.UUID{remoteStorageID, unreachableStorageID}

	updateResponse, err := root.Client.UpdateInstanceBackupSettingsWithResponse(t.Context(), settings)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updateResponse.StatusCode(), string(updateResponse.Body))

	var (
		storedCopy  apiclient.InstanceBackupCopy
		savedBackup apiclient.InstanceBackup
	)

	testwait.Within(backupLimit).Until(t, "the Databasus backup stores one copy and retries the other",
		func(ctx context.Context) error {
			response, err := root.Client.ListInstanceBackupsWithResponse(ctx, nil)
			if err != nil {
				return err
			}

			if response.StatusCode() != http.StatusOK || len(response.JSON200.Items) == 0 {
				return fmt.Errorf("no Databasus backup is listed yet: %s", response.Body)
			}

			savedBackup = response.JSON200.Items[0]

			return getPartialCopy(savedBackup, remoteStorageID, &storedCopy)
		})

	tokenResponse, err := root.Client.CreateInstanceBackupDownloadTokenWithResponse(t.Context(), savedBackup.Id,
		apiclient.InstanceBackupDownloadTokenCreate{CopyId: storedCopy.Id})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, tokenResponse.StatusCode(), string(tokenResponse.Body))

	downloadResponse, err := instance.newClient(t).DownloadInstanceBackupWithResponse(t.Context(),
		&apiclient.DownloadInstanceBackupParams{Token: tokenResponse.JSON201.Token})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, downloadResponse.StatusCode())
	require.NotEmpty(t, downloadResponse.Body)

	return downloadResponse.Body, savedBackup.SourceRelease
}

// getPartialCopy accepts the backup once its copy in the remote storage is stored and the copy in
// the unreachable storage failed and waits for a retry: a partial storage failure.
func getPartialCopy(
	backup apiclient.InstanceBackup,
	remoteStorageID uuid.UUID,
	storedCopy *apiclient.InstanceBackupCopy,
) error {
	isRetrying := false

	for _, backupCopy := range backup.Copies {
		switch {
		case backupCopy.StorageId != nil && *backupCopy.StorageId == remoteStorageID:
			if backupCopy.Status != apiclient.InstanceBackupCopyStatusSTORED || !backupCopy.IsDownloadable {
				return fmt.Errorf("the remote copy is %s", backupCopy.Status)
			}

			*storedCopy = backupCopy
		case backupCopy.Status == apiclient.InstanceBackupCopyStatusRETRYING && backupCopy.LastError != nil:
			isRetrying = true
		}
	}

	if storedCopy.Id == uuid.Nil || !isRetrying {
		return fmt.Errorf("the backup %s has not stored one copy and failed the other yet", backup.Id)
	}

	return nil
}

func exportMasterKey(t *testing.T, instance runningInstance, root accountSession) []byte {
	t.Helper()

	exportResponse, err := root.Client.CreateMasterKeyExportWithResponse(
		t.Context(),
		apiclient.MasterKeyExportCreate{Code: nil},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, exportResponse.StatusCode(), string(exportResponse.Body))

	downloadResponse, err := instance.newClient(t).DownloadMasterKeyWithResponse(context.WithoutCancel(t.Context()),
		&apiclient.DownloadMasterKeyParams{Token: exportResponse.JSON201.Token})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, downloadResponse.StatusCode())
	require.NotEmpty(t, downloadResponse.Body)

	return downloadResponse.Body
}
