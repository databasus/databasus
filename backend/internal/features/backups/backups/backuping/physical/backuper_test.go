package backuping_physical

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	postgresql_executor "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	storages_bridge "databasus-backend/internal/features/storages"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

func Test_ResolveParentManifest_WhenChainRootFull_ReturnsRootRef(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	rootFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, walmath.LSN(0x1000000), walmath.LSN(0x2000000))
	rootFull.ManifestFileName = new("root.manifest")
	rootFull.Encryption = backups_core_enums.BackupEncryptionEncrypted
	rootFull.ManifestEncryptionSalt = new("root-salt")
	rootFull.ManifestEncryptionIV = new("root-iv")
	physical_testing.CreateTestFullBackup(t, rootFull)

	incrBackup := &physical_models.PhysicalIncrementalBackup{
		RootFullBackupID: rootFull.ID,
		DatabaseID:       prereqs.DB.ID,
		StorageID:        prereqs.Storage.ID,
	}

	parentRef, err := backuper.resolveParentManifest(incrBackup)
	require.NoError(t, err)
	assert.Equal(t, rootFull.ID, parentRef.BackupID)
	assert.Equal(t, "root.manifest", parentRef.FileName)
	assert.Equal(t, backups_core_enums.BackupEncryptionEncrypted, parentRef.Encryption)
	assert.Equal(t, "root-salt", parentRef.Salt)
	assert.Equal(t, "root-iv", parentRef.IV)
}

func Test_ResolveParentManifest_WhenParentIncrementalPresent_ReturnsParentRef(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	rootFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, walmath.LSN(0x1000000), walmath.LSN(0x2000000))
	rootFull.ManifestFileName = new("root.manifest")
	physical_testing.CreateTestFullBackup(t, rootFull)

	parentIncr := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.DB.ID, prereqs.Storage.ID, rootFull.ID, nil, 1,
		walmath.LSN(0x2000000), walmath.LSN(0x3000000))
	parentIncr.ManifestFileName = new("parent.manifest")
	parentIncr.Encryption = backups_core_enums.BackupEncryptionNone
	physical_testing.CreateTestIncrementalBackup(t, parentIncr)

	incrBackup := &physical_models.PhysicalIncrementalBackup{
		RootFullBackupID:          rootFull.ID,
		ParentIncrementalBackupID: new(parentIncr.ID),
		DatabaseID:                prereqs.DB.ID,
		StorageID:                 prereqs.Storage.ID,
	}

	parentRef, err := backuper.resolveParentManifest(incrBackup)
	require.NoError(t, err)
	assert.Equal(t, parentIncr.ID, parentRef.BackupID)
	assert.Equal(t, "parent.manifest", parentRef.FileName)
}

func Test_ResolveParentManifest_WhenParentMissingManifestFileName_ReturnsError(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	rootFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, walmath.LSN(0x1000000), walmath.LSN(0x2000000))
	rootFull.ManifestFileName = new("root.manifest")
	physical_testing.CreateTestFullBackup(t, rootFull)

	// Parent incremental with no manifest_file_name — the chain reference is unusable.
	parentIncr := physical_testing.NewTestCompletedIncrementalBackup(
		prereqs.DB.ID, prereqs.Storage.ID, rootFull.ID, nil, 1,
		walmath.LSN(0x2000000), walmath.LSN(0x3000000))
	physical_testing.CreateTestIncrementalBackup(t, parentIncr)

	incrBackup := &physical_models.PhysicalIncrementalBackup{
		RootFullBackupID:          rootFull.ID,
		ParentIncrementalBackupID: new(parentIncr.ID),
		DatabaseID:                prereqs.DB.ID,
		StorageID:                 prereqs.Storage.ID,
	}

	_, err := backuper.resolveParentManifest(incrBackup)
	require.Error(t, err)
}

func Test_ResolveParentManifest_WhenRootFullMissing_ReturnsError(t *testing.T) {
	backuper := CreateTestPhysicalBackuper(nil)

	incrBackup := &physical_models.PhysicalIncrementalBackup{
		RootFullBackupID: uuid.New(),
	}

	_, err := backuper.resolveParentManifest(incrBackup)
	require.Error(t, err)
}

