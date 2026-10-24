package backuping_logical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	backups_core_enums "databasus-backend/internal/features/backups/backups/core/enums"
	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	users_enums "databasus-backend/internal/features/users/enums"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/util/cache"
	"databasus-backend/internal/util/encryption"
	util_logger "databasus-backend/internal/util/logger"
	test_utils "databasus-backend/internal/util/testing"
)

const partialLogicalArtifact = "partial logical backup"

type createStoredArtifactThenFailBackupUsecase struct{}

func (*createStoredArtifactThenFailBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	_ *backups_config_logical.LogicalBackupConfig,
	_ *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	_ func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	_, err := fileStore.WriteFile(
		ctx,
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		strings.NewReader(partialLogicalArtifact),
	)
	if err != nil {
		return nil, err
	}

	return nil, errors.New("database dump failed after producing output")
}

func Test_BackupExecuted_NotificationSent(t *testing.T) {
	cache.GetStore().Clear(t.Context())
	user := users_testing.CreateTestUser(t.Context(), users_enums.UserRoleAdmin)
	router := CreateTestRouter()
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)
	backups_config_logical.EnableBackupsForTestDatabase(t.Context(), database.ID, storage)

	defer func() {
		// cleanup backups first
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond) // Wait for cascading deletes
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	t.Run("BackupFailed_FailNotificationSent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuper := CreateTestBackuper()
		backuper.notificationSender = mockNotificationSender
		backuper.createBackupUseCase = &CreateFailedBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			FileName:   "notification-" + uuid.New().String(),
			Status:     backups_core_logical.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err := getBackupRepository().Save(backup)
		assert.NoError(t, err)

		// Set up expectations
		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.MatchedBy(func(notification notifier_models.Notification) bool {
				return notification.Type == notifier_models.NotificationTypeBackupFailed &&
					strings.Contains(notification.Heading, "❌ Backup failed") &&
					strings.Contains(notification.Message, "backup failed")
			}),
		).Once()

		backuper.MakeBackup(t.Context(), backup.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_SuccessNotificationSent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuper := CreateTestBackuper()
		backuper.notificationSender = mockNotificationSender
		backuper.createBackupUseCase = &CreateSuccessBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			FileName:   "notification-" + uuid.New().String(),
			Status:     backups_core_logical.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err := getBackupRepository().Save(backup)
		assert.NoError(t, err)

		// Set up expectations
		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.MatchedBy(func(notification notifier_models.Notification) bool {
				return notification.Type == notifier_models.NotificationTypeBackupSuccess &&
					strings.Contains(notification.Heading, "✅ Backup completed") &&
					strings.Contains(notification.Message, "Backup completed successfully")
			}),
		).Once()

		backuper.MakeBackup(t.Context(), backup.ID, true)

		// Verify all expectations were met
		mockNotificationSender.AssertExpectations(t)
	})

	t.Run("BackupSuccess_VerifyNotificationContent", func(t *testing.T) {
		mockNotificationSender := &MockNotificationSender{}
		backuper := CreateTestBackuper()
		backuper.notificationSender = mockNotificationSender
		backuper.createBackupUseCase = &CreateSuccessBackupUsecase{}

		// Create a backup record directly that will be looked up by MakeBackup
		backup := &backups_core_logical.LogicalBackup{
			DatabaseID: database.ID,
			StorageID:  storage.ID,
			FileName:   "notification-" + uuid.New().String(),
			Status:     backups_core_logical.BackupStatusInProgress,
			CreatedAt:  time.Now().UTC(),
		}
		err := getBackupRepository().Save(backup)
		assert.NoError(t, err)

		// capture arguments
		var capturedNotifier *notifiers.Notifier
		var capturedNotification notifier_models.Notification

		mockNotificationSender.On("SendNotification",
			mock.Anything,
			mock.AnythingOfType("notifier_models.Notification"),
		).Run(func(args mock.Arguments) {
			capturedNotifier = args.Get(0).(*notifiers.Notifier)
			capturedNotification = args.Get(1).(notifier_models.Notification)
		}).Once()

		backuper.MakeBackup(t.Context(), backup.ID, true)

		// Verify expectations were met
		mockNotificationSender.AssertExpectations(t)

		// Additional detailed assertions
		assert.Equal(t, notifier_models.NotificationTypeBackupSuccess, capturedNotification.Type)
		assert.Contains(t, capturedNotification.Heading, "✅ Backup completed")
		assert.Contains(t, capturedNotification.Heading, database.Name)
		assert.Contains(t, capturedNotification.Message, "Backup completed successfully")
		assert.Contains(t, capturedNotification.Message, "10.00 MB")
		assert.Equal(t, notifier.ID, capturedNotifier.ID)
	})
}

