// Package enginetest holds what the end-to-end tests of every engine share: a team that backs up a
// real database and restores it through the API, and the connections the tests reach their servers
// with. Only those tests import it.
package enginetest

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

const (
	bytesInMegabyte = 1024 * 1024
	// sizeToleranceMb absorbs the rounding of the recorded size.
	sizeToleranceMb = 0.0001
)

// Team is an instance with a workspace owned by a user who is not an instance admin, and a local
// storage of that workspace the tests back up into.
type Team struct {
	App       *apptest.App
	Admin     apptest.Session
	Owner     apptest.Session
	Workspace apiclient.Workspace
	StorageID uuid.UUID
}

func NewTeam(t *testing.T, options ...apptest.Option) Team {
	t.Helper()

	app := apptest.New(t, options...)
	admin := app.CreateAdmin(t)
	owner := app.CreateNonAdminUser(t, admin)
	workspace := apptest.CreateWorkspace(t, owner, "Team "+NewUniqueSuffix())

	return Team{
		App:       app,
		Admin:     admin,
		Owner:     owner,
		Workspace: workspace,
		StorageID: app.CreateLocalStorage(t, owner, workspace.Id),
	}
}

func (tm Team) CreateDatabase(t *testing.T, settings any) apiclient.Database {
	t.Helper()

	return apptest.CreateDatabase(t, tm.Owner, tm.Workspace.Id, NewDatabaseSettings(t, settings))
}

// UpdateDatabaseSettings replaces the settings of the database and returns it after the change. A
// secret the settings leave out keeps its stored value.
func (tm Team) UpdateDatabaseSettings(t *testing.T, database apiclient.Database, settings any) apiclient.Database {
	t.Helper()

	response, err := tm.Owner.Client.UpdateDatabaseWithResponse(t.Context(), database.Id, apiclient.DatabaseUpdate{
		Name:        database.Name,
		NotifierIds: database.NotifierIds,
		Settings:    NewDatabaseSettings(t, settings),
	})
	requireStatus(t, "update the database", err, http.StatusOK, response.StatusCode(), response.Body)

	return *response.JSON200
}

// CreateLimitedUser creates a limited user on the server of the saved database, whose stored secrets
// fill the ones its returned settings leave out.
func (tm Team) CreateLimitedUser(t *testing.T, database apiclient.Database) apiclient.LimitedUser {
	t.Helper()

	response, err := tm.Owner.Client.CreateLimitedUserWithResponse(t.Context(), apiclient.DatabaseSettingsTarget{
		DatabaseId:  &database.Id,
		Settings:    database.Settings,
		WorkspaceId: database.WorkspaceId,
	})
	requireStatus(t, "create a limited user", err, http.StatusCreated, response.StatusCode(), response.Body)

	return *response.JSON201
}

// EnableBackups turns on the backups of the database into the storage with the encryption, keeping
// the rest of the config as read.
func (tm Team) EnableBackups(
	t *testing.T,
	databaseID uuid.UUID,
	storageID uuid.UUID,
	encryption apiclient.BackupEncryption,
) {
	t.Helper()

	readConfig := apptest.GetBackupConfig(t, tm.Owner, databaseID)

	response, err := tm.Owner.Client.UpdateBackupConfigWithResponse(t.Context(), databaseID,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &storageID,
			Encryption:       encryption,
			Settings:         readConfig.Settings,
		})
	requireStatus(t, "update the backup config", err, http.StatusOK, response.StatusCode(), response.Body)
}

func (tm Team) StartBackup(t *testing.T, databaseID uuid.UUID) apiclient.LogicalBackup {
	t.Helper()

	response, err := tm.Owner.Client.StartLogicalBackupWithResponse(t.Context(),
		apiclient.LogicalBackupStart{DatabaseId: databaseID})
	requireStatus(t, "start the backup", err, http.StatusAccepted, response.StatusCode(), response.Body)

	return *response.JSON202
}

// WaitForBackupOutcome waits until the backup leaves IN_PROGRESS and returns it.
func (tm Team) WaitForBackupOutcome(t *testing.T, backupID uuid.UUID) apiclient.LogicalBackup {
	t.Helper()

	var finishedBackup apiclient.LogicalBackup

	testwait.Until(t, "the logical backup finishes", func(ctx context.Context) error {
		response, err := tm.Owner.Client.GetLogicalBackupWithResponse(ctx, backupID)
		if err != nil {
			return err
		}

		if response.StatusCode() != http.StatusOK {
			return fmt.Errorf("get the logical backup answered %d: %s", response.StatusCode(), response.Body)
		}

		if response.JSON200.Status == apiclient.LogicalBackupStatusINPROGRESS {
			return errBackupRunning
		}

		finishedBackup = *response.JSON200

		return nil
	})

	return finishedBackup
}