func Test_PersistFullResult_WhenCompleted_CopiesCompressionManifestFieldsAndReleasesInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:                 physical_enums.PhysicalBackupStatusCompleted,
		TimelineID:             7,
		StartLSN:               walmath.LSN(0x100),
		StopLSN:                walmath.LSN(0x200),
		BackupSizeMb:           42.5,
		BackupDurationMs:       1234,
		EncryptionAlgo:         backups_core_enums.BackupEncryptionNone,
		Compression:            physical_enums.PhysicalBackupCompressionZstd,
		ManifestFileName:       "artifact.manifest",
		ManifestEncryptionSalt: "manifest-salt",
		ManifestEncryptionIV:   "manifest-iv",
		CompletedAt:            time.Now().UTC(),
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, nil))

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusCompleted, persisted.Status)
	assert.Equal(t, physical_enums.PhysicalBackupCompressionZstd, persisted.Compression)
	require.NotNil(t, persisted.ManifestFileName)
	assert.Equal(t, "artifact.manifest", *persisted.ManifestFileName)
	require.NotNil(t, persisted.ManifestEncryptionSalt)
	assert.Equal(t, "manifest-salt", *persisted.ManifestEncryptionSalt)
	require.NotNil(t, persisted.ManifestEncryptionIV)
	assert.Equal(t, "manifest-iv", *persisted.ManifestEncryptionIV)
	require.NotNil(t, persisted.CompletedAt)
	require.NotNil(t, persisted.BackupSizeMb)
	assert.InDelta(t, 42.5, *persisted.BackupSizeMb, 0.001)

	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_PersistFullResult_WhenCompletedWithMeasuredClusterSize_StoresRawSizeMb(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusCompleted,
		BackupSizeMb: 42.5,
		CompletedAt:  time.Now().UTC(),
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, new(4321.5)))

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	require.NotNil(t, persisted.RawSizeMb)
	assert.InDelta(t, 4321.5, *persisted.RawSizeMb, 0.001)
}

func Test_PersistFullResult_WhenClusterSizeUnmeasured_LeavesRawSizeMbNil(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusCompleted,
		BackupSizeMb: 42.5,
		CompletedAt:  time.Now().UTC(),
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, nil))

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusCompleted, persisted.Status,
		"a nil raw size must not downgrade the persisted status")
	assert.Nil(t, persisted.RawSizeMb)
}

func Test_PersistFullResult_WhenErrorStatus_RecordsTerminalTimeAndReleasesInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:           physical_enums.PhysicalBackupStatusError,
		ErrorReason:      new(physical_enums.PhysicalBackupErrorPgBasebackupFailed),
		ManifestFileName: "should-not-be-copied",
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, nil))

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *persisted.ErrorReason)
	assert.Nil(t, persisted.ManifestFileName, "completion-only fields must not be copied on a non-COMPLETED result")
	require.NotNil(t, persisted.CompletedAt)

	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_PersistIncrementalResult_WhenErrorStatus_RecordsTerminalTime(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)
	rootFull := seedCompletedRootFull(t, prereqs)
	incremental := seedInProgressIncr(t, prereqs, rootFull.ID)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeIncremental, incremental.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusChainBroken,
		ErrorReason: new(physical_enums.PhysicalBackupErrorTimelineSwitchDetected),
	}
	require.NoError(t, backuper.persistIncrResult(t.Context(), incremental, result))

	persistedIncremental, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incremental.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedIncremental.CompletedAt)
}

func Test_PersistIncrementalResult_WhenCompleted_CopiesFieldsAndReleasesInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	rootFull := seedCompletedRootFull(t, prereqs)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeIncremental, incrBackup.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:           physical_enums.PhysicalBackupStatusCompleted,
		TimelineID:       3,
		StartLSN:         walmath.LSN(0x300),
		StopLSN:          walmath.LSN(0x400),
		BackupSizeMb:     7.25,
		BackupDurationMs: 555,
		EncryptionAlgo:   backups_core_enums.BackupEncryptionNone,
		Compression:      physical_enums.PhysicalBackupCompressionGzip,
		ManifestFileName: "incr.manifest",
		CompletedAt:      time.Now().UTC(),
	}

	require.NoError(t, backuper.persistIncrResult(t.Context(), incrBackup, result))

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusCompleted, persisted.Status)
	assert.Equal(t, physical_enums.PhysicalBackupCompressionGzip, persisted.Compression)
	require.NotNil(t, persisted.ManifestFileName)
	assert.Equal(t, "incr.manifest", *persisted.ManifestFileName)
	require.NotNil(t, persisted.CompletedAt)

	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_FinalizeFullAsError_SetsErrorStatusReasonAndReleasesInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	backuper.finalizeFullAsError(
		fullBackup,
		physical_enums.PhysicalBackupErrorPgBasebackupFailed,
		"failed to load backup context",
	)

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *persisted.ErrorReason)

	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_FinalizeIncrementalAsError_SetsErrorStatusReasonAndReleasesInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	rootFull := seedCompletedRootFull(t, prereqs)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeIncremental, incrBackup.ID)

	backuper.finalizeIncrAsError(
		incrBackup,
		physical_enums.PhysicalBackupErrorPgBasebackupFailed,
		"transient executor failure",
	)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *persisted.ErrorReason)

	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_FinalizeIncrementalAsChainBroken_SetsChainBrokenStatusReasonAndReleasesInFlight(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	rootFull := seedCompletedRootFull(t, prereqs)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeIncremental, incrBackup.ID)

	backuper.finalizeIncrAsChainBroken(
		incrBackup,
		physical_enums.PhysicalBackupErrorParentManifestMissing,
		"parent manifest missing",
	)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	require.NotNil(t, persisted)

	assert.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, persisted.Status)
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorParentManifestMissing, *persisted.ErrorReason)

	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_ClassifyFullBackupNotification_WhenCompleted_ReturnsSuccessNotification(t *testing.T) {
	database := databases.Database{Name: "mydb"}
	fullBackup := &physical_models.PhysicalFullBackup{
		ID:     uuid.New(),
		Status: physical_enums.PhysicalBackupStatusCompleted,
	}
	result := postgresql_executor.PhysicalBackupResult{BackupSizeMb: 12.5, BackupDurationMs: 999}

	notificationType, title, message := classifyFullBackupNotification(
		t.Context(),
		database, fullBackup, result, workspaces_bridge.GetAccess())

	assert.Equal(t, backups.NotificationTriggerBackupSuccess, notificationType)
	assert.Contains(t, title, "completed")
	assert.Contains(t, title, "mydb")
	assert.Contains(t, message, fullBackup.ID.String())
}