func Test_MakeBackup_WhenCallerContextCarriesRequestID_FinishLineIsAttributed(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Attribution Test Workspace")

	capturingHandler := &requestIDCapturingHandler{requestIDByMessage: map[string]string{}}

	mockNotificationSender := &MockNotificationSender{}
	mockNotificationSender.On("SendNotification", mock.Anything, mock.Anything).Maybe()

	backuper := CreateTestBackuper()
	backuper.notificationSender = mockNotificationSender
	backuper.createBackupUseCase = &CreateSuccessBackupUsecase{}
	backuper.logger = slog.New(capturingHandler)

	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)

	const requestID = "request-id-under-test"

	backuper.MakeBackup(util_logger.ContextWithRequestID(t.Context(), requestID), backup.ID, true)

	capturedRequestID, isLogged := capturingHandler.GetRequestIDForMessagePrefix("logical backup finished")

	assert.True(t, isLogged, "the backup finish line was never logged")
	assert.Equal(t, requestID, capturedRequestID)
}

func Test_MakeBackup_WhenUsecaseStoresArtifactThenFails_ArtifactRemoved(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Failed Backup Artifact Workspace")
	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)
	backup.FileName = "failed-logical-" + backup.ID.String()
	require.NoError(t, getBackupRepository().Save(backup))

	notificationSender := &MockNotificationSender{}
	notificationSender.On("SendNotification", mock.Anything, mock.Anything).Maybe()

	backuper := CreateTestBackuperWithUseCase(&createStoredArtifactThenFailBackupUsecase{})
	backuper.notificationSender = notificationSender
	backuper.MakeBackup(t.Context(), backup.ID, true)

	persistedBackup, err := getBackupRepository().FindByID(backup.ID)
	require.NoError(t, err)
	assert.Equal(t, backups_core_logical.BackupStatusFailed, persistedBackup.Status)

	require.NoError(t, storages.DrainStorageFileDeletions(t.Context(),
		storage_files.StoredFileReference{StorageID: fixture.Storage.ID, FileName: backup.FileName},
	))

	_, err = fixture.Storage.GetFile(
		t.Context(),
		encryption.GetFieldEncryptor(),
		util_logger.GetLogger(),
		backup.FileName,
	)
	assert.Error(t, err, "a dump that failed after producing output must not leave its artifact behind")
}

// A record logged on the detached execution context is indistinguishable from one that never
// carried a request at all, which is what this pins.
type requestIDCapturingHandler struct {
	mutex              sync.Mutex
	requestIDByMessage map[string]string
}

func (h *requestIDCapturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *requestIDCapturingHandler) Handle(ctx context.Context, record slog.Record) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.requestIDByMessage[record.Message] = util_logger.GetRequestID(ctx)

	return nil
}

func (h *requestIDCapturingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *requestIDCapturingHandler) WithGroup(string) slog.Handler { return h }

func (h *requestIDCapturingHandler) GetRequestIDForMessagePrefix(prefix string) (string, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for message, requestID := range h.requestIDByMessage {
		if strings.HasPrefix(message, prefix) {
			return requestID, true
		}
	}

	return "", false
}

