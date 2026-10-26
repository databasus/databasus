package storage_files

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	db "databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
)

type fakeProvider struct {
	saveErr    error
	deleteErr  error
	saved      []string
	contents   map[string][]byte
	deleted    []string
	onSave     func()
	onDelete   func()
	savePanics bool
}

func (p *fakeProvider) SaveFile(
	_ context.Context, _ encryption.FieldEncryptor, _ *slog.Logger, fileName string, file io.Reader,
) error {
	if p.onSave != nil {
		p.onSave()
	}

	if p.savePanics {
		panic("provider exploded")
	}

	var content []byte
	if file != nil {
		content, _ = io.ReadAll(file)
	}

	if p.saveErr != nil {
		return p.saveErr
	}

	p.saved = append(p.saved, fileName)

	if p.contents == nil {
		p.contents = map[string][]byte{}
	}

	p.contents[fileName] = content

	return nil
}

func (p *fakeProvider) DeleteFile(
	_ context.Context, _ encryption.FieldEncryptor, _ *slog.Logger, fileName string,
) error {
	if p.onDelete != nil {
		p.onDelete()
	}

	if p.deleteErr != nil {
		return p.deleteErr
	}

	p.deleted = append(p.deleted, fileName)

	return nil
}

func (p *fakeProvider) GetFile(
	_ context.Context, _ encryption.FieldEncryptor, _ *slog.Logger, fileName string,
) (io.ReadCloser, error) {
	content, isSaved := p.contents[fileName]
	if !isSaved {
		return nil, errors.New("file not found")
	}

	return io.NopCloser(bytes.NewReader(content)), nil
}

type fakeLocator struct {
	provider *fakeProvider
	err      error
}

func (l *fakeLocator) GetFileWriter(_ context.Context, _ uuid.UUID) (FileWriter, error) {
	if l.err != nil {
		return nil, l.err
	}

	return l.provider, nil
}

func (l *fakeLocator) GetFileReader(_ context.Context, _ uuid.UUID) (FileReader, error) {
	if l.err != nil {
		return nil, l.err
	}

	return l.provider, nil
}

func (l *fakeLocator) GetFileRemover(_ context.Context, _ uuid.UUID) (FileRemover, error) {
	if l.err != nil {
		return nil, l.err
	}

	return l.provider, nil
}

func testDependencies(locator StorageLocator) Dependencies {
	return Dependencies{
		Repository:     &PendingDeletionRepository{},
		Locator:        locator,
		FieldEncryptor: encryption.GetFieldEncryptor(),
		Logger:         logger.GetLogger(),
		Timings:        TimingsForTest(),
	}
}

func newTestStore(locator StorageLocator) *Store {
	return NewStore(testDependencies(locator))
}

// Eligibility is a database-clock question, so the database answers it.
func isDue(t *testing.T, reference StoredFileReference) bool {
	t.Helper()

	var count int64

	require.NoError(t, db.GetDb().Model(&PendingDeletion{}).
		Where("storage_id = ? AND file_name = ? AND not_before <= now()", reference.StorageID, reference.FileName).
		Count(&count).Error)

	return count == 1
}

func liveWriteCount(t *testing.T, store *Store) int {
	t.Helper()

	count := 0
	require.NoError(t, store.withLiveWrites(func(excluded []uuid.UUID) error {
		count = len(excluded)

		return nil
	}))

	return count
}

func Test_WriteFile_WhenProviderAccepts_ReturnsReceiptAndHoldsObligation(t *testing.T) {
	provider := &fakeProvider{}
	store := newTestStore(&fakeLocator{provider: provider})
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	receipt, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

	require.NoError(t, err)
	assert.Equal(t, reference, receipt.Reference)
	assert.Equal(t, 0, receipt.Generation)
	assert.Equal(t, []string{"backup-1"}, provider.saved)

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, pending, "the obligation stands until a catalog transaction claims it")
	assert.False(t, isDue(t, reference), "a fresh upload gets its commit window before cleanup may act")
	assert.Zero(t, liveWriteCount(t, store))
}

func Test_WriteFile_WhenNameAlreadyRegistered_SendsNoBytes(t *testing.T) {
	provider := &fakeProvider{}
	store := newTestStore(&fakeLocator{provider: provider})
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.NoError(t, store.repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))

	_, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

	assert.ErrorIs(t, err, ErrFileNameAlreadyRegistered)
	assert.Empty(t, provider.saved)
	assert.Zero(t, liveWriteCount(t, store))
}

func Test_WriteFile_WhenProviderFails_LeavesFileEligibleOnceWriterIsGone(t *testing.T) {
	provider := &fakeProvider{saveErr: errors.New("provider is down")}
	store := newTestStore(&fakeLocator{provider: provider})
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	_, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

	require.ErrorIs(t, err, provider.saveErr)

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, pending)
	assert.True(t, isDue(t, reference), "a failed write is eligible immediately")
	assert.Zero(t, liveWriteCount(t, store))
}

func Test_WriteFile_WhileRunning_KeepsTheRowOutOfReachOfCleanup(t *testing.T) {
	store := newTestStore(nil)
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	var duringWrite int

	provider := &fakeProvider{onSave: func() {
		require.NoError(t, store.repository.InsertOrTake(db.GetDb(), []StoredFileReference{reference}))

		duringWrite = liveWriteCount(t, store)
	}}
	store.locator = &fakeLocator{provider: provider}

	_, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

	assert.ErrorIs(t, err, ErrFileTakenForDeletion)
	assert.Equal(t, 1, duringWrite, "the write is excluded from cleanup while it runs")
	assert.Zero(t, liveWriteCount(t, store))
}

func Test_WriteFile_WhenProviderPanics_LeavesTheLiveSet(t *testing.T) {
	provider := &fakeProvider{savePanics: true}
	store := newTestStore(&fakeLocator{provider: provider})
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	require.Panics(t, func() {
		_, _ = store.WriteFile(t.Context(), reference, strings.NewReader("payload"))
	})

	assert.Zero(t, liveWriteCount(t, store))
}

func Test_WriteFile_WhenStorageCannotBeLocated_LeavesFileEligible(t *testing.T) {
	store := newTestStore(&fakeLocator{err: errors.New("no such storage")})
	reference := StoredFileReference{StorageID: createStorageRow(t), FileName: "backup-1"}

	_, err := store.WriteFile(t.Context(), reference, strings.NewReader("payload"))

	require.Error(t, err)

	pending, err := store.repository.FindByReference(db.GetDb(), reference)
	require.NoError(t, err)
	require.NotNil(t, pending)
	assert.True(t, isDue(t, reference))
}