func Test_ClassifyFullBackupNotification_WhenErrorStatus_ReturnsFailedNotification(t *testing.T) {
	database := databases.Database{Name: "mydb"}
	fullBackup := &physical_models.PhysicalFullBackup{
		ID:          uuid.New(),
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: new(physical_enums.PhysicalBackupErrorPgBasebackupFailed),
	}
	result := postgresql_executor.PhysicalBackupResult{ErrorMessage: "disk full"}

	notificationType, title, _ := classifyFullBackupNotification(
		t.Context(),
		database, fullBackup, result, workspaces_bridge.GetAccess())

	assert.Equal(t, backups.NotificationTriggerBackupFailed, notificationType)
	assert.Contains(t, title, "failed")
}

func Test_ClassifyFullBackupNotification_WhenChainBroken_ReturnsChainBrokenNotification(t *testing.T) {
	database := databases.Database{Name: "mydb"}
	fullBackup := &physical_models.PhysicalFullBackup{
		ID:     uuid.New(),
		Status: physical_enums.PhysicalBackupStatusChainBroken,
	}

	notificationType, title, _ := classifyFullBackupNotification(
		t.Context(),
		database, fullBackup, postgresql_executor.PhysicalBackupResult{}, workspaces_bridge.GetAccess())

	assert.Equal(t, backups.NotificationTriggerChainBroken, notificationType)
	assert.Contains(t, title, "chain-broken")
}

func Test_ClassifyFullBackupNotification_WhenInProgress_ReturnsEmpty(t *testing.T) {
	database := databases.Database{Name: "mydb"}
	fullBackup := &physical_models.PhysicalFullBackup{
		ID:     uuid.New(),
		Status: physical_enums.PhysicalBackupStatusInProgress,
	}

	notificationType, _, _ := classifyFullBackupNotification(
		t.Context(),
		database, fullBackup, postgresql_executor.PhysicalBackupResult{}, workspaces_bridge.GetAccess())

	assert.Empty(t, notificationType)
}

func Test_SendFullBackupNotification_WhenTypeNotInSendOn_DoesNotSend(t *testing.T) {
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)

	// Completed maps to BACKUP_SUCCESS, which is NOT in SendNotificationsOn.
	cfg := backups.PhysicalBackupConfig{
		BackupConfig: backups.BackupConfig{
			Encryption: backups.EncryptionNone,
			NotificationTriggers: []backups.NotificationTrigger{
				backups.NotificationTriggerChainBroken,
			},
		},
	}
	database := databases.Database{
		Name:        "mydb",
		NotifierIDs: []uuid.UUID{uuid.New()},
	}
	fullBackup := &physical_models.PhysicalFullBackup{
		ID:     uuid.New(),
		Status: physical_enums.PhysicalBackupStatusCompleted,
	}

	backuper.sendFullBackupNotification(
		t.Context(),
		cfg,
		database,
		fullBackup,
		postgresql_executor.PhysicalBackupResult{},
	)

	assert.Empty(t, sender.sentNotifications)
}

func Test_SendFullBackupNotification_WhenEnabled_NotifiesEveryNotifierInOneCall(t *testing.T) {
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)

	cfg := backups.PhysicalBackupConfig{
		BackupConfig: backups.BackupConfig{
			Encryption: backups.EncryptionNone,
			NotificationTriggers: []backups.NotificationTrigger{
				backups.NotificationTriggerBackupSuccess,
			},
		},
	}
	firstNotifierID, secondNotifierID := uuid.New(), uuid.New()
	database := databases.Database{
		Name:        "mydb",
		NotifierIDs: []uuid.UUID{firstNotifierID, secondNotifierID},
	}
	fullBackup := &physical_models.PhysicalFullBackup{
		ID:     uuid.New(),
		Status: physical_enums.PhysicalBackupStatusCompleted,
	}

	backuper.sendFullBackupNotification(
		t.Context(),
		cfg,
		database,
		fullBackup,
		postgresql_executor.PhysicalBackupResult{},
	)

	require.Len(t, sender.sentNotifications, 1)
	assert.ElementsMatch(t, []uuid.UUID{firstNotifierID, secondNotifierID}, sender.sentNotifications[0].NotifierIDs)
	assert.Contains(t, sender.sentNotifications[0].Notification.Heading, "completed")
	assert.Equal(t, notifiers.CategoryBackupSuccess, sender.sentNotifications[0].Notification.Category)
}