// Cancellation and metadata failure are backuper concerns, not engine concerns:
// every engine reaches storage through the same file store, so the discard logic
// these cover is the same code for MySQL, MariaDB, MongoDB and PostgreSQL.
// createStoredArtifactThenBlockBackupUsecase stores an artifact, reports that it started, and holds
// the backup open until its context ends, the way a real engine sits in a running dump when the
// user cancels. It then returns errAfterCancel, or the cancellation an engine reports by default.
type createStoredArtifactThenBlockBackupUsecase struct {
	startedBackup  chan struct{}
	errAfterCancel error
}

func (uc *createStoredArtifactThenBlockBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	_ *backups_config_logical.LogicalBackupConfig,
	_ *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	_ func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	if _, err := fileStore.WriteFile(
		ctx,
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		strings.NewReader(partialLogicalArtifact),
	); err != nil {
		return nil, err
	}

	close(uc.startedBackup)
	<-ctx.Done()

	if uc.errAfterCancel != nil {
		return nil, uc.errAfterCancel
	}

	return nil, fmt.Errorf("backup cancelled: %w", context.Canceled)
}

type returnErrorBackupUsecase struct {
	backupErr error
}

func (uc *returnErrorBackupUsecase) Execute(
	_ context.Context,
	_ *backups_core_logical.LogicalBackup,
	_ *backups_config_logical.LogicalBackupConfig,
	_ *databases.Database,
	_ backups_core_logical.BackupFileStore,
	_ func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	return nil, uc.backupErr
}

type createUnusableMetadataBackupUsecase struct{}

func (*createUnusableMetadataBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	_ *backups_config_logical.LogicalBackupConfig,
	_ *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	_ func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	receipt, err := fileStore.WriteFile(
		ctx,
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		strings.NewReader(partialLogicalArtifact),
	)
	if err != nil {
		return nil, err
	}

	// Encrypted metadata with no salt or IV cannot be restored, so Validate refuses it.
	return &backups_core_logical.BackupArtifacts{
		Metadata: &backups_core_logical.BackupMetadata{Encryption: backups_core_enums.BackupEncryptionEncrypted},
		Receipts: []storage_files.WriteReceipt{receipt},
	}, nil
}

func Test_MakeBackup_WhenCancelledAfterStoringArtifact_ArtifactRemoved(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Cancelled Backup Artifact Workspace")
	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)
	notificationSender := createRecordingNotificationSender()

	backuper := CreateTestBackuperWithUseCase(&createStoredArtifactThenBlockBackupUsecase{
		startedBackup: make(chan struct{}),
	})
	backuper.notificationSender = notificationSender

	makeBackupCancellingThroughRegistry(t, backuper, backup.ID)

	assertBackupCancelledSilently(t, fixture, backup.ID, notificationSender)
	assertBackupFilesRemoved(t, fixture, backup.FileName)
}

func Test_MakeBackup_WhenCancelledAndStorageThenFails_BackupCancelledWithoutFailureNotification(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Cancelled Then Storage Failed Workspace")
	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)
	notificationSender := createRecordingNotificationSender()

	backuper := CreateTestBackuperWithUseCase(&createStoredArtifactThenBlockBackupUsecase{
		startedBackup:  make(chan struct{}),
		errAfterCancel: fmt.Errorf("save to storage: %w", test_utils.GetNetworkTimeoutError(t)),
	})
	backuper.notificationSender = notificationSender

	makeBackupCancellingThroughRegistry(t, backuper, backup.ID)

	assertBackupCancelledSilently(t, fixture, backup.ID, notificationSender)
}

// Test_MakeBackup_WhenStorageTimesOut_BackupFailedAndReported_RegressionForIssue816 pins issue
// #816: a storage upload that timed out was recorded as cancelled, so no failure notification
// went out.
func Test_MakeBackup_WhenStorageTimesOut_BackupFailedAndReported_RegressionForIssue816(t *testing.T) {
	storageErr := fmt.Errorf("save to storage: failed to upload file to S3: %w", test_utils.GetNetworkTimeoutError(t))

	assertBackupFailsAndIsReported(t, storageErr, storageErr.Error())
}

func Test_MakeBackup_WhenUsecaseErrorWrapsCanceledWithoutCancelRequest_BackupFailedAndReported(t *testing.T) {
	storageErr := fmt.Errorf("save to storage: upload aborted: %w", context.Canceled)

	assertBackupFailsAndIsReported(t, storageErr, storageErr.Error())
}

