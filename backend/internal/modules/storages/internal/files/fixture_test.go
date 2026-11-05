package files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/lib/testdb"
	"databasus-backend/internal/modules/storages/internal/providers"
)

// testTimings keep the production shape. The clock is manual, so a window passes when the test moves
// the clock, not when the test waits.
func testTimings() Timings {
	return Timings{
		CommitWindow:   time.Hour,
		AttemptLease:   time.Minute,
		RetryBaseDelay: 10 * time.Second,
		RetryMaxDelay:  40 * time.Second,
		WorkerTick:     time.Minute,
		ClaimBatchSize: 10,
	}
}

// testFiles is a store and a deletion job over a database of their own, so the claims of one test
// never reach the rows of another.
type testFiles struct {
	db         *gorm.DB
	transactor *database.Transactor
	repository *PendingDeletionRepository
	clock      *clock.Manual
	provider   *fakeProvider
	locator    *fakeLocator
	logs       *capturingHandler
	uploads    *background.Group
	store      *Store
	job        *DeletionJob
}

func newTestFiles(t *testing.T) *testFiles {
	t.Helper()

	return newTestFilesWithTimings(t, testTimings())
}

func newTestFilesWithTimings(t *testing.T, timings Timings) *testFiles {
	t.Helper()

	db, err := database.Open(t.Context(), testdb.Open(t))
	require.NoError(t, err)

	t.Cleanup(func() {
		sqlDB, dbErr := db.DB()
		if dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	logs := newCapturingHandler()
	logger := logging.NewWithSinks(logs).GetLogger()
	provider := &fakeProvider{}
	locator := &fakeLocator{provider: provider}
	repository := NewPendingDeletionRepository(db)
	manualClock := clock.NewManual(time.Date(2026, time.October, 26, 12, 0, 0, 0, time.UTC))
	uploads := background.New(background.Deps{Logger: logger})

	deps := Deps{
		Repository:      repository,
		WriteProtection: NewWriteProtectionRepository(db),
		Transactor:      database.NewTransactor(db),
		Locator:         locator,
		Logger:          logger,
		Clock:           manualClock,
		Timings:         timings,
		Uploads:         uploads,
	}
	store := NewStore(deps)
	job := NewDeletionJob(store, deps)
	job.jitter = func(delay time.Duration) time.Duration { return delay }

	return &testFiles{
		db:         db,
		transactor: database.NewTransactor(db),
		repository: repository,
		clock:      manualClock,
		provider:   provider,
		locator:    locator,
		logs:       logs,
		uploads:    uploads,
		store:      store,
		job:        job,
	}
}

// createStorage inserts the rows through SQL, because the storage service lives in a package that
// imports this one.
func (f *testFiles) createStorage(t *testing.T) uuid.UUID {
	t.Helper()

	workspaceID := uuid.New()
	storageID := uuid.New()

	f.execute(t, "INSERT INTO workspaces (id, name) VALUES (?, ?)", workspaceID, "workspace "+workspaceID.String())
	f.execute(t, "INSERT INTO storages (id, workspace_id, type, name) VALUES (?, ?, 'LOCAL', ?)",
		storageID, workspaceID, "storage "+storageID.String())

	return storageID
}

func (f *testFiles) execute(t *testing.T, statement string, args ...any) {
	t.Helper()

	require.NoError(t, f.db.WithContext(t.Context()).Exec(statement, args...).Error)
}

func (f *testFiles) inTransaction(t *testing.T, work func(ctx context.Context) error) error {
	t.Helper()

	return f.transactor.Transaction(t.Context(), work)
}

func (f *testFiles) requestDeletions(t *testing.T, references ...StoredFileReference) {
	t.Helper()

	require.NoError(t, f.inTransaction(t, func(ctx context.Context) error {
		return f.store.RequestFileDeletions(ctx, references)
	}))
}

func (f *testFiles) runPass(t *testing.T) {
	t.Helper()

	require.NoError(t, f.job.RunPass(t.Context(), f.clock.Now()))
}

func (f *testFiles) findPending(t *testing.T, reference StoredFileReference) (PendingDeletion, bool) {
	t.Helper()

	var pending PendingDeletion

	err := f.db.WithContext(t.Context()).
		Where("storage_id = ? AND file_name = ?", reference.StorageID, reference.FileName).
		First(&pending).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return PendingDeletion{}, false
	}

	require.NoError(t, err)

	return pending, true
}

// isDue asks the database, because eligibility is a comparison the claim makes there.
func (f *testFiles) isDue(t *testing.T, reference StoredFileReference) bool {
	t.Helper()

	var count int64

	require.NoError(t, f.db.WithContext(t.Context()).Model(&PendingDeletion{}).
		Where("storage_id = ? AND file_name = ? AND not_before <= ?",
			reference.StorageID, reference.FileName, f.clock.Now()).
		Count(&count).Error)

	return count == 1
}

func (f *testFiles) getLiveWriteCount() int {
	count := 0
	_ = f.store.withLiveWrites(func(ids []uuid.UUID) error { count = len(ids); return nil })
	return count
}

func referencesOf(rows []PendingDeletion) []StoredFileReference {
	references := make([]StoredFileReference, 0, len(rows))
	for _, row := range rows {
		references = append(references, row.GetReference())
	}

	return references
}

type fakeProvider struct {
	mutex           sync.Mutex
	saveErr         error
	deleteErr       error
	saved           []string
	contents        map[string][]byte
	deleted         []string
	onSave          func()
	onDelete        func()
	onDeleteContext func(context.Context)
	savePanics      bool
}

func (p *fakeProvider) SaveFile(_ context.Context, file providers.FileAccess, content io.Reader) error {
	if p.onSave != nil {
		p.onSave()
	}

	if p.savePanics {
		panic("provider exploded")
	}

	readContent, _ := io.ReadAll(content)

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.saveErr != nil {
		return p.saveErr
	}

	p.saved = append(p.saved, file.FileName)

	if p.contents == nil {
		p.contents = map[string][]byte{}
	}

	p.contents[file.FileName] = readContent

	return nil
}

func (p *fakeProvider) OpenFile(_ context.Context, file providers.FileAccess) (io.ReadCloser, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	content, isSaved := p.contents[file.FileName]
	if !isSaved {
		return nil, errFakeFileNotFound
	}

	return io.NopCloser(bytes.NewReader(content)), nil
}

func (p *fakeProvider) DeleteFile(ctx context.Context, file providers.FileAccess) error {
	if p.onDeleteContext != nil {
		p.onDeleteContext(ctx)
	}
	if p.onDelete != nil {
		p.onDelete()
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.deleteErr != nil {
		return p.deleteErr
	}

	p.deleted = append(p.deleted, file.FileName)

	return nil
}

func (p *fakeProvider) getSaved() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return append([]string{}, p.saved...)
}

func (p *fakeProvider) getDeleted() []string {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return append([]string{}, p.deleted...)
}

func (p *fakeProvider) setDeleteErr(err error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.deleteErr = err
}

type fakeLocator struct {
	provider *fakeProvider
	err      error
}

func (l *fakeLocator) LocateStorage(_ context.Context, storageID uuid.UUID) (LocatedStorage, error) {
	if l.err != nil {
		return LocatedStorage{}, l.err
	}

	return LocatedStorage{
		Provider: l.provider,
		Location: providers.Location{StorageID: storageID, Settings: nil},
	}, nil
}

// capturingHandler shares its records with the handlers derived from it, because the code under test
// logs through loggers built with With.
type capturingHandler struct {
	mutex   *sync.Mutex
	records *[]slog.Record
	attrs   []slog.Attr
}

func newCapturingHandler() *capturingHandler {
	return &capturingHandler{mutex: &sync.Mutex{}, records: &[]slog.Record{}, attrs: nil}
}

func (h *capturingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, record slog.Record) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	record.AddAttrs(h.attrs...)
	*h.records = append(*h.records, record)

	return nil
}

func (h *capturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &capturingHandler{
		mutex:   h.mutex,
		records: h.records,
		attrs:   append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
}

func (h *capturingHandler) WithGroup(string) slog.Handler { return h }

func (h *capturingHandler) findRecord(message string) (map[string]string, bool) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for _, record := range *h.records {
		if record.Message != message {
			continue
		}

		attributes := map[string]string{}

		record.Attrs(func(attr slog.Attr) bool {
			attributes[attr.Key] = attr.Value.String()

			return true
		})

		return attributes, true
	}

	return nil, false
}

var errFakeFileNotFound = errors.New("file not found")