func Test_OpenBackupContext_WhenNotEncrypted_LeavesMasterKeyEmpty(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	backupCtx, ok := backuper.openBackupContext(t.Context(), logger.GetLogger(), prereqs.DB.ID)
	require.True(t, ok)
	require.NotNil(t, backupCtx)

	defer backupCtx.Close()

	assert.Empty(t, backupCtx.MasterKey)
	assert.NotNil(t, backupCtx.Config)
	assert.NotNil(t, backupCtx.Database)
	assert.NotNil(t, backupCtx.Storage)
}

func Test_OpenBackupContext_WhenEncrypted_PopulatesMasterKey(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	prereqs.Config.Encryption = backups.EncryptionEncrypted
	_, err := backups_bridge.SaveTestBackupConfig(t.Context(), prereqs.Config)
	require.NoError(t, err)

	backupCtx, ok := backuper.openBackupContext(t.Context(), logger.GetLogger(), prereqs.DB.ID)
	require.True(t, ok)
	require.NotNil(t, backupCtx)

	defer backupCtx.Close()

	assert.NotEmpty(t, backupCtx.MasterKey)
}

func Test_OpenBackupContext_WhenConfigHasNoStorageID_ReturnsNotOk(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	prereqs.Config.IsBackupsEnabled = false
	prereqs.Config.StorageID = nil
	_, err := backups_bridge.SaveTestBackupConfig(t.Context(), prereqs.Config)
	require.NoError(t, err)

	_, ok := backuper.openBackupContext(t.Context(), logger.GetLogger(), prereqs.DB.ID)
	assert.False(t, ok)
}

func Test_MakeBackup_WhenRowAbsentInBothTables_InvokesNoExecutor(t *testing.T) {
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	fullExecutor, incrExecutor := installFakeExecutors(backuper)

	backuper.MakeBackup(t.Context(), uuid.New(), false)

	assert.Equal(t, 0, fullExecutor.callCount)
	assert.Equal(t, 0, incrExecutor.callCount)
}

func Test_MakeBackup_WhenFullRowExists_InvokesFullExecutorOnly(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	fullExecutor, incrExecutor := installFakeExecutors(backuper)
	fullExecutor.result = erroredResult()

	fullBackup := seedInProgressFull(t, prereqs)

	backuper.MakeBackup(t.Context(), fullBackup.ID, false)

	assert.Equal(t, 1, fullExecutor.callCount)
	assert.Equal(t, 0, incrExecutor.callCount)

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)
}

func Test_MakeBackup_WhenIncrementalRowExists_InvokesIncrementalExecutorOnly(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	fullExecutor, incrExecutor := installFakeExecutors(backuper)
	incrExecutor.result = erroredResult()

	rootFull := seedCompletedRootFull(t, prereqs)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)

	backuper.MakeBackup(t.Context(), incrBackup.ID, false)

	assert.Equal(t, 0, fullExecutor.callCount)
	assert.Equal(t, 1, incrExecutor.callCount)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)
}

func Test_RunFullBackup_WhenExecutorReturnsGoError_FlipsToErrorAndSkipsNotification(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	fullExecutor, _ := installFakeExecutors(backuper)
	fullExecutor.err = errors.New("executor blew up")

	fullBackup := seedInProgressFull(t, prereqs)

	backuper.MakeBackup(t.Context(), fullBackup.ID, true)

	assert.Equal(t, 1, fullExecutor.callCount)

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *persisted.ErrorReason)

	assert.Empty(t, sender.sentNotifications, "a Go error returns before the notification block")
}

func Test_RunFullBackup_WhenExecutorResultErrorStatus_PersistsErrorAndSendsFailedNotification(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	fullExecutor, _ := installFakeExecutors(backuper)
	fullExecutor.result = postgresql_executor.PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  new(physical_enums.PhysicalBackupErrorPgBasebackupFailed),
		ErrorMessage: "disk full",
	}

	fullBackup := seedInProgressFull(t, prereqs)

	backuper.MakeBackup(t.Context(), fullBackup.ID, true)

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)

	require.Len(t, sender.sentNotifications, 1)
	assert.Contains(t, sender.sentNotifications[0].Notification.Heading, "failed")
	assert.Equal(t, notifiers.CategoryBackupFailed, sender.sentNotifications[0].Notification.Category)
}