func Test_MakeBackup_WhenBackupExceedsTimeLimit_BackupFailedAndReported(t *testing.T) {
	assertBackupFailsAndIsReported(t,
		backups_core_logical.ErrBackupTimeout,
		"backup exceeded the time limit of 23 hours",
	)
}

func Test_MakeBackup_WhenApplicationShutsDown_BackupFailedNotCancelled(t *testing.T) {
	assertBackupFailsAndIsReported(t,
		backups_core_logical.ErrBackupShutdown,
		"backup interrupted by application shutdown",
	)
}

func assertBackupFailsAndIsReported(t *testing.T, backupErr error, expectedFailMessage string) {
	t.Helper()

	fixture := CreateBackupTestFixture(t, "Failed Backup Outcome Workspace")
	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)

	notificationSender := &MockNotificationSender{}
	notificationSender.On("SendNotification",
		mock.Anything,
		mock.MatchedBy(func(notification notifier_models.Notification) bool {
			return notification.Type == notifier_models.NotificationTypeBackupFailed &&
				strings.Contains(notification.Message, expectedFailMessage)
		}),
	).Once()

	backuper := CreateTestBackuperWithUseCase(&returnErrorBackupUsecase{backupErr})
	backuper.notificationSender = notificationSender
	backuper.MakeBackup(t.Context(), backup.ID, true)

	persistedBackup, err := getBackupRepository().FindByID(backup.ID)
	require.NoError(t, err)
	assert.Equal(t, backups_core_logical.BackupStatusFailed, persistedBackup.Status)
	require.NotNil(t, persistedBackup.FailMessage)
	assert.Equal(t, expectedFailMessage, *persistedBackup.FailMessage)

	persistedDatabase, err := databases.GetDatabaseService().GetDatabaseByID(fixture.Database.ID)
	require.NoError(t, err)
	require.NotNil(t, persistedDatabase.LastBackupErrorMessage, "a failed backup must mark its database as failing")
	assert.Equal(t, expectedFailMessage, *persistedDatabase.LastBackupErrorMessage)

	notificationSender.AssertExpectations(t)
}

func createRecordingNotificationSender() *MockNotificationSender {
	notificationSender := &MockNotificationSender{}
	notificationSender.On("SendNotification", mock.Anything, mock.Anything).Maybe()

	return notificationSender
}

// makeBackupCancellingThroughRegistry runs MakeBackup and, once the use case has started, cancels
// the backup the way the cancel endpoint does: through the task cancellation registry.
func makeBackupCancellingThroughRegistry(t *testing.T, backuper *Backuper, backupID uuid.UUID) {
	t.Helper()

	blockingUsecase, isBlockingUsecase := backuper.createBackupUseCase.(*createStoredArtifactThenBlockBackupUsecase)
	require.True(t, isBlockingUsecase, "cancelling through the registry needs a use case that waits for it")

	finishedBackup := make(chan struct{})

	go func() {
		defer close(finishedBackup)
		backuper.MakeBackup(t.Context(), backupID, true)
	}()

	select {
	case <-blockingUsecase.startedBackup:
	case <-time.After(30 * time.Second):
		t.Fatal("the backup use case never started")
	}

	getTaskCancellationRegistry().CancelRegisteredTask(backupID)

	select {
	case <-finishedBackup:
	case <-time.After(30 * time.Second):
		t.Fatal("the backup did not finish after it was cancelled")
	}
}

func assertBackupCancelledSilently(
	t *testing.T,
	fixture *BackupTestFixture,
	backupID uuid.UUID,
	notificationSender *MockNotificationSender,
) {
	t.Helper()

	persistedBackup, err := getBackupRepository().FindByID(backupID)
	require.NoError(t, err)
	assert.Equal(t, backups_core_logical.BackupStatusCanceled, persistedBackup.Status)

	notificationSender.AssertNotCalled(t, "SendNotification", mock.Anything, mock.Anything)

	persistedDatabase, err := databases.GetDatabaseService().GetDatabaseByID(fixture.Database.ID)
	require.NoError(t, err)
	assert.Nil(t, persistedDatabase.LastBackupErrorMessage, "a cancelled backup must not mark its database as failing")
}

