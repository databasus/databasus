package s3_storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"

	io_utils "databasus-backend/internal/util/io"
)

type reassemblingReaderSpec struct {
	streamCtx  context.Context
	logger     *slog.Logger
	coreClient *minio.Core
	bucket     string
	parts      []objectSpan

	// Files stored before chunking, and any file small enough to fit one object, have no manifest
	// and therefore no recorded sha256 to compare against; their length comes from StatObject.
	hasManifestChecksum bool

	stallTimeout               time.Duration
	maxAttemptsWithoutProgress int
	retryBaseDelay             time.Duration
}

// reassemblingReader streams a chunked backup back as one byte stream, holding at most one part
// connection at a time to preserve the bounded-memory streaming of the upload path
// (docs/architecture.md, "Backup data flow").
//
// A dropped or stalled connection is retried in place, resuming the current part from the last byte
// handed out via a Range request. It has to recover here rather than downstream: the bytes are an
// encrypted stream with no end-of-stream marker and the decryptor is stateful, so a caller that sees
// a short read cannot resynchronise.
type reassemblingReader struct {
	streamCtx  context.Context
	logger     *slog.Logger
	coreClient *minio.Core
	bucket     string
	parts      []objectSpan

	hasManifestChecksum bool
	stallTimeout        time.Duration
	retryPolicy         *io_utils.ReadRetryPolicy

	isClosed         bool
	partIndex        int
	partBody         io.ReadCloser
	partHasher       hash.Hash
	attemptCancel    context.CancelFunc
	stallTimer       *time.Timer
	partReadBytes    int64
	attemptReadBytes int64
}

func newReassemblingReader(spec reassemblingReaderSpec) *reassemblingReader {
	if spec.stallTimeout == 0 {
		spec.stallTimeout = io_utils.DefaultReadStallTimeout
	}

	// The retry path is the only thing that logs, so a nil logger would panic exactly during an
	// incident and nowhere else.
	if spec.logger == nil {
		spec.logger = slog.New(slog.DiscardHandler)
	}

	return &reassemblingReader{
		streamCtx:           spec.streamCtx,
		logger:              spec.logger,
		coreClient:          spec.coreClient,
		bucket:              spec.bucket,
		parts:               spec.parts,
		hasManifestChecksum: spec.hasManifestChecksum,
		stallTimeout:        spec.stallTimeout,
		retryPolicy: io_utils.NewReadRetryPolicy(io_utils.ReadRetryPolicySpec{
			MaxAttemptsWithoutProgress: spec.maxAttemptsWithoutProgress,
			BaseDelay:                  spec.retryBaseDelay,
		}),
		partHasher: sha256.New(),
	}
}

func (r *reassemblingReader) Read(p []byte) (int, error) {
	for {
		if r.isClosed {
			return 0, io_utils.ErrReaderClosed
		}

		if err := r.streamCtx.Err(); err != nil {
			return 0, err
		}

		if r.partIndex >= len(r.parts) {
			return 0, io.EOF
		}

		if r.partBody == nil {
			if r.partReadBytes >= r.parts[r.partIndex].Size {
				if err := r.finishCurrentPart(); err != nil {
					return 0, err
				}

				continue
			}

			if err := r.openCurrentPart(); err != nil {
				if retryErr := r.resumeAfterFailure(err); retryErr != nil {
					return 0, retryErr
				}

				continue
			}
		}

		r.stallTimer.Reset(r.stallTimeout)
		n, err := r.partBody.Read(p)
		r.stallTimer.Stop()

		if n > 0 {
			if r.hasManifestChecksum {
				r.partHasher.Write(p[:n])
			}

			r.partReadBytes += int64(n)
			r.attemptReadBytes += int64(n)
		}

		// A body that ends before the length the response promised is a truncated transfer, not the
		// end of the part: net/http reports that as ErrUnexpectedEOF, but a chunked-encoded response
		// can deliver it as a plain EOF, so trust the byte count rather than the error.
		isPartComplete := r.partReadBytes >= r.parts[r.partIndex].Size
		if errors.Is(err, io.EOF) && isPartComplete {
			if finishErr := r.finishCurrentPart(); finishErr != nil {
				return n, finishErr
			}

			if n > 0 {
				return n, nil
			}

			continue
		}

		if err != nil {
			readErr := fmt.Errorf("failed to read chunk %s: %w", r.parts[r.partIndex].Key, err)
			if retryErr := r.resumeAfterFailure(readErr); retryErr != nil {
				return n, retryErr
			}

			if n > 0 {
				return n, nil
			}

			continue
		}

		return n, nil
	}
}

func (r *reassemblingReader) Close() error {
	r.isClosed = true

	r.abandonAttempt()

	return nil
}

func (r *reassemblingReader) openCurrentPart() error {
	part := r.parts[r.partIndex]

	attemptCtx, cancelAttempt := context.WithCancel(r.streamCtx)

	options := minio.GetObjectOptions{}

	if r.partReadBytes > 0 {
		if err := options.SetRange(r.partReadBytes, part.Size-1); err != nil {
			cancelAttempt()

			return fmt.Errorf(
				"%w: chunk %s at offset %d: %w",
				io_utils.ErrRangeNotHonoured,
				part.Key,
				r.partReadBytes,
				err,
			)
		}
	}

	body, objectInfo, responseHeader, err := r.coreClient.GetObject(attemptCtx, r.bucket, part.Key, options)
	if err != nil {
		cancelAttempt()

		return fmt.Errorf("failed to open chunk %s: %w", part.Key, err)
	}

	if err := r.verifyResponseSpan(objectInfo, responseHeader); err != nil {
		cancelAttempt()
		_ = body.Close()

		return err
	}

	r.partBody = body
	r.attemptCancel = cancelAttempt
	r.stallTimer = time.AfterFunc(r.stallTimeout, cancelAttempt)
	r.stallTimer.Stop()

	return nil
}