func Test_RunFullBackup_WhenExecutorResultChainBroken_PersistsChainBrokenAndSendsChainBrokenNotification(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	fullExecutor, _ := installFakeExecutors(backuper)
	fullExecutor.result = postgresql_executor.PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusChainBroken,
		ErrorReason: new(physical_enums.PhysicalBackupErrorManifestCorrupted),
	}

	fullBackup := seedInProgressFull(t, prereqs)

	backuper.MakeBackup(t.Context(), fullBackup.ID, true)

	persisted, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, persisted.Status)

	require.Len(t, sender.sentNotifications, 1)
	assert.Contains(t, sender.sentNotifications[0].Notification.Heading, "chain-broken")
	assert.Equal(t, notifiers.CategoryBackupFailed, sender.sentNotifications[0].Notification.Category)
}

func Test_RunIncrementalBackup_WhenParentManifestMissing_FlipsToChainBrokenBeforeExecutor(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	_, incrExecutor := installFakeExecutors(backuper)

	// Root full with no manifest_file_name: resolveParentManifest fails before the executor runs.
	rootFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, walmath.LSN(0x1000000), walmath.LSN(0x2000000))
	physical_testing.CreateTestFullBackup(t, rootFull)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)

	backuper.MakeBackup(t.Context(), incrBackup.ID, true)

	assert.Equal(t, 0, incrExecutor.callCount, "executor must not run when the parent manifest is unresolved")

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, persisted.Status,
		"a missing parent manifest is irreversible, so the chain must break (not retry as ERROR)")
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorParentManifestMissing, *persisted.ErrorReason)

	assert.Empty(t, sender.sentNotifications,
		"pre-executor finalize paths are silent, matching the full-backup path")
}

func Test_RunIncrementalBackup_WhenExecutorResultErrorStatus_PersistsErrorAndSendsFailedNotification(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	_, incrExecutor := installFakeExecutors(backuper)
	incrExecutor.result = erroredResult()

	rootFull := seedCompletedRootFull(t, prereqs)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)

	backuper.MakeBackup(t.Context(), incrBackup.ID, true)

	assert.Equal(t, 1, incrExecutor.callCount)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status)

	require.Len(t, sender.sentNotifications, 1)
	assert.Contains(t, sender.sentNotifications[0].Notification.Heading, "INCR failed")
	assert.Equal(t, notifiers.CategoryBackupFailed, sender.sentNotifications[0].Notification.Category)
}

func Test_RunIncrementalBackup_WhenSecondConsecutiveFailure_KeepsChainExtendable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(&recordingNotificationSender{})
	_, incrExecutor := installFakeExecutors(backuper)
	incrExecutor.result = erroredResult()

	rootFull := seedCompletedRootFull(t, prereqs)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusError, 2, 2)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)

	backuper.MakeBackup(t.Context(), incrBackup.ID, true)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status,
		"the budget is three attempts, so the second failure must leave the chain extendable")
}

func Test_RunIncrementalBackup_WhenRetryBudgetExhausted_BreaksChainKeepingErrorReason(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	sender := &recordingNotificationSender{}
	backuper := CreateTestPhysicalBackuper(sender)
	_, incrExecutor := installFakeExecutors(backuper)
	incrExecutor.result = erroredResult()

	rootFull := seedCompletedRootFull(t, prereqs)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusError, 2, 3)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusError, 3, 2)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)

	backuper.MakeBackup(t.Context(), incrBackup.ID, true)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusChainBroken, persisted.Status)
	require.NotNil(t, persisted.ErrorReason)
	assert.Equal(t, physical_enums.PhysicalBackupErrorPgBasebackupFailed, *persisted.ErrorReason,
		"the operator still needs to see what actually failed, not just that the chain closed")

	require.Len(t, sender.sentNotifications, 1)
	assert.Contains(t, sender.sentNotifications[0].Notification.Heading, "chain-broken",
		"the row and the notification must agree on the verdict")
}

func Test_RunIncrementalBackup_WhenCompletedBackupSeparatesFailures_KeepsChainExtendable(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(&recordingNotificationSender{})
	_, incrExecutor := installFakeExecutors(backuper)
	incrExecutor.result = erroredResult()

	rootFull := seedCompletedRootFull(t, prereqs)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusError, 2, 4)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusCompleted, 3, 3)
	seedIncrWithStatusAndAge(t, prereqs, rootFull.ID, physical_enums.PhysicalBackupStatusError, 4, 2)
	incrBackup := seedInProgressIncr(t, prereqs, rootFull.ID)

	backuper.MakeBackup(t.Context(), incrBackup.ID, true)

	persisted, err := physical_repositories.GetIncrementalBackupRepository().FindByID(incrBackup.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, persisted.Status,
		"a COMPLETED backup between failures ends the run, so the budget is not spent")
}