// BackUp enables the backups of the database, starts one and fails the test unless it completes.
func (tm Team) BackUp(
	t *testing.T,
	databaseID uuid.UUID,
	encryption apiclient.BackupEncryption,
) apiclient.LogicalBackup {
	t.Helper()

	tm.EnableBackups(t, databaseID, tm.StorageID, encryption)

	completedBackup := tm.WaitForBackupOutcome(t, tm.StartBackup(t, databaseID).Id)
	if completedBackup.Status != apiclient.LogicalBackupStatusCOMPLETED {
		t.Fatalf("backup reached %s instead of completing: %s",
			completedBackup.Status, getValueOrUnknown(completedBackup.FailMessage))
	}

	return completedBackup
}

// RestoreVersionCheck names a backup, the table holding its database's recorded server version and
// a reachable target whose server runs TargetVersion.
type RestoreVersionCheck struct {
	DatabaseID    uuid.UUID
	BackupID      uuid.UUID
	SettingsTable string
	Target        apiclient.RestoreTarget
	TargetVersion string
}

// RequireRestoreVersionCheck rewrites the recorded version by SQL, because the API records only
// versions the server really reported.
func (tm Team) RequireRestoreVersionCheck(t *testing.T, check RestoreVersionCheck) {
	t.Helper()

	t.Run("returns the comparison error when the backup's recorded version cannot be parsed", func(t *testing.T) {
		tm.setRecordedVersion(t, check, "not-a-version")

		response, err := tm.Owner.Client.StartRestoreWithResponse(t.Context(), apiclient.RestoreStart{
			BackupId: check.BackupID,
			Target:   check.Target,
		})

		requireStatus(t, "start the restore", err, http.StatusInternalServerError, response.StatusCode(), response.Body)
		assert.Contains(t, string(response.Body), string(apiclient.ProblemCodeInternal))
		assert.NotContains(t, string(response.Body), "lower than")
		assert.Zero(t, tm.countRestores(t, check.BackupID))
	})

	t.Run("returns 400 invalid_request when the target is older", func(t *testing.T) {
		const newerVersion = "99.0"

		tm.setRecordedVersion(t, check, newerVersion)

		response, err := tm.Owner.Client.StartRestoreWithResponse(t.Context(), apiclient.RestoreStart{
			BackupId: check.BackupID,
			Target:   check.Target,
		})

		requireStatus(t, "start the restore", err, http.StatusBadRequest, response.StatusCode(), response.Body)
		assert.Contains(t, string(response.Body), string(apiclient.ProblemCodeInvalidRequest))
		assert.Contains(t, string(response.Body), newerVersion)
		assert.Contains(t, string(response.Body), check.TargetVersion)
		assert.Zero(t, tm.countRestores(t, check.BackupID))
	})
}

// Restore restores the backup into the target and fails the test unless the restore completes.
func (tm Team) Restore(t *testing.T, backupID uuid.UUID, target apiclient.RestoreTarget) apiclient.Restore {
	t.Helper()

	response, err := tm.Owner.Client.StartRestoreWithResponse(t.Context(), apiclient.RestoreStart{
		BackupId: backupID,
		Target:   target,
	})
	requireStatus(t, "start the restore", err, http.StatusAccepted, response.StatusCode(), response.Body)

	var finishedRestore apiclient.Restore

	testwait.Until(t, "the restore finishes", func(ctx context.Context) error {
		restoreResponse, err := tm.Owner.Client.GetRestoreWithResponse(ctx, response.JSON202.Id)
		if err != nil {
			return err
		}

		if restoreResponse.StatusCode() != http.StatusOK {
			return fmt.Errorf("get the restore answered %d: %s", restoreResponse.StatusCode(), restoreResponse.Body)
		}

		if restoreResponse.JSON200.Status == apiclient.RestoreStatusINPROGRESS {
			return errRestoreRunning
		}

		finishedRestore = *restoreResponse.JSON200

		return nil
	})

	if finishedRestore.Status != apiclient.RestoreStatusCOMPLETED {
		t.Fatalf("restore reached %s instead of completing: %s",
			finishedRestore.Status, getValueOrUnknown(finishedRestore.FailMessage))
	}

	return finishedRestore
}

