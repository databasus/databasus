package iox

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ResumingReader_ConnectionDropsMidStream_ResumesFromOffsetAndReturnsWholeFile(t *testing.T) {
	fileBytes := generateBytes(8192)
	source := &stubSource{content: fileBytes, breakAfterBytes: severOnAttempts(1, 3000)}

	streamedBytes, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes: int64(len(fileBytes)),
	}))

	require.NoError(t, err)
	assert.Equal(t, fileBytes, streamedBytes)
	assert.Equal(
		t,
		[]int64{0, 3000},
		source.observedOffsets,
		"the retry must resume from the last byte handed out",
	)
}

// Issue #714: a multi-hour rclone restore died once the drops outnumbered the budget, even though
// every one of them resumed and made progress.
func Test_ResumingReader_DropsFarExceedTheBudgetButEachDeliversBytes_CompletesStream(t *testing.T) {
	fileBytes := generateBytes(30000)
	source := &stubSource{
		content:         fileBytes,
		breakAfterBytes: func(int) int { return 1000 },
	}

	streamedBytes, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes:      int64(len(fileBytes)),
		RetryPolicySpec: ReadRetryPolicySpec{MaxAttemptsWithoutProgress: 10},
	}))

	require.NoError(t, err)
	assert.Equal(t, fileBytes, streamedBytes)
	assert.Len(
		t,
		source.observedOffsets,
		30,
		"a budget of 10 must not be spent by 30 drops that each made progress",
	)
}

func Test_ResumingReader_RepeatedFailuresWithoutProgress_ReturnsBudgetExhausted(t *testing.T) {
	fileBytes := generateBytes(4096)
	source := &stubSource{
		content:   fileBytes,
		openError: func(_ int) error { return errors.New("connection reset") },
	}

	_, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes:      int64(len(fileBytes)),
		RetryPolicySpec: ReadRetryPolicySpec{MaxAttemptsWithoutProgress: 3, BaseDelay: time.Millisecond},
	}))

	require.ErrorIs(t, err, ErrReadRetryBudgetExhausted)
	assert.Len(t, source.observedOffsets, 4)
}

func Test_ResumingReader_BodyGoesSilent_CancelsAttemptAndResumes(t *testing.T) {
	fileBytes := generateBytes(4096)
	source := &stubSource{
		content:    fileBytes,
		isStalling: func(attempt int) bool { return attempt == 1 },
	}

	streamedBytes, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes:      int64(len(fileBytes)),
		StallTimeout:    50 * time.Millisecond,
		RetryPolicySpec: ReadRetryPolicySpec{BaseDelay: time.Millisecond},
	}))

	require.NoError(t, err)
	assert.Equal(t, fileBytes, streamedBytes)
	assert.Equal(t, []int64{0, 0}, source.observedOffsets)
}

func TestResumingReader(t *testing.T) {
	t.Parallel()

	t.Run("leaves the stall timer stopped when the consumer is idle", func(t *testing.T) {
		t.Parallel()

		fileBytes := generateBytes(4096)
		source := &stubSource{content: fileBytes}

		reader := newTestResumingReader(t, source, ResumingReaderSpec{
			TotalBytes:   int64(len(fileBytes)),
			StallTimeout: 50 * time.Millisecond,
		})

		firstChunk := make([]byte, 1024)
		readBytes, err := reader.Read(firstChunk)
		require.NoError(t, err)
		require.Equal(t, 1024, readBytes)

		assert.False(t, reader.stallTimer.Stop(), "the stall timer must be stopped between consumer reads")

		remainingBytes, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, fileBytes, append(firstChunk, remainingBytes...))
		assert.Len(t, source.observedOffsets, 1, "the pause is the consumer's, not the connection's")
	})
}

func Test_ResumingReader_BodyEndsBeforeTheDeclaredSize_ResumesInsteadOfReturningShortStream(t *testing.T) {
	fileBytes := generateBytes(4096)
	source := &stubSource{content: fileBytes, truncateOnAttempts: severOnAttempts(1, 1500)}

	streamedBytes, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes:      int64(len(fileBytes)),
		RetryPolicySpec: ReadRetryPolicySpec{BaseDelay: time.Millisecond},
	}))

	require.NoError(t, err)
	assert.Equal(t, fileBytes, streamedBytes)
	assert.Equal(t, []int64{0, 1500}, source.observedOffsets)
}

func Test_ResumingReader_ResumeAnsweredFromTheStartOfTheFile_ReturnsRangeError(t *testing.T) {
	fileBytes := generateBytes(4096)
	source := &stubSource{
		content:         fileBytes,
		breakAfterBytes: severOnAttempts(1, 1500),
		isRangeIgnored:  true,
	}

	_, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes: int64(len(fileBytes)),
	}))

	require.ErrorIs(t, err, ErrRangeNotHonoured)
}

func Test_ResumingReader_SizeIsUnknown_StreamsUntilEOFWithoutCompletenessCheck(t *testing.T) {
	fileBytes := generateBytes(4096)
	source := &stubSource{content: fileBytes}

	streamedBytes, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes: -1,
	}))

	require.NoError(t, err)
	assert.Equal(t, fileBytes, streamedBytes)
	assert.Len(t, source.observedOffsets, 1)
}