// A resumed read must not begin before the offset already handed out: replaying bytes into a
// stateful decryptor corrupts the stream. Content-Range is the positive proof the backend applied the
// Range; failing that, a body whose length equals the remainder is accepted, and anything else is
// refused because an unverifiable resume cannot be told apart from a whole-object reply.
func (r *reassemblingReader) verifyResponseSpan(
	objectInfo minio.ObjectInfo,
	responseHeader http.Header,
) error {
	part := r.parts[r.partIndex]
	expectedBytes := part.Size - r.partReadBytes

	if r.partReadBytes > 0 {
		if rangeStart, isParsed := io_utils.GetRangeStartOfContentRange(responseHeader.Get("Content-Range")); isParsed {
			if rangeStart != r.partReadBytes {
				return fmt.Errorf(
					"%w: chunk %s resumed at offset %d but the response starts at %d",
					io_utils.ErrRangeNotHonoured, part.Key, r.partReadBytes, rangeStart,
				)
			}

			return nil
		}

		if objectInfo.Size != expectedBytes {
			return fmt.Errorf(
				"%w: chunk %s resumed at offset %d answered %d bytes, expected %d",
				io_utils.ErrRangeNotHonoured, part.Key, r.partReadBytes, objectInfo.Size, expectedBytes,
			)
		}

		return nil
	}

	// A response sent with chunked transfer-encoding carries no Content-Length, which minio-go
	// reports as size -1. That is not a short object: the part's byte count and sha256 are still
	// verified when it ends, so refusing here would break backends that never declare a length.
	if objectInfo.Size >= 0 && objectInfo.Size != expectedBytes {
		return fmt.Errorf(
			"%w: chunk %s is %d bytes, expected %d",
			ErrChunkSizeMismatch, part.Key, objectInfo.Size, expectedBytes,
		)
	}

	return nil
}

func (r *reassemblingReader) finishCurrentPart() error {
	part := r.parts[r.partIndex]

	r.abandonAttempt()

	if r.partReadBytes != part.Size {
		return fmt.Errorf(
			"%w: chunk %s read %d bytes, expected %d",
			ErrChunkSizeMismatch, part.Key, r.partReadBytes, part.Size,
		)
	}

	if r.hasManifestChecksum {
		if checksum := hex.EncodeToString(r.partHasher.Sum(nil)); checksum != part.SHA256 {
			return fmt.Errorf("%w: chunk %s", ErrChunkChecksumMismatch, part.Key)
		}
	}

	r.partIndex++
	r.partReadBytes = 0
	r.retryPolicy.ResetAttemptsAfterCompletedRead()
	r.partHasher = sha256.New()

	return nil
}

func (r *reassemblingReader) resumeAfterFailure(cause error) error {
	hasDeliveredBytes := r.attemptReadBytes > 0
	partKey := r.parts[r.partIndex].Key

	r.abandonAttempt()

	if err := r.streamCtx.Err(); err != nil {
		return errors.Join(cause, err)
	}

	if !isRetryableReadError(cause) {
		return cause
	}

	r.retryPolicy.RegisterFailedAttempt(hasDeliveredBytes)

	if r.retryPolicy.IsExhausted() {
		return fmt.Errorf(
			"%w: chunk %s stuck at offset %d: %w",
			io_utils.ErrReadRetryBudgetExhausted, partKey, r.partReadBytes, cause,
		)
	}

	r.logger.Warn(
		fmt.Sprintf(
			"s3 chunk read failed, resuming from offset %d (attempt %d of %d)",
			r.partReadBytes,
			r.retryPolicy.GetAttemptsSinceProgress(),
			r.retryPolicy.GetMaxAttemptsWithoutProgress(),
		),
		"part_key", partKey,
		"error", cause,
	)

	if hasDeliveredBytes {
		return nil
	}

	return r.retryPolicy.WaitBeforeRetry(r.streamCtx)
}

func (r *reassemblingReader) abandonAttempt() {
	r.attemptReadBytes = 0

	if r.stallTimer != nil {
		r.stallTimer.Stop()
		r.stallTimer = nil
	}

	if r.partBody != nil {
		_ = r.partBody.Close()
		r.partBody = nil
	}

	if r.attemptCancel != nil {
		r.attemptCancel()
		r.attemptCancel = nil
	}
}

func isRetryableReadError(err error) bool {
	if errors.Is(err, io_utils.ErrRangeNotHonoured) ||
		errors.Is(err, ErrChunkSizeMismatch) ||
		errors.Is(err, ErrChunkChecksumMismatch) {
		return false
	}

	var errorResponse minio.ErrorResponse
	if errors.As(err, &errorResponse) {
		return errorResponse.StatusCode == http.StatusRequestTimeout ||
			errorResponse.StatusCode == http.StatusTooManyRequests ||
			errorResponse.StatusCode >= http.StatusInternalServerError
	}

	return true
}
