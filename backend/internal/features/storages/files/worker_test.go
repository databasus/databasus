package storage_files

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	db "databasus-backend/internal/storage"
)

// A derived handler shares the record list, because the code under test logs
// through loggers built with With(...).
type capturingHandler struct {
	records *[]slog.Record
	attrs   []slog.Attr
}

func newCapturingHandler() *capturingHandler {
	return &capturingHandler{records: &[]slog.Record{}}
}

func (h *capturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, record slog.Record) error {
	record.AddAttrs(h.attrs...)

	*h.records = append(*h.records, record)

	return nil
}

func (h *capturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &capturingHandler{records: h.records, attrs: append(append([]slog.Attr{}, h.attrs...), attrs...)}
}

func (h *capturingHandler) WithGroup(string) slog.Handler { return h }

func (h *capturingHandler) attrsOf(index int) map[string]any {
	found := map[string]any{}

	(*h.records)[index].Attrs(func(attr slog.Attr) bool {
		found[attr.Key] = attr.Value.Any()

		return true
	})

	return found
}

func newTestWorker(t *testing.T, store *Store, locator StorageLocator) *DeletionWorker {
	t.Helper()

	worker := NewDeletionWorker(store, testDependencies(locator))
	worker.jitter = func(d time.Duration) time.Duration { return d }

	return worker
}

func makeDue(t *testing.T, reference StoredFileReference) {
	t.Helper()

	require.NoError(t, db.GetDb().Model(&PendingDeletion{}).
		Where("storage_id = ? AND file_name = ?", reference.StorageID, reference.FileName).
		Update("not_before", gorm.Expr("now() - interval '1 second'")).Error)
}

func Test_RunOnce_WhenObligationIsDue_DeletesTheFileAndReleasesIt(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
	}))

	worker.RunOnce(t.Context())

	assert.Equal(t, []string{"backup-1"}, provider.deleted)

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func Test_RunOnce_WhenWriteIsLive_LeavesTheFileAloneUntilTheWriterIsGone(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	var deletedDuringWrite []string

	provider.onSave = func() {
		require.NoError(t, inTransaction(t, func(ctx context.Context) error {
			return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
		}))

		worker.RunOnce(t.Context())

		deletedDuringWrite = append([]string{}, provider.deleted...)
	}

	_, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
	require.ErrorIs(t, err, ErrFileTakenForDeletion)

	assert.Empty(t, deletedDuringWrite, "cleanup must not delete a file whose write is still running")

	worker.RunOnce(t.Context())

	assert.Equal(t, []string{"backup-1"}, provider.deleted)
}

func Test_RunOnce_WhenWriteWasInheritedFromAStoppedProcess_ClaimsItOnTheFirstPass(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	worker := newTestWorker(t, newTestStore(locator), locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	_, err := worker.repository.InsertIfAbsent(db.GetDb(), uuid.New(), reference, -time.Minute)
	require.NoError(t, err)

	worker.RunOnce(t.Context())

	assert.Equal(t, []string{"backup-1"}, provider.deleted)
}

func Test_RunOnce_WhenClaimingAFile_InvalidatesAnOutstandingReceipt(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	receipt, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
	require.NoError(t, err)

	makeDue(t, reference)
	worker.RunOnce(t.Context())

	err = inTransaction(t, func(ctx context.Context) error {
		return store.ConfirmFileWrites(ctx, []WriteReceipt{receipt})
	})

	assert.ErrorIs(t, err, ErrReceiptNoLongerValid)
	assert.Equal(t, []string{"backup-1"}, provider.deleted)
}

func Test_RunOnce_WhenProviderFailsThenRecovers_RetriesUntilTheFileIsGone(t *testing.T) {
	provider := &fakeProvider{deleteErr: errors.New("storage is unavailable")}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
	}))

	worker.RunOnce(t.Context())

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, pending, "a failed deletion stays an obligation")
	assert.Equal(t, 1, pending.AttemptCount)
	require.NotNil(t, pending.LastError)
	assert.Contains(t, *pending.LastError, "storage is unavailable")

	provider.deleteErr = nil
	makeDue(t, reference)

	worker.RunOnce(t.Context())

	pending, err = store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func Test_RunOnce_WhenDeletionWasRequestedRepeatedly_CountsOnlyTheAttempt(t *testing.T) {
	provider := &fakeProvider{deleteErr: errors.New("storage is unavailable")}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	for range 3 {
		require.NoError(t, inTransaction(t, func(ctx context.Context) error {
			return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
		}))
	}

	worker.RunOnce(t.Context())

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, pending)
	assert.Equal(t, 3, pending.Generation, "each request takes the file from whoever held it")
	assert.Equal(t, 1, pending.AttemptCount,
		"three requests and one failed deletion is one attempt, so the retry waits the base delay")
}