func Test_MakeBackup_WhenMetadataIsUnusable_BackupFailsAndArtifactRemoved(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Unusable Metadata Workspace")
	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)

	notificationSender := &MockNotificationSender{}
	notificationSender.On("SendNotification", mock.Anything, mock.Anything).Maybe()

	backuper := CreateTestBackuperWithUseCase(&createUnusableMetadataBackupUsecase{})
	backuper.notificationSender = notificationSender
	backuper.MakeBackup(t.Context(), backup.ID, true)

	persistedBackup, err := getBackupRepository().FindByID(backup.ID)
	require.NoError(t, err)
	assert.Equal(t, backups_core_logical.BackupStatusFailed, persistedBackup.Status,
		"a backup whose metadata cannot be restored from must not be published")

	assertBackupFilesRemoved(t, fixture, backup.FileName)
}

// A receipt goes stale when something claims the file between the upload and the
// publishing transaction. Bumping the generation by hand reproduces that without a
// second goroutine racing the test.
type createStaleReceiptBackupUsecase struct{}

func (*createStaleReceiptBackupUsecase) Execute(
	ctx context.Context,
	backup *backups_core_logical.LogicalBackup,
	_ *backups_config_logical.LogicalBackupConfig,
	_ *databases.Database,
	fileStore backups_core_logical.BackupFileStore,
	_ func(completedMBs float64),
) (*backups_core_logical.BackupArtifacts, error) {
	receipt, err := fileStore.WriteFile(
		ctx,
		storage_files.StoredFileReference{StorageID: backup.StorageID, FileName: backup.FileName},
		strings.NewReader(partialLogicalArtifact),
	)
	if err != nil {
		return nil, err
	}

	receipt.Generation++

	return &backups_core_logical.BackupArtifacts{
		Metadata: &backups_core_logical.BackupMetadata{Encryption: backups_core_enums.BackupEncryptionNone},
		Receipts: []storage_files.WriteReceipt{receipt},
	}, nil
}

func Test_MakeBackup_WhenPublicationFails_EveryFileOfTheAttemptIsRemoved(t *testing.T) {
	fixture := CreateBackupTestFixture(t, "Failed Publication Workspace")
	backup := SeedInProgressTestBackup(t, fixture.Database.ID, fixture.Storage.ID)

	notificationSender := &MockNotificationSender{}
	notificationSender.On("SendNotification", mock.Anything, mock.Anything).Maybe()

	backuper := CreateTestBackuperWithUseCase(&createStaleReceiptBackupUsecase{})
	backuper.notificationSender = notificationSender
	backuper.MakeBackup(t.Context(), backup.ID, true)

	persistedBackup, err := getBackupRepository().FindByID(backup.ID)
	require.NoError(t, err)
	assert.Equal(t, backups_core_logical.BackupStatusFailed, persistedBackup.Status,
		"a backup whose files cannot be claimed must not be published")

	// The caller names the files its scenario actually produced: asserting the
	// absence of a file nothing ever wrote would pass with no cleanup at all.
	assertBackupFilesRemoved(t, fixture, backup.FileName, backup.FileName+metadataSuffix)
}

func assertBackupFilesRemoved(t *testing.T, fixture *BackupTestFixture, fileNames ...string) {
	t.Helper()

	references := make([]storage_files.StoredFileReference, 0, len(fileNames))
	for _, name := range fileNames {
		references = append(references, storage_files.StoredFileReference{
			StorageID: fixture.Storage.ID,
			FileName:  name,
		})
	}

	require.NoError(t, storages.DrainStorageFileDeletions(t.Context(), references...))

	for _, name := range fileNames {
		_, err := fixture.Storage.GetFile(
			t.Context(), encryption.GetFieldEncryptor(), util_logger.GetLogger(), name,
		)
		assert.Error(t, err, "%s must not survive a backup that never published", name)
	}
}