func Test_ResumingReader_ErrorIsNotRetryable_FailsWithoutSpendingTheBudget(t *testing.T) {
	errFileMissing := errors.New("file does not exist")
	source := &stubSource{
		content:   generateBytes(1024),
		openError: func(int) error { return errFileMissing },
	}

	_, err := io.ReadAll(newTestResumingReader(t, source, ResumingReaderSpec{
		TotalBytes:       1024,
		IsRetryableError: func(err error) bool { return !errors.Is(err, errFileMissing) },
	}))

	require.ErrorIs(t, err, errFileMissing)
	assert.Len(t, source.observedOffsets, 1)
}

func Test_ResumingReader_StreamContextCancelled_ReturnsPromptlyWithoutRetrying(t *testing.T) {
	streamCtx, cancelStream := context.WithCancel(t.Context())
	opened := make(chan struct{})
	source := &stubSource{
		content:   generateBytes(4096),
		openError: func(int) error { close(opened); return errors.New("connection reset") },
	}

	reader := NewResumingReader(ResumingReaderSpec{
		StreamCtx:       streamCtx,
		FileName:        "backup.dump",
		OpenAtOffset:    source.OpenAtOffset,
		TotalBytes:      4096,
		RetryPolicySpec: ReadRetryPolicySpec{BaseDelay: time.Hour},
	})
	t.Cleanup(func() { _ = reader.Close() })

	go func() {
		<-opened
		cancelStream()
	}()

	readStart := time.Now()
	_, err := io.ReadAll(reader)

	require.ErrorIs(t, err, context.Canceled)
	assert.Less(t, time.Since(readStart), time.Second, "cancellation must not wait out the backoff")
}

func Test_ResumingReader_ReadAfterClose_ReturnsClosedError(t *testing.T) {
	source := &stubSource{content: generateBytes(1024)}
	reader := newTestResumingReader(t, source, ResumingReaderSpec{TotalBytes: 1024})

	require.NoError(t, reader.Close())

	_, err := reader.Read(make([]byte, 16))

	require.ErrorIs(t, err, ErrReaderClosed)
}

func newTestResumingReader(t *testing.T, source *stubSource, spec ResumingReaderSpec) *ResumingReader {
	t.Helper()

	spec.StreamCtx = t.Context()
	spec.FileName = "backup.dump"
	spec.OpenAtOffset = source.OpenAtOffset

	reader := NewResumingReader(spec)
	t.Cleanup(func() { _ = reader.Close() })

	return reader
}

type stubSource struct {
	content            []byte
	observedOffsets    []int64
	openError          func(attempt int) error
	breakAfterBytes    func(attempt int) int
	truncateOnAttempts func(attempt int) int
	isStalling         func(attempt int) bool
	isRangeIgnored     bool
}

func (s *stubSource) OpenAtOffset(ctx context.Context, offsetBytes int64) (io.ReadCloser, error) {
	attempt := len(s.observedOffsets) + 1
	s.observedOffsets = append(s.observedOffsets, offsetBytes)

	if s.openError != nil {
		if err := s.openError(attempt); err != nil {
			return nil, err
		}
	}

	servedFrom := offsetBytes
	if s.isRangeIgnored {
		servedFrom = 0
	}

	body := &stubBody{
		attemptCtx: ctx,
		remaining:  s.content[servedFrom:],
		breakAfter: -1,
		truncateAt: -1,
	}

	if s.breakAfterBytes != nil {
		body.breakAfter = s.breakAfterBytes(attempt)
	}

	if s.truncateOnAttempts != nil {
		body.truncateAt = s.truncateOnAttempts(attempt)
	}

	if s.isStalling != nil {
		body.isStalling = s.isStalling(attempt)
	}

	return body, nil
}

func severOnAttempts(throughAttempt, afterBytes int) func(int) int {
	return func(attempt int) int {
		if attempt > throughAttempt {
			return -1
		}

		return afterBytes
	}
}

type stubBody struct {
	attemptCtx   context.Context
	remaining    []byte
	deliveredNow int
	breakAfter   int
	truncateAt   int
	isStalling   bool
}

func (b *stubBody) Read(p []byte) (int, error) {
	if b.isStalling {
		<-b.attemptCtx.Done()

		return 0, b.attemptCtx.Err()
	}

	if b.breakAfter >= 0 && b.deliveredNow >= b.breakAfter {
		return 0, io.ErrUnexpectedEOF
	}

	if b.truncateAt >= 0 && b.deliveredNow >= b.truncateAt {
		return 0, io.EOF
	}

	if len(b.remaining) == 0 {
		return 0, io.EOF
	}

	readable := min(len(p), len(b.remaining))

	if b.breakAfter >= 0 {
		readable = min(readable, b.breakAfter-b.deliveredNow)
	}

	if b.truncateAt >= 0 {
		readable = min(readable, b.truncateAt-b.deliveredNow)
	}

	copy(p, b.remaining[:readable])
	b.remaining = b.remaining[readable:]
	b.deliveredNow += readable

	return readable, nil
}

func (b *stubBody) Close() error {
	return nil
}

func generateBytes(size int) []byte {
	generated := make([]byte, size)
	for i := range generated {
		generated[i] = byte(i % 251)
	}

	return generated
}
