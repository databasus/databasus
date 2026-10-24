//go:build testhelpers

package storages

import (
	"context"

	"github.com/google/uuid"

	storage_files "databasus-backend/internal/features/storages/files"
	local_storage "databasus-backend/internal/features/storages/models/local"
	s3_storage "databasus-backend/internal/features/storages/models/s3"
	"databasus-backend/internal/util/encryption"
)

// A test can assert absence right after the operation that caused it instead of
// waiting on the production ticker.
func DrainStorageFileDeletions(
	ctx context.Context,
	references ...storage_files.StoredFileReference,
) error {
	dependencies := getStorageFileDependencies()
	dependencies.Timings = storage_files.TimingsForTest()

	return storage_files.NewDeletionWorker(getStorageFileStore(), dependencies).DrainForTest(ctx, references...)
}

func SetStorageDatabaseCountersForTest(counters ...StorageDatabaseCounter) {
	getStorageService().storageDatabaseCounters = counters
}

func SetStorageBackupCountersForTest(counters ...StorageBackupCounter) {
	getStorageService().storageBackupCounters = counters
}

func CreateTestStorage(workspaceID uuid.UUID) *Storage {
	storage := &Storage{
		WorkspaceID:  workspaceID,
		Type:         StorageTypeLocal,
		Name:         "Test Storage " + uuid.New().String(),
		LocalStorage: &local_storage.LocalStorage{},
	}

	storage, err := getStorageRepository().Save(storage)
	if err != nil {
		panic(err)
	}

	return storage
}

func CreateTestFlakyS3Storage(workspaceID uuid.UUID, endpoint string) *Storage {
	encryptor := encryption.GetFieldEncryptor()

	accessKey, err := encryptor.Encrypt("issue-582-access")
	if err != nil {
		panic(err)
	}

	secretKey, err := encryptor.Encrypt("issue-582-secret")
	if err != nil {
		panic(err)
	}

	storage := &Storage{
		WorkspaceID: workspaceID,
		Type:        StorageTypeS3,
		Name:        "Flaky S3 " + uuid.New().String(),
		S3Storage: &s3_storage.S3Storage{
			S3Bucket:    "issue-582-no-such-bucket-" + uuid.New().String(),
			S3Region:    "us-east-1",
			S3AccessKey: accessKey,
			S3SecretKey: secretKey,
			S3Endpoint:  endpoint,
		},
	}

	saved, err := getStorageRepository().Save(storage)
	if err != nil {
		panic(err)
	}

	return saved
}

// The context is stripped of cancellation because callers pass t.Context() from a t.Cleanup, and
// the test context is already cancelled by the time cleanup runs.
func RemoveTestStorage(ctx context.Context, id uuid.UUID) {
	ctx = context.WithoutCancel(ctx)

	storage, err := getStorageRepository().FindByID(ctx, id)
	if err != nil {
		panic(err)
	}

	err = getStorageRepository().Delete(storage)
	if err != nil {
		panic(err)
	}
}