func Test_ReleaseOwned_WhenForeignBackupHoldsClaim_LeavesItIntact(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	inFlightRepo := physical_repositories.GetInFlightBackupRepository()

	liveBackupID := uuid.New()
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, liveBackupID)

	staleBackupID := uuid.New()
	require.NoError(t, inFlightRepo.ReleaseOwned(prereqs.DB.ID, staleBackupID))

	claim, err := inFlightRepo.FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, claim, "a stale backup's release must not delete the live claim")
	assert.Equal(t, liveBackupID, claim.BackupID)

	require.NoError(t, inFlightRepo.ReleaseOwned(prereqs.DB.ID, liveBackupID))
	assertInFlightReleased(t, prereqs.DB.ID)
}

func Test_PersistFullResult_WhenRowNoLongerInProgress_DoesNotResurrect(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)
	fullRepo := physical_repositories.GetFullBackupRepository()
	inFlightRepo := physical_repositories.GetInFlightBackupRepository()

	full := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, full.ID)

	// A restart recovery already failed this backup, released its claim, and a
	// fresh backup took the database's in-flight slot.
	require.NoError(t, fullRepo.UpdateStatus(full.ID, physical_enums.PhysicalBackupStatusError, nil))
	require.NoError(t, inFlightRepo.ReleaseOwned(prereqs.DB.ID, full.ID))

	newBackupID := uuid.New()
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, newBackupID)

	err := backuper.persistFullResult(t.Context(), full, postgresql_executor.PhysicalBackupResult{
		Status: physical_enums.PhysicalBackupStatusCompleted,
	}, nil)
	require.NoError(t, err)

	reloaded, err := fullRepo.FindByID(full.ID)
	require.NoError(t, err)
	assert.Equal(t, physical_enums.PhysicalBackupStatusError, reloaded.Status,
		"a superseded backup must not be resurrected to COMPLETED")

	claim, err := inFlightRepo.FindByDatabaseID(prereqs.DB.ID)
	require.NoError(t, err)
	require.NotNil(t, claim, "the newer backup's claim must survive")
	assert.Equal(t, newBackupID, claim.BackupID)
}

func installFakeExecutors(backuper *PhysicalBackuper) (*fakeFullExecutor, *fakeIncrementalExecutor) {
	fullExecutor := &fakeFullExecutor{}
	incrExecutor := &fakeIncrementalExecutor{}
	backuper.fullExecutor = fullExecutor
	backuper.incrExecutor = incrExecutor

	return fullExecutor, incrExecutor
}

// erroredResult is the canned ERROR outcome the fake executors return when a
// test only cares that a non-COMPLETED result is persisted and routed.
func erroredResult() postgresql_executor.PhysicalBackupResult {
	return postgresql_executor.PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusError,
		ErrorReason: new(physical_enums.PhysicalBackupErrorPgBasebackupFailed),
	}
}

func seedInProgressFull(t *testing.T, prereqs *backupPrereqs) *physical_models.PhysicalFullBackup {
	t.Helper()

	return physical_testing.CreateTestFullBackup(t, &physical_models.PhysicalFullBackup{
		DatabaseID: prereqs.DB.ID,
		StorageID:  prereqs.Storage.ID,
		TimelineID: 1,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
		Encryption: backups_core_enums.BackupEncryptionNone,
		CreatedAt:  time.Now().UTC(),
	})
}

func seedCompletedRootFull(t *testing.T, prereqs *backupPrereqs) *physical_models.PhysicalFullBackup {
	t.Helper()

	rootFull := physical_testing.NewTestCompletedFullBackup(
		prereqs.DB.ID, prereqs.Storage.ID, 1, walmath.LSN(0x1000000), walmath.LSN(0x2000000))
	rootFull.ManifestFileName = new("root.manifest")

	return physical_testing.CreateTestFullBackup(t, rootFull)
}

func seedInProgressIncr(
	t *testing.T,
	prereqs *backupPrereqs,
	rootFullBackupID uuid.UUID,
) *physical_models.PhysicalIncrementalBackup {
	t.Helper()

	return physical_testing.CreateTestIncrementalBackup(t, &physical_models.PhysicalIncrementalBackup{
		DatabaseID:       prereqs.DB.ID,
		StorageID:        prereqs.Storage.ID,
		RootFullBackupID: rootFullBackupID,
		TimelineID:       1,
		Status:           physical_enums.PhysicalBackupStatusInProgress,
		Encryption:       backups_core_enums.BackupEncryptionNone,
		CreatedAt:        time.Now().UTC(),
	})
}

func claimInFlight(
	t *testing.T,
	databaseID uuid.UUID,
	backupType physical_enums.PhysicalBackupType,
	backupID uuid.UUID,
) {
	t.Helper()

	claimed, err := physical_repositories.GetInFlightBackupRepository().Claim(
		storage.GetDb(), physical_repositories.ClaimSpec{
			DatabaseID: databaseID,
			BackupType: backupType,
			BackupID:   backupID,
		})
	require.NoError(t, err)
	require.True(t, claimed)
}

