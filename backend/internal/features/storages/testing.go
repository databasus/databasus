//go:build testhelpers

package storages_bridge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
)

const (
	storageFileDeletionJobName = "storage_file_deletion"
	// drainDeadlineForTest keeps a provider that never succeeds from hanging a test.
	drainDeadlineForTest = 30 * time.Second
	drainPassInterval    = 50 * time.Millisecond
)

var errStorageFileDeletionJobMissing = errors.New("the storages module has no storage file deletion job")

// CreateTestStorage inserts a local storage, as the old tests need a storage without a workspace
// member who could create one through the contract.
func CreateTestStorage(workspaceID uuid.UUID) storages.Storage {
	createdStorage := storages.Storage{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        "Test Storage " + uuid.NewString(),
		Type:        storages.TypeLocal,
	}

	insertTestStorage(createdStorage)
	execute("INSERT INTO storages_local (storage_id) VALUES (?)", createdStorage.ID)

	return createdStorage
}

// CreateTestFlakyS3Storage inserts an S3 storage whose bucket does not exist at the endpoint.
func CreateTestFlakyS3Storage(workspaceID uuid.UUID, endpoint string) storages.Storage {
	createdStorage := storages.Storage{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Name:        "Flaky S3 " + uuid.NewString(),
		Type:        storages.TypeS3,
	}

	insertTestStorage(createdStorage)
	execute(`INSERT INTO storages_s3 (storage_id, s3_bucket, s3_region, s3_access_key, s3_secret_key, s3_endpoint)
		VALUES (?, ?, 'us-east-1', ?, ?, ?)`,
		createdStorage.ID,
		"issue-582-no-such-bucket-"+uuid.NewString(),
		encryptTestSecret("issue-582-access"),
		encryptTestSecret("issue-582-secret"),
		endpoint,
	)

	return createdStorage
}

// RemoveTestStorage deletes the storage row; its provider row and its pending deletions go with it.
// The context is stripped of cancellation because callers pass t.Context() from a t.Cleanup, and the
// test context is already cancelled by the time cleanup runs.
func RemoveTestStorage(ctx context.Context, storageID uuid.UUID) {
	err := storage.GetDb().WithContext(context.WithoutCancel(ctx)).
		Exec("DELETE FROM storages WHERE id = ?", storageID).Error
	if err != nil {
		panic(err)
	}
}

// WriteTestFile writes a file through the file store and publishes it, as a completed backup does.
func WriteTestFile(ctx context.Context, reference storages.StoredFileReference, content io.Reader) error {
	receipt, err := GetModule().WriteFile(ctx, reference, content)
	if err != nil {
		return err
	}

	return storage.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		return GetModule().ConfirmFileWrites(ctx, []storages.WriteReceipt{receipt})
	})
}

// DrainStorageFileDeletions runs passes of the module's deletion job until none of the named files has
// a pending deletion. It watches only what the caller names, because the database of the test process
// is shared with every other test of the package running in parallel.
func DrainStorageFileDeletions(ctx context.Context, references ...storages.StoredFileReference) error {
	deletionJob, err := getStorageFileDeletionJob()
	if err != nil {
		return err
	}

	deadline := time.Now().UTC().Add(drainDeadlineForTest)

	for {
		outstandingCount, err := countPendingDeletions(ctx, references)
		if err != nil {
			return err
		}

		if outstandingCount == 0 {
			return nil
		}

		if time.Now().UTC().After(deadline) {
			return fmt.Errorf("%d file deletions still pending after %s", outstandingCount, drainDeadlineForTest)
		}

		err = deletionJob.Run(ctx, time.Now().UTC())
		if err != nil {
			return err
		}

		time.Sleep(drainPassInterval)
	}
}

func getStorageFileDeletionJob() (background.Job, error) {
	for _, job := range GetModule().GetJobs() {
		if job.Name == storageFileDeletionJobName {
			return job, nil
		}
	}

	return background.Job{}, errStorageFileDeletionJobMissing
}

func countPendingDeletions(ctx context.Context, references []storages.StoredFileReference) (int64, error) {
	var outstandingCount int64

	for _, reference := range references {
		var referenceCount int64

		err := storage.GetDb().WithContext(ctx).
			Raw("SELECT COUNT(*) FROM storages_pending_deletions WHERE storage_id = ? AND file_name = ?",
				reference.StorageID, reference.FileName).
			Scan(&referenceCount).Error
		if err != nil {
			return 0, fmt.Errorf("count pending deletions: %w", err)
		}

		outstandingCount += referenceCount
	}

	return outstandingCount, nil
}

func insertTestStorage(createdStorage storages.Storage) {
	execute("INSERT INTO storages (id, workspace_id, name, type) VALUES (?, ?, ?, ?)",
		createdStorage.ID, createdStorage.WorkspaceID, createdStorage.Name, createdStorage.Type)
}

func encryptTestSecret(secret string) string {
	encryptedSecret, err := encryption.GetFieldEncryptor().Encrypt(secret)
	if err != nil {
		panic(err)
	}

	return encryptedSecret
}

func execute(statement string, values ...any) {
	err := storage.GetDb().Exec(statement, values...).Error
	if err != nil {
		panic(err)
	}
}

// WriteFileFunc is the WriteFile of a fake file store.
type WriteFileFunc func(
	ctx context.Context,
	reference storages.StoredFileReference,
	file io.Reader,
) (storages.WriteReceipt, error)

// StartFakeBackgroundWrite runs writeFile beside its producer the way the module's background write
// does, so a fake file store of an old test can stand in for the module.
func StartFakeBackgroundWrite(
	ctx context.Context,
	writeFile WriteFileFunc,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	write := storages.BackgroundWrite{
		Errors:   make(chan error, 1),
		Receipts: make(chan storages.WriteReceipt, 1),
	}

	go func() {
		receipt, err := writeFile(ctx, spec.Reference, spec.Source)
		if err != nil {
			_ = spec.Source.CloseWithError(err)
			spec.OnFailure(err)
		}

		write.Receipts <- receipt
		write.Errors <- err
	}()

	return write
}

// OpenTestFile opens a stored file through the storage reader, as restores and downloads read it.
func OpenTestFile(ctx context.Context, reference storages.StoredFileReference) (io.ReadCloser, error) {
	storageReader, err := GetModule().GetStorageReader(ctx, reference.StorageID)
	if err != nil {
		return nil, err
	}

	return storageReader.OpenFile(ctx, reference.FileName)
}

// DeleteTestFile requests the deletion of a stored file and waits until the deletion job removed it.
func DeleteTestFile(ctx context.Context, reference storages.StoredFileReference) error {
	ctx = context.WithoutCancel(ctx)

	err := storage.GetTransactor().Transaction(ctx, func(ctx context.Context) error {
		return GetModule().RequestFileDeletions(ctx, []storages.StoredFileReference{reference})
	})
	if err != nil {
		return err
	}

	return DrainStorageFileDeletions(ctx, reference)
}
