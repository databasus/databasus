package files

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartBackgroundWrite(t *testing.T) {
	t.Parallel()

	t.Run("delivers the receipt and no error once the producer closes the pipe", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		source, producer := io.Pipe()

		write := files.store.StartBackgroundWrite(t.Context(), BackgroundWriteSpec{
			Reference: reference,
			Source:    source,
			OnFailure: func(error) { t.Error("a successful upload must not report a failure") },
		})

		_, err := producer.Write([]byte("payload"))
		require.NoError(t, err)
		require.NoError(t, producer.Close())

		receipt := <-write.Receipts
		require.NoError(t, <-write.Errors)
		assert.Equal(t, reference, receipt.Reference)
		assert.Equal(t, []string{"backup-1"}, files.provider.getSaved())
	})

	t.Run("closes the pipe and reports the failure when the upload fails", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.saveErr = errors.New("provider is down")
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		source, producer := io.Pipe()
		reportedFailures := make(chan error, 1)

		write := files.store.StartBackgroundWrite(t.Context(), BackgroundWriteSpec{
			Reference: reference,
			Source:    source,
			OnFailure: func(cause error) { reportedFailures <- cause },
		})

		_, writeErr := producer.Write([]byte("payload"))
		require.NoError(t, writeErr)
		require.NoError(t, producer.Close())

		<-write.Receipts
		require.ErrorIs(t, <-write.Errors, files.provider.saveErr)
		assert.ErrorIs(t, <-reportedFailures, files.provider.saveErr)
	})

	t.Run("reports a stopped write when the upload panics", func(t *testing.T) {
		t.Parallel()
		files := newTestFiles(t)
		files.provider.savePanics = true
		reference := StoredFileReference{StorageID: files.createStorage(t), FileName: "backup-1"}
		source, producer := io.Pipe()
		reportedFailures := make(chan error, 1)

		write := files.store.StartBackgroundWrite(t.Context(), BackgroundWriteSpec{
			Reference: reference,
			Source:    source,
			OnFailure: func(cause error) { reportedFailures <- cause },
		})

		<-write.Receipts
		require.ErrorIs(t, <-write.Errors, ErrBackgroundWriteStopped)
		require.ErrorIs(t, <-reportedFailures, ErrBackgroundWriteStopped)

		_, err := producer.Write([]byte("payload"))
		assert.ErrorIs(t, err, ErrBackgroundWriteStopped, "the producer learns that nobody reads the pipe")
	})
}