func assertInFlightReleased(t *testing.T, databaseID uuid.UUID) {
	t.Helper()

	inFlight, err := physical_repositories.GetInFlightBackupRepository().FindByDatabaseID(databaseID)
	require.NoError(t, err)
	assert.Nil(t, inFlight, "in-flight claim must be released")
}

func Test_PersistFullResult_WhenCompleted_KeepsTheFilesTheAttemptWrote(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	fileName := "physical-completed-" + fullBackup.ID.String()

	receipt, err := storages_bridge.GetModule().WriteFile(
		t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
		strings.NewReader("physical artifact"),
	)
	require.NoError(t, err)

	result := postgresql_executor.PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusCompleted,
		FileName:    fileName,
		Receipts:    []storages.WriteReceipt{receipt},
		CompletedAt: time.Now().UTC(),
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, nil))

	require.NoError(t, storages_bridge.DrainStorageFileDeletions(t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
	))

	reader, err := storages_bridge.OpenTestFile(
		t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
	)
	require.NoError(t, err, "a published backup keeps its artifact")
	require.NoError(t, reader.Close())
}

func Test_PersistFullResult_WhenCancelled_HandsTheFilesBackToCleanup(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	fileName := "physical-cancelled-" + fullBackup.ID.String()
	manifestName := fileName + ".manifest"

	for _, name := range []string{fileName, manifestName} {
		_, err := storages_bridge.GetModule().WriteFile(
			t.Context(),
			storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: name},
			strings.NewReader("partial physical artifact"),
		)
		require.NoError(t, err)
	}

	result := postgresql_executor.PhysicalBackupResult{
		Status:           physical_enums.PhysicalBackupStatusCanceled,
		FileName:         fileName,
		ManifestFileName: manifestName,
		CompletedAt:      time.Now().UTC(),
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, nil))

	require.NoError(t, storages_bridge.DrainStorageFileDeletions(t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: manifestName},
	))

	for _, name := range []string{fileName, manifestName} {
		_, err := storages_bridge.OpenTestFile(
			t.Context(),
			storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: name},
		)
		assert.Error(t, err, "%s must not survive a cancelled backup", name)
	}
}

func Test_PersistFullResult_WhenFailed_HandsTheFilesBackToCleanup(t *testing.T) {
	prereqs := seedBackupPrereqs(t)
	backuper := CreateTestPhysicalBackuper(nil)

	fullBackup := seedInProgressFull(t, prereqs)
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	fileName := "physical-failed-" + fullBackup.ID.String()

	_, err := storages_bridge.GetModule().WriteFile(
		t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
		strings.NewReader("partial physical artifact"),
	)
	require.NoError(t, err)

	reason := physical_enums.PhysicalBackupErrorPgBasebackupFailed
	result := postgresql_executor.PhysicalBackupResult{
		Status:       physical_enums.PhysicalBackupStatusError,
		ErrorReason:  &reason,
		ErrorMessage: "pg_basebackup failed after producing output",
		FileName:     fileName,
		CompletedAt:  time.Now().UTC(),
	}

	require.NoError(t, backuper.persistFullResult(t.Context(), fullBackup, result, nil))

	require.NoError(t, storages_bridge.DrainStorageFileDeletions(t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
	))

	_, err = storages_bridge.OpenTestFile(
		t.Context(),
		storages.StoredFileReference{StorageID: prereqs.Storage.ID, FileName: fileName},
	)
	assert.Error(t, err, "a failed physical backup must not leave its artifact behind")
}

func TestPhysicalBackupRecordsLastBackupTime(t *testing.T) {
	t.Run("shows the start of a full backup when it completes", func(t *testing.T) {
		prereqs := seedBackupPrereqs(t)
		fullBackupStartedAt := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)

		completeFullBackup(t, prereqs, fullBackupStartedAt)

		lastBackupTime := getLastBackupTime(t, prereqs.DB.ID)
		assert.True(t, fullBackupStartedAt.Equal(lastBackupTime), lastBackupTime)
	})

	t.Run("shows the time a WAL segment was received after the last full backup", func(t *testing.T) {
		prereqs := seedBackupPrereqs(t)
		completeFullBackup(t, prereqs, time.Now().UTC().Add(-2*time.Hour))

		walSegmentReceivedAt := publishWalSegment(t, prereqs)

		lastBackupTime := getLastBackupTime(t, prereqs.DB.ID)
		assert.True(t, walSegmentReceivedAt.Equal(lastBackupTime), lastBackupTime)
	})

	t.Run("keeps the WAL segment time when a full backup that started before it completes later", func(t *testing.T) {
		prereqs := seedBackupPrereqs(t)
		fullBackupStartedAt := time.Now().UTC().Add(-time.Hour)
		walSegmentReceivedAt := publishWalSegment(t, prereqs)

		completeFullBackup(t, prereqs, fullBackupStartedAt)

		lastBackupTime := getLastBackupTime(t, prereqs.DB.ID)
		assert.True(t, walSegmentReceivedAt.Equal(lastBackupTime), lastBackupTime)
	})
}

