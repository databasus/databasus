package pipeline

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	io_utils "databasus-backend/internal/lib/iox"
)

func TestGetBackupStreamFailure(t *testing.T) {
	t.Parallel()

	t.Run(
		"reports the transport error instead of the decode error when the storage drops mid-stream",
		func(t *testing.T) {
			t.Parallel()
			errConnectionReset := errors.New("connection reset by peer")
			compressedBackup := encodeStoredFile(t, newZstdStoredCompression(3),
				bytes.Repeat([]byte("INSERT INTO users VALUES (1);\n"), 1000))
			storageRead := io_utils.NewFailureTrackingReader(&severingReader{
				content:    compressedBackup,
				failAfter:  len(compressedBackup) / 2,
				failWith:   errConnectionReset,
				readAtOnce: 64,
			})
			decodedStream := io_utils.NewFailureTrackingReader(decodeZstd(t, storageRead))

			_, copyErr := io.Copy(io.Discard, decodedStream)
			require.Error(t, copyErr, "the truncated download must break decompression")

			streamFailure := getBackupStreamFailure(
				backupStreamTrackers{storageRead: storageRead, decodedStream: decodedStream},
			)

			require.ErrorIs(t, streamFailure, errConnectionReset)
			require.ErrorIs(t, streamFailure, errBackupStreamFailed)
		},
	)

	t.Run("reports the decode error when the storage read is intact", func(t *testing.T) {
		t.Parallel()
		errCorruptArchive := errors.New("corrupt zstd frame")
		storageRead := io_utils.NewFailureTrackingReader(bytes.NewReader([]byte("payload")))
		decodedStream := io_utils.NewFailureTrackingReader(&severingReader{
			content:    []byte("payload"),
			failAfter:  2,
			failWith:   errCorruptArchive,
			readAtOnce: 1,
		})

		_, _ = io.Copy(io.Discard, storageRead)
		_, _ = io.Copy(io.Discard, decodedStream)

		streamFailure := getBackupStreamFailure(
			backupStreamTrackers{storageRead: storageRead, decodedStream: decodedStream},
		)

		require.ErrorIs(t, streamFailure, errCorruptArchive)
	})

	t.Run("reports nothing when both streams completed", func(t *testing.T) {
		t.Parallel()
		storageRead := io_utils.NewFailureTrackingReader(bytes.NewReader([]byte("payload")))
		decodedStream := io_utils.NewFailureTrackingReader(bytes.NewReader([]byte("payload")))

		_, _ = io.Copy(io.Discard, storageRead)
		_, _ = io.Copy(io.Discard, decodedStream)

		require.NoError(t, getBackupStreamFailure(backupStreamTrackers{
			storageRead:   storageRead,
			decodedStream: decodedStream,
		}))
	})
}

func decodeZstd(t *testing.T, compressed io.Reader) io.Reader {
	t.Helper()

	decoder, err := zstd.NewReader(compressed)
	require.NoError(t, err)
	t.Cleanup(decoder.Close)

	return decoder
}

type severingReader struct {
	content        []byte
	failAfter      int
	failWith       error
	readAtOnce     int
	deliveredBytes int
}

func (r *severingReader) Read(p []byte) (int, error) {
	if r.deliveredBytes >= r.failAfter {
		return 0, r.failWith
	}

	readable := min(len(p), r.readAtOnce, r.failAfter-r.deliveredBytes)

	copy(p, r.content[r.deliveredBytes:r.deliveredBytes+readable])
	r.deliveredBytes += readable

	return readable, nil
}