func Test_RunOnce_WhenAnotherCallerTakesTheFileMidAttempt_DoesNotReleaseTheirObligation(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
	}))

	provider.onDelete = func() {
		require.NoError(t, inTransaction(t, func(ctx context.Context) error {
			return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
		}))
	}

	worker.RunOnce(t.Context())

	pending, err := store.repository.FindByReference(db.GetDb(), reference)

	require.NoError(t, err)
	assert.NotNil(t, pending, "an attempt that lost its claim must not release the newer one")
}

func Test_RunOnce_WhenStorageDisappearsMidAttempt_FinishesQuietly(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	storageID := createStorageRow(t)
	reference := StoredFileReference{StorageID: storageID, FileName: "backup-1"}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
	}))

	provider.onDelete = func() {
		require.NoError(t, db.GetDb().Exec("DELETE FROM storages WHERE id = ?", storageID).Error)
	}

	require.NotPanics(t, func() { worker.RunOnce(t.Context()) })

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	assert.Nil(t, pending)
}

func Test_DrainForTest_WhenProviderFailsThenRecovers_ReturnsOnlyAfterTheFilesAreGone(t *testing.T) {
	provider := &fakeProvider{deleteErr: errors.New("storage is unavailable")}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	storageID := createStorageRow(t)
	first := StoredFileReference{StorageID: storageID, FileName: "backup-1"}
	second := StoredFileReference{StorageID: storageID, FileName: "backup-2"}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{first, second})
	}))

	// onDelete runs before DeleteFile reads deleteErr, so the first attempt has to
	// keep the error to fail; clearing it from the second call on lets the drain
	// recover without writing the field from another goroutine.
	attempts := 0

	provider.onDelete = func() {
		attempts++

		if attempts > 1 {
			provider.deleteErr = nil
		}
	}

	assert.NoError(t, worker.DrainForTest(t.Context(), first, second))
	assert.Greater(t, attempts, 2, "the drain must keep going past a failed attempt")

	for _, reference := range []StoredFileReference{first, second} {
		pending, err := store.repository.FindByReference(db.GetDb(), reference)
		require.NoError(t, err)
		assert.Nil(t, pending)
	}
}

func Test_DrainForTest_WhenNothingIsPending_ReturnsImmediately(t *testing.T) {
	locator := &fakeLocator{provider: &fakeProvider{}}
	worker := newTestWorker(t, newTestStore(locator), locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "never-written"}

	assert.NoError(t, worker.DrainForTest(t.Context(), reference))
}

func Test_RetryDelay_GrowsExponentiallyAndStopsAtTheMaximum(t *testing.T) {
	worker := newTestWorker(t, newTestStore(nil), nil)
	worker.timings.RetryBaseDelay = 10 * time.Millisecond
	worker.timings.RetryMaxDelay = 40 * time.Millisecond

	assert.Equal(t, 10*time.Millisecond, worker.retryDelay(1))
	assert.Equal(t, 20*time.Millisecond, worker.retryDelay(2))
	assert.Equal(t, 40*time.Millisecond, worker.retryDelay(3))
	assert.Equal(t, 40*time.Millisecond, worker.retryDelay(9))
	assert.Equal(t, 40*time.Millisecond, worker.retryDelay(500), "an overflowing exponent still lands on the cap")
}

func Test_SanitizeError_RemovesCredentialsAndBoundsLength(t *testing.T) {
	sanitized := sanitizeError(errors.New("dial postgres://admin:hunter2@db:5432 failed"))

	assert.NotContains(t, sanitized, "hunter2")

	truncated := sanitizeError(errors.New(strings.Repeat("x", maxPersistedErrorLength*2)))

	assert.Len(t, truncated, maxPersistedErrorLength)
}

