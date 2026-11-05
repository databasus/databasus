package iox

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"
)

// Once headers have arrived the transport's response timeout no longer applies, so a backend that
// accepts the read and then goes silent would block until the OS TCP timeout. It is armed around
// each blocking read only: the gap between a consumer's reads is its own business, and a restore
// target that pauses on a slow statement must not lose its connection over it.
const DefaultReadStallTimeout = 60 * time.Second

// UnknownTotalBytes skips the completeness check rather than guessing a size: a wrong expected
// size would reject an intact file.
const UnknownTotalBytes = -1

// The offset is where the stream must continue from; a backend that cannot honour it must fail
// rather than answer from the start, because replaying bytes corrupts a stateful decryptor.
type OpenFileAtOffsetFunc func(ctx context.Context, offsetBytes int64) (io.ReadCloser, error)

type ResumingReaderSpec struct {
	StreamCtx    context.Context
	Logger       *slog.Logger
	FileName     string
	OpenAtOffset OpenFileAtOffsetFunc

	TotalBytes int64

	StallTimeout    time.Duration
	RetryPolicySpec ReadRetryPolicySpec

	// Left nil, every failure is retried. Backends that can tell a missing file from a severed
	// connection should say so here, otherwise a permanent failure costs the whole budget.
	IsRetryableError func(error) bool
}

// A dropped or stalled connection is retried in place, resuming from the last byte handed out.
// It has to recover here rather than downstream: backups are an encrypted, compressed stream with
// no end-of-stream marker, so a caller that sees a short read cannot resynchronise and would feed
// the restore target a silently truncated dump.
type ResumingReader struct {
	streamCtx        context.Context
	logger           *slog.Logger
	fileName         string
	openAtOffset     OpenFileAtOffsetFunc
	totalBytes       int64
	stallTimeout     time.Duration
	retryPolicy      *ReadRetryPolicy
	isRetryableError func(error) bool

	isClosed         bool
	isFinished       bool
	body             io.ReadCloser
	attemptCancel    context.CancelFunc
	stallTimer       *time.Timer
	deliveredBytes   int64
	attemptReadBytes int64
}

func NewResumingReader(spec ResumingReaderSpec) *ResumingReader {
	if spec.StallTimeout == 0 {
		spec.StallTimeout = DefaultReadStallTimeout
	}

	// The retry path is the only thing that logs, so a nil logger would panic exactly during an
	// incident and nowhere else.
	if spec.Logger == nil {
		spec.Logger = slog.New(slog.DiscardHandler)
	}

	if spec.IsRetryableError == nil {
		spec.IsRetryableError = func(error) bool { return true }
	}

	return &ResumingReader{
		streamCtx:        spec.StreamCtx,
		logger:           spec.Logger,
		fileName:         spec.FileName,
		openAtOffset:     spec.OpenAtOffset,
		totalBytes:       spec.TotalBytes,
		stallTimeout:     spec.StallTimeout,
		retryPolicy:      NewReadRetryPolicy(spec.RetryPolicySpec),
		isRetryableError: spec.IsRetryableError,
	}
}

func (r *ResumingReader) Read(p []byte) (int, error) {
	for {
		if r.isClosed {
			return 0, ErrReaderClosed
		}

		if err := r.streamCtx.Err(); err != nil {
			return 0, err
		}

		if r.isFinished || r.hasReadDeclaredSize() {
			return 0, io.EOF
		}

		if r.body == nil {
			if err := r.openAtCurrentOffset(); err != nil {
				if retryErr := r.resumeAfterFailure(err); retryErr != nil {
					return 0, retryErr
				}

				continue
			}
		}

		r.stallTimer.Reset(r.stallTimeout)
		n, err := r.body.Read(p)
		r.stallTimer.Stop()

		if n > 0 {
			r.deliveredBytes += int64(n)
			r.attemptReadBytes += int64(n)
		}

		if r.totalBytes >= 0 && r.deliveredBytes > r.totalBytes {
			r.abandonAttempt()

			return n, fmt.Errorf(
				"%w: %s answered %d bytes, expected %d",
				ErrRangeNotHonoured, r.fileName, r.deliveredBytes, r.totalBytes,
			)
		}

		// A body that ends before the declared size is a severed transfer, not the end of the
		// file: net/http reports that as ErrUnexpectedEOF, but a chunked-encoded response can
		// deliver it as a plain EOF, so trust the byte count rather than the error.
		if errors.Is(err, io.EOF) {
			if r.totalBytes < 0 || r.hasReadDeclaredSize() {
				r.finish()

				if n > 0 {
					return n, nil
				}

				return 0, io.EOF
			}

			err = fmt.Errorf(
				"%w: %s ended at %d of %d bytes",
				ErrStreamTruncated, r.fileName, r.deliveredBytes, r.totalBytes,
			)
		}

		if err != nil {
			if retryErr := r.resumeAfterFailure(err); retryErr != nil {
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

func (r *ResumingReader) Close() error {
	r.isClosed = true

	r.abandonAttempt()

	return nil
}

func (r *ResumingReader) hasReadDeclaredSize() bool {
	return r.totalBytes >= 0 && r.deliveredBytes >= r.totalBytes
}

func (r *ResumingReader) finish() {
	r.isFinished = true

	r.abandonAttempt()
}

func (r *ResumingReader) openAtCurrentOffset() error {
	attemptCtx, cancelAttempt := context.WithCancel(r.streamCtx)

	body, err := r.openAtOffset(attemptCtx, r.deliveredBytes)
	if err != nil {
		cancelAttempt()

		return fmt.Errorf("open %s at offset %d: %w", r.fileName, r.deliveredBytes, err)
	}

	r.body = body
	r.attemptCancel = cancelAttempt
	r.stallTimer = time.AfterFunc(r.stallTimeout, cancelAttempt)
	r.stallTimer.Stop()

	return nil
}

func (r *ResumingReader) resumeAfterFailure(cause error) error {
	hasDeliveredBytes := r.attemptReadBytes > 0

	r.abandonAttempt()

	if err := r.streamCtx.Err(); err != nil {
		return errors.Join(cause, err)
	}

	if errors.Is(cause, ErrRangeNotHonoured) || !r.isRetryableError(cause) {
		return cause
	}

	r.retryPolicy.RegisterFailedAttempt(hasDeliveredBytes)

	if r.retryPolicy.IsExhausted() {
		return fmt.Errorf(
			"%w: %s stuck at offset %d: %w",
			ErrReadRetryBudgetExhausted, r.fileName, r.deliveredBytes, cause,
		)
	}

	r.logger.WarnContext(
		r.streamCtx,
		"storage read failed, resuming",
		slog.Int64("offset", r.deliveredBytes),
		slog.Int("attempt", r.retryPolicy.GetAttemptsSinceProgress()),
		slog.Int("max_attempts", r.retryPolicy.GetMaxAttemptsWithoutProgress()),
		slog.String("file_name", r.fileName),
		slog.Any("error", cause),
	)

	if hasDeliveredBytes {
		return nil
	}

	return r.retryPolicy.WaitBeforeRetry(r.streamCtx)
}

func (r *ResumingReader) abandonAttempt() {
	r.attemptReadBytes = 0

	if r.stallTimer != nil {
		r.stallTimer.Stop()
		r.stallTimer = nil
	}

	if r.body != nil {
		_ = r.body.Close()
		r.body = nil
	}

	if r.attemptCancel != nil {
		r.attemptCancel()
		r.attemptCancel = nil
	}
}