func completeFullBackup(t *testing.T, prereqs *backupPrereqs, startedAt time.Time) {
	t.Helper()

	fullBackup := physical_testing.CreateTestFullBackup(t, &physical_models.PhysicalFullBackup{
		DatabaseID: prereqs.DB.ID,
		StorageID:  prereqs.Storage.ID,
		TimelineID: 1,
		Status:     physical_enums.PhysicalBackupStatusInProgress,
		Encryption: backups_core_enums.BackupEncryptionNone,
		CreatedAt:  startedAt,
	})
	claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

	result := postgresql_executor.PhysicalBackupResult{
		Status:      physical_enums.PhysicalBackupStatusCompleted,
		CompletedAt: time.Now().UTC(),
	}

	require.NoError(t, CreateTestPhysicalBackuper(nil).persistFullResult(t.Context(), fullBackup, result, nil))
}

// publishWalSegment runs one segment through the uploader the WAL streamer uses and returns the
// time its row says the segment was received.
func publishWalSegment(t *testing.T, prereqs *backupPrereqs) time.Time {
	t.Helper()

	const walSegmentSizeBytes = int64(16 * 1024 * 1024)
	const walSegmentName = "000000010000000000000005"

	watchDir := t.TempDir()
	segmentPath := filepath.Join(watchDir, walSegmentName)
	segmentFile, err := os.Create(segmentPath)
	require.NoError(t, err)
	require.NoError(t, segmentFile.Truncate(walSegmentSizeBytes))
	require.NoError(t, segmentFile.Close())

	uploader := postgresql_executor.NewWalUploader(postgresql_executor.WalUploadDeps{
		DatabaseID:          prereqs.DB.ID,
		StorageID:           prereqs.Storage.ID,
		FileStore:           storages_bridge.GetModule(),
		BackupStatus:        databases_bridge.GetModule(),
		Encryption:          backups_core_enums.BackupEncryptionNone,
		MasterKey:           nil,
		WalSegmentRepo:      physical_repositories.GetWalSegmentRepository(),
		WalSegmentSizeBytes: walSegmentSizeBytes,
		Logger:              logger.GetLogger(),
		WatchDir:            watchDir,
		OnGapDetected:       nil,
	})
	require.NoError(t, uploader.ProcessSegment(t.Context(), segmentPath, walSegmentName))

	segment, err := physical_repositories.GetWalSegmentRepository().FindByChainKey(
		prereqs.DB.ID, 1, walmath.LSN(5*uint64(walSegmentSizeBytes)))
	require.NoError(t, err)
	require.NotNil(t, segment)
	require.NotNil(t, segment.FileName, "the segment must be published")

	return segment.ReceivedAt
}

func getLastBackupTime(t *testing.T, databaseID uuid.UUID) time.Time {
	t.Helper()

	foundDatabases, err := databases_bridge.GetModule().GetDatabases(t.Context(), []uuid.UUID{databaseID})
	require.NoError(t, err)
	require.Len(t, foundDatabases, 1)
	require.NotNil(t, foundDatabases[0].LastBackupTime)

	return *foundDatabases[0].LastBackupTime
}

func TestPersistFullResultRacingDatabaseDeletion(t *testing.T) {
	t.Run("deletes the database when the deletion locks the database before the completed backup is saved",
		func(t *testing.T) {
			prereqs := seedBackupPrereqs(t)
			backuper := CreateTestPhysicalBackuper(nil)

			fullBackup := seedInProgressFull(t, prereqs)
			claimInFlight(t, prereqs.DB.ID, physical_enums.PhysicalBackupTypeFull, fullBackup.ID)

			deletion := databases_bridge.BeginTestDatabaseDeletion(t, prereqs.DB.ID)

			persistErr := make(chan error, 1)
			go func() {
				persistErr <- backuper.persistFullResult(t.Context(), fullBackup, postgresql_executor.PhysicalBackupResult{
					Status:      physical_enums.PhysicalBackupStatusCompleted,
					CompletedAt: time.Now().UTC(),
				}, nil)
			}()

			deletion.WaitForBlockedDatabaseWrite(t)

			require.NoError(t, deletion.Finish())
			require.ErrorIs(t, <-persistErr, databases.ErrDatabaseNotFound)

			persistedBackup, err := physical_repositories.GetFullBackupRepository().FindByID(fullBackup.ID)
			require.NoError(t, err)
			require.Nil(t, persistedBackup)
		})
}