func Test_RunOnce_WhenAttemptFails_LogsJobStorageFileAndRetry(t *testing.T) {
	handler := newCapturingHandler()
	provider := &fakeProvider{deleteErr: errors.New("storage is unavailable")}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	worker.logger = slog.New(handler)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
	}))

	worker.RunOnce(t.Context())

	require.NotEmpty(t, *handler.records)

	attempt := handler.attrsOf(0)
	assert.Equal(t, jobName, attempt["job_name"])
	assert.NotNil(t, attempt["job_id"])
	assert.Equal(t, reference.StorageID, attempt["storage_id"])
	assert.Equal(t, reference.FileName, attempt["file_name"])
	assert.Contains(t, (*handler.records)[0].Message, "attempt 1")
	assert.Contains(t, (*handler.records)[0].Message, "retrying in")

	summary := (*handler.records)[len(*handler.records)-1]
	assert.Contains(t, summary.Message, "pending")
	assert.Contains(t, summary.Message, "overdue")
	assert.Contains(t, summary.Message, "oldest")
}

func Test_Run_WhenCalledTwice_Panics(t *testing.T) {
	locator := &fakeLocator{provider: &fakeProvider{}}
	worker := newTestWorker(t, newTestStore(locator), locator)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	worker.Run(ctx)

	assert.Panics(t, func() { worker.Run(ctx) })
}

func Test_Run_WhenAPassIsSlow_DoesNotOverlapPasses(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	storageID := createStorageRow(t)
	// One row per pass and three rows waiting, one of which the startup pass takes
	// before the ticker exists: a pass that began while another was still running
	// would find work of its own and the counter would reach two.
	worker.timings.ClaimBatchSize = 1

	references := []StoredFileReference{
		{StorageID: storageID, FileName: "backup-1"},
		{StorageID: storageID, FileName: "backup-2"},
		{StorageID: storageID, FileName: "backup-3"},
	}

	var concurrentPasses atomic.Int32
	var maxConcurrentPasses atomic.Int32

	provider.onDelete = func() {
		if inFlight := concurrentPasses.Add(1); inFlight > maxConcurrentPasses.Load() {
			maxConcurrentPasses.Store(inFlight)
		}

		time.Sleep(5 * worker.timings.WorkerTick)

		concurrentPasses.Add(-1)
	}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, references)
	}))

	ctx, cancel := context.WithCancel(t.Context())
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		worker.Run(ctx)
	}()

	time.Sleep(20 * worker.timings.WorkerTick)
	cancel()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not stop")
	}

	assert.Equal(t, int32(1), maxConcurrentPasses.Load(),
		"exactly one pass runs at a time: a tick arriving during a pass is skipped, not stacked")
}

func Test_Run_WhenContextIsCancelledMidPass_FinishesTheProviderCallBeforeReturning(t *testing.T) {
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	store := newTestStore(locator)
	worker := newTestWorker(t, store, locator)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	ctx, cancel := context.WithCancel(t.Context())

	deleteReturned := false
	provider.onDelete = func() {
		cancel()
		time.Sleep(20 * time.Millisecond)

		deleteReturned = true
	}

	require.NoError(t, inTransaction(t, func(ctx context.Context) error {
		return store.RequestFileDeletions(ctx, []StoredFileReference{reference})
	}))

	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		worker.Run(ctx)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not stop")
	}

	assert.True(t, deleteReturned, "shutdown must not leave a provider call unjoined")
}

func Test_TimingsForTest_ShrinksEveryBoundReachingStoreAndWorker(t *testing.T) {
	production := ProductionTimings()
	shrunk := TimingsForTest()

	assert.Less(t, shrunk.CommitWindow, production.CommitWindow)
	assert.Less(t, shrunk.AttemptLease, production.AttemptLease)
	assert.Less(t, shrunk.RetryBaseDelay, production.RetryBaseDelay)
	assert.Less(t, shrunk.RetryMaxDelay, production.RetryMaxDelay)
	assert.Less(t, shrunk.WorkerTick, production.WorkerTick)

	locator := &fakeLocator{provider: &fakeProvider{}}
	store := newTestStore(locator)

	assert.Equal(t, shrunk, store.GetTimings())
	assert.Equal(t, shrunk, newTestWorker(t, store, locator).timings)
}
