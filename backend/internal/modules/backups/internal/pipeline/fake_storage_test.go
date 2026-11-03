package pipeline

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/modules/storages"
)

// writeFileFunc is the upload a fake storage runs in the background of a backup.
type writeFileFunc func(ctx context.Context, file io.Reader) error

// newTestPipeline builds a pipeline without engines whose storage runs writeFile in the background,
// and returns it with its master key.
func newTestPipeline(t *testing.T, storage fileStorage) *Pipeline {
	t.Helper()

	masterKey, err := crypto.OpenOrCreateMasterKey(filepath.Join(t.TempDir(), "secret.key"))
	require.NoError(t, err)

	return &Pipeline{engines: nil, storages: storage, masterKey: masterKey, shutdownBudget: nil}
}

func newTestBackupSpec() BackupSpec {
	backupID := uuid.New()

	return BackupSpec{
		BackupID:         backupID,
		DatabaseID:       uuid.New(),
		DatabaseType:     "",
		StorageID:        uuid.New(),
		FileName:         "test-backup-" + backupID.String(),
		IsEncrypted:      false,
		OnProgress:       func(float64) {},
		ProgressInterval: time.Hour,
		Logger:           slog.New(slog.DiscardHandler),
	}
}

// backgroundWritingStorage runs writeFile the way the file store runs an upload: it reads the pipe,
// closes it with the upload's error and ends the backup through OnFailure when the upload fails.
type backgroundWritingStorage struct {
	writeFile writeFileFunc
}

func (s *backgroundWritingStorage) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	write := storages.BackgroundWrite{
		Errors:   make(chan error, 1),
		Receipts: make(chan storages.WriteReceipt, 1),
	}

	go func() {
		err := s.writeFile(ctx, spec.Source)
		if err != nil {
			_ = spec.Source.CloseWithError(err)
			spec.OnFailure(err)
		}

		write.Receipts <- storages.WriteReceipt{}
		write.Errors <- err
	}()

	return write
}

func (s *backgroundWritingStorage) GetStorageReader(context.Context, uuid.UUID) (storages.StorageReader, error) {
	return nil, errors.New("the background writing storage reads nothing")
}

func newDiscardingStorage() *backgroundWritingStorage {
	return &backgroundWritingStorage{writeFile: func(_ context.Context, file io.Reader) error {
		_, err := io.Copy(io.Discard, file)

		return err
	}}
}

// memoryStorage keeps the last written file in memory and serves it back.
type memoryStorage struct {
	mutex      sync.Mutex
	storedFile []byte
}

func (s *memoryStorage) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	writer := &backgroundWritingStorage{writeFile: func(_ context.Context, file io.Reader) error {
		storedFile, err := io.ReadAll(file)

		s.mutex.Lock()
		s.storedFile = storedFile
		s.mutex.Unlock()

		return err
	}}

	return writer.StartBackgroundWrite(ctx, spec)
}

func (s *memoryStorage) GetStorageReader(context.Context, uuid.UUID) (storages.StorageReader, error) {
	return &memoryStorageReader{file: s.getStoredFile(), failAtEndErr: nil}, nil
}

func (s *memoryStorage) getStoredFile() []byte {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return bytes.Clone(s.storedFile)
}

// failingStorage fails every write with failureErr. It either drains the whole stream first, so the
// dump runs to completion before the storage fails, or fails before reading a byte, so the storage
// fails while the dump is still producing data.
type failingStorage struct {
	failureErr             error
	isFailingBeforeReading bool
	readBytes              atomic.Int64
}

func (s *failingStorage) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	writer := &backgroundWritingStorage{writeFile: func(_ context.Context, file io.Reader) error {
		if !s.isFailingBeforeReading {
			readBytes, _ := io.Copy(io.Discard, file)
			s.readBytes.Store(readBytes)
		}

		return s.failureErr
	}}

	return writer.StartBackgroundWrite(ctx, spec)
}

func (s *failingStorage) GetStorageReader(context.Context, uuid.UUID) (storages.StorageReader, error) {
	return nil, s.failureErr
}

// stalledStorage reads the first byte of a write and then stops reading without closing the stream,
// like a storage stuck on the network. It returns only when the backup context ends.
type stalledStorage struct {
	hasReadData atomic.Bool
}

func (s *stalledStorage) StartBackgroundWrite(
	ctx context.Context,
	spec storages.BackgroundWriteSpec,
) storages.BackgroundWrite {
	write := storages.BackgroundWrite{
		Errors:   make(chan error, 1),
		Receipts: make(chan storages.WriteReceipt, 1),
	}

	go func() {
		_, err := io.ReadFull(spec.Source, make([]byte, 1))
		if err == nil {
			s.hasReadData.Store(true)
			<-ctx.Done()
			err = context.Cause(ctx)
		}

		write.Receipts <- storages.WriteReceipt{}
		write.Errors <- err
	}()

	return write
}

func (s *stalledStorage) GetStorageReader(context.Context, uuid.UUID) (storages.StorageReader, error) {
	return nil, errors.New("the stalled storage reads nothing")
}

// memoryStorageReader serves one file from memory and fails the read at its end with failAtEndErr,
// like a download severed mid-file. Without failAtEndErr the file ends normally.
type memoryStorageReader struct {
	file         []byte
	failAtEndErr error
}

func (r *memoryStorageReader) OpenFile(context.Context, string) (io.ReadCloser, error) {
	var fileReader io.Reader = bytes.NewReader(r.file)
	if r.failAtEndErr != nil {
		fileReader = io.MultiReader(fileReader, failingReader{err: r.failAtEndErr})
	}

	return io.NopCloser(fileReader), nil
}

// stallingStorageReader serves one file from memory and then, instead of ending it, blocks the next
// read until the restore context ends, like a download that stopped arriving.
type stallingStorageReader struct {
	file          []byte
	hasServedFile atomic.Bool
}

func (r *stallingStorageReader) OpenFile(ctx context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(io.MultiReader(bytes.NewReader(r.file), stallingReader{ctx: ctx, reader: r})), nil
}

type stallingReader struct {
	ctx    context.Context
	reader *stallingStorageReader
}

func (r stallingReader) Read([]byte) (int, error) {
	r.reader.hasServedFile.Store(true)
	<-r.ctx.Done()

	return 0, context.Cause(r.ctx)
}

type failingReader struct {
	err error
}

func (r failingReader) Read([]byte) (int, error) {
	return 0, r.err
}