// RequireBackupSizeMatchesDownload pins issue #691: the download announced a length taken from the
// recorded size, so a size that disagreed with the file announced bytes the download never sent.
func (tm Team) RequireBackupSizeMatchesDownload(t *testing.T, backup apiclient.LogicalBackup) {
	t.Helper()

	tokenResponse, err := tm.Owner.Client.CreateDownloadTokenWithResponse(t.Context(),
		apiclient.DownloadTokenCreate{BackupId: backup.Id})
	requireStatus(t, "create a download token", err, http.StatusCreated, tokenResponse.StatusCode(),
		tokenResponse.Body)

	downloadResponse, err := tm.Owner.Client.DownloadLogicalBackupWithResponse(t.Context(), backup.Id,
		&apiclient.DownloadLogicalBackupParams{Token: tokenResponse.JSON201.Token},
		func(_ context.Context, request *http.Request) error {
			request.Header.Set("Accept-Encoding", "identity")
			return nil
		})
	requireStatus(t, "download the backup", err, http.StatusOK, downloadResponse.StatusCode(), downloadResponse.Body)

	downloadedBytes := int64(len(downloadResponse.Body))
	announcedLength := downloadResponse.HTTPResponse.ContentLength

	if announcedLength != downloadedBytes {
		t.Fatalf("the download announced %d bytes and delivered %d", announcedLength, downloadedBytes)
	}

	downloadedSizeMb := float64(downloadedBytes) / bytesInMegabyte
	if math.Abs(downloadedSizeMb-backup.SizeMb) > sizeToleranceMb {
		t.Fatalf("the backup records %f MB and the download delivers %f MB", backup.SizeMb, downloadedSizeMb)
	}
}

// RequireBackupFailsWithoutHanging pins issue #582: a backup whose upload fails must end FAILED with a
// message instead of hanging.
func (tm Team) RequireBackupFailsWithoutHanging(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	tm.EnableBackups(t, databaseID, tm.createMissingBucketStorage(t), apiclient.BackupEncryptionNONE)

	failedBackup := tm.WaitForBackupOutcome(t, tm.StartBackup(t, databaseID).Id)

	if failedBackup.Status != apiclient.LogicalBackupStatusFAILED {
		t.Fatalf("a backup whose upload fails reached %s instead of failing", failedBackup.Status)
	}

	if failedBackup.FailMessage == nil {
		t.Fatal("a failed backup must carry the upload error")
	}
}

// GetTimescaledbVersion reads the TimescaleDB version the backup recorded, which the API does not
// return.
func (tm Team) GetTimescaledbVersion(t *testing.T, backupID uuid.UUID) string {
	t.Helper()

	var timescaledbVersion string
	tm.App.QuerySQL(t, &timescaledbVersion,
		"SELECT COALESCE(timescaledb_version, '') FROM backups_logical WHERE id = ?", backupID)

	return timescaledbVersion
}

func (tm Team) setRecordedVersion(t *testing.T, check RestoreVersionCheck, version string) {
	t.Helper()

	tm.App.ExecuteSQL(
		t,
		"UPDATE "+check.SettingsTable+" SET version = ? WHERE database_id = ?",
		version,
		check.DatabaseID,
	)
}

func (tm Team) countRestores(t *testing.T, backupID uuid.UUID) int64 {
	t.Helper()

	var restoreCount int64
	tm.App.QuerySQL(t, &restoreCount, "SELECT count(*) FROM backups_restores WHERE backup_id = ?", backupID)

	return restoreCount
}

// createMissingBucketStorage creates an S3 storage on a reachable MinIO whose bucket does not exist,
// so a backup fails when it uploads the file rather than when it connects.
func (tm Team) createMissingBucketStorage(t *testing.T) uuid.UUID {
	t.Helper()

	minio := containers.StartMinio(t)
	endpoint := fmt.Sprintf("http://%s:%d", minio.Host, minio.Port)
	accessKey := containers.MinioRootUser
	secretKey := containers.MinioRootPassword
	region := containers.MinioRegion

	var settings apiclient.StorageSettings

	err := settings.FromS3StorageSettings(apiclient.S3StorageSettings{
		Type:                 apiclient.S3StorageSettingsTypeS3,
		S3Bucket:             "missing-bucket-" + NewUniqueSuffix(),
		S3Endpoint:           &endpoint,
		S3Region:             &region,
		S3AccessKey:          &accessKey,
		S3SecretKey:          &secretKey,
		S3Prefix:             nil,
		S3StorageClass:       nil,
		IsSkipTlsVerify:      false,
		IsVirtualHostedStyle: false,
	})
	if err != nil {
		t.Fatalf("encode the S3 storage settings: %v", err)
	}

	return apptest.CreateStorage(t, tm.Owner, tm.Workspace.Id, settings).Id
}

func requireStatus(t *testing.T, action string, err error, expectedStatus, status int, body []byte) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", action, err)
	}

	if status != expectedStatus {
		t.Fatalf("%s answered %d: %s", action, status, body)
	}
}

func getValueOrUnknown(value *string) string {
	if value == nil {
		return "unknown error"
	}

	return *value
}
