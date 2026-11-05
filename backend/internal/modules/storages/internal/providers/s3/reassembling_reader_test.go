package s3

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	io_utils "databasus-backend/internal/lib/iox"
	"databasus-backend/internal/modules/storages/internal/providers/providertest"
)

func TestReadChunkedStream(t *testing.T) {
	t.Parallel()

	t.Run("resumes from the offset and returns the full stream when the connection drops mid-part", func(t *testing.T) {
		t.Parallel()

		firstPart := generateBytes(8192)
		secondPart := generateBytes(4096)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": firstPart, "file.part000002": secondPart},
			bodyForAttempt:   severBodyOnAttempts(severSpec{throughAttempt: 1, breakAfterBytes: 3000}),
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001", "file.part000002"),
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, append(append([]byte{}, firstPart...), secondPart...), reassembledBytes)
		assert.Equal(
			t,
			[]string{"", "bytes=3000-8191", ""},
			transport.observedRanges,
			"the retry must resume the first part from the last byte handed out, then read the second part whole",
		)
	})

	t.Run("returns an error without corrupting the stream when the server ignores the range", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
			isRangeIgnored:   true,
			bodyForAttempt:   severBodyOnAttempts(severSpec{throughAttempt: 1, breakAfterBytes: 3000}),
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		deliveredBytes, err := io.ReadAll(reader)

		require.ErrorIs(t, err, io_utils.ErrRangeNotHonoured)
		assert.Equal(
			t,
			partBytes[:3000],
			deliveredBytes,
			"only the bytes read before the drop may reach the caller; the ignored range must not replay the part from zero",
		)
	})

	t.Run("streams the part when the backend declares no content length", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		// A chunked transfer-encoded reply carries no Content-Length, which minio-go reports as size -1.
		// Refusing that would lock out every backend that never declares a length.
		transport := &stubS3Transport{objects: map[string][]byte{"file.part000001": partBytes}}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, partBytes, reassembledBytes)
	})

	t.Run("accepts the content range as proof when a resume has no content length", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isContentRangeSent: true,
			objects:            map[string][]byte{"file.part000001": partBytes},
			bodyForAttempt:     severBodyOnAttempts(severSpec{throughAttempt: 1, breakAfterBytes: 3000}),
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err, "Content-Range proves the resume was honoured even with no declared length")

		assert.Equal(t, partBytes, reassembledBytes)
		assert.Equal(t, []string{"", "bytes=3000-8191"}, transport.observedRanges)
	})

	t.Run("returns an error when a resume is answered from the start of the object", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		// The backend reports a range it did not apply: Content-Range starts at 0, not at the offset.
		transport := &stubS3Transport{
			isContentRangeSent: true,
			isRangeIgnored:     true,
			isLengthDeclared:   true,
			objects:            map[string][]byte{"file.part000001": partBytes},
			bodyForAttempt:     severBodyOnAttempts(severSpec{throughAttempt: 1, breakAfterBytes: 3000}),
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		deliveredBytes, err := io.ReadAll(reader)

		require.ErrorIs(t, err, io_utils.ErrRangeNotHonoured)
		assert.Equal(t, partBytes[:3000], deliveredBytes)
	})

	t.Run("does not retry when a part is missing", func(t *testing.T) {
		t.Parallel()

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": generateBytes(8192)},
			statusForAttempt: func(int) int { return http.StatusNotFound },
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		_, err := io.ReadAll(reader)

		require.Error(t, err)
		assert.Equal(t, 1, transport.attempts, "a missing chunk is permanent; retrying it only delays the failure")
	})

	t.Run("cancels the attempt and resumes when the body stalls", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
			bodyForAttempt: func(attempt int, payload []byte, request *http.Request) io.ReadCloser {
				if attempt == 1 {
					return &stallingBody{requestCtx: request.Context()}
				}

				return wholeBody(payload, request)
			},
		}

		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
			stallTimeout:        50 * time.Millisecond,
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, partBytes, reassembledBytes)
		assert.Equal(t, 2, transport.attempts)
	})

	t.Run("keeps the same connection when the consumer pauses longer than the stall timeout", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
			stallTimeout:        30 * time.Millisecond,
		})

		// A restore target blocked on a slow statement stops reading for a while; that is backpressure,
		// not a stalled connection, and must not cost the stream its part.
		reassembledBytes := make([]byte, 0, len(partBytes))
		buffer := make([]byte, 4096)

		for len(reassembledBytes) < len(partBytes) {
			<-time.After(60 * time.Millisecond)

			n, err := reader.Read(buffer)
			require.NoError(t, err)

			reassembledBytes = append(reassembledBytes, buffer[:n]...)
		}

		assert.Equal(t, partBytes, reassembledBytes)
		assert.Equal(t, 1, transport.attempts, "a slow consumer must not trigger a re-open")
	})

	t.Run("completes the stream when drops repeat with progress between them", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
			bodyForAttempt: func(_ int, payload []byte, request *http.Request) io.ReadCloser {
				return &connectionBody{
					payload:         payload,
					breakAfterBytes: 1000,
					requestCtx:      request.Context(),
				}
			},
		}

		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum:        true,
			parts:                      objectSpansOf(transport.objects, "file.part000001"),
			maxAttemptsWithoutProgress: 2,
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(
			t,
			err,
			"every attempt delivered bytes, so a budget of 2 must never be spent even across 9 drops",
		)

		assert.Equal(t, partBytes, reassembledBytes)
		assert.Equal(t, 9, transport.attempts)
	})

	t.Run("resumes instead of failing when the body ends short with a plain EOF", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
			bodyForAttempt: func(attempt int, payload []byte, request *http.Request) io.ReadCloser {
				if attempt == 1 {
					return &connectionBody{
						payload:         payload[:3000],
						breakAfterBytes: 3000,
						requestCtx:      request.Context(),
					}
				}

				return wholeBody(payload, request)
			},
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err, "a truncated transfer must resume even when it arrives as a plain EOF")

		assert.Equal(t, partBytes, reassembledBytes)
		assert.Equal(t, []string{"", "bytes=3000-8191"}, transport.observedRanges)
	})

	t.Run("returns an error when the retry budget is exhausted", func(t *testing.T) {
		t.Parallel()

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": generateBytes(8192)},
			bodyForAttempt: func(_ int, payload []byte, request *http.Request) io.ReadCloser {
				return &connectionBody{payload: payload, requestCtx: request.Context()}
			},
		}

		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum:        true,
			parts:                      objectSpansOf(transport.objects, "file.part000001"),
			maxAttemptsWithoutProgress: 2,
		})

		_, err := io.ReadAll(reader)

		require.ErrorIs(t, err, io_utils.ErrReadRetryBudgetExhausted)
		assert.Equal(t, 3, transport.attempts, "the first attempt plus the two retries the budget allows")
	})

	t.Run("exhausts the budget when reopening keeps failing after a productive attempt", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
			// 501 is retryable to us but not to minio-go, so the re-open fails without its internal retries.
			statusForAttempt: func(attempt int) int {
				if attempt == 1 {
					return 0
				}

				return http.StatusNotImplemented
			},
			bodyForAttempt: severBodyOnAttempts(severSpec{throughAttempt: 1, breakAfterBytes: 1000}),
		}

		// A regression that spins without spending the budget trips this deadline instead of hanging.
		streamCtx, cancelStream := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancelStream()

		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			streamCtx:                  streamCtx,
			hasManifestChecksum:        true,
			parts:                      objectSpansOf(transport.objects, "file.part000001"),
			maxAttemptsWithoutProgress: 2,
		})

		_, err := io.ReadAll(reader)

		require.ErrorIs(
			t,
			err,
			io_utils.ErrReadRetryBudgetExhausted,
			"bytes delivered by an earlier attempt must not keep crediting later re-open failures with progress",
		)
	})

	t.Run("returns promptly without a retry when the stream context is cancelled", func(t *testing.T) {
		t.Parallel()

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": generateBytes(8192)},
			bodyForAttempt: func(_ int, payload []byte, request *http.Request) io.ReadCloser {
				return &connectionBody{payload: payload, requestCtx: request.Context()}
			},
		}

		streamCtx, cancelStream := context.WithCancel(t.Context())
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			streamCtx:           streamCtx,
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		cancelStream()

		_, err := io.ReadAll(reader)

		require.ErrorIs(t, err, context.Canceled)
		assert.Zero(t, transport.attempts, "a cancelled restore must not open another chunk")
	})

	t.Run("skips the checksum check when the file was stored without a manifest", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"legacy-object": partBytes},
		}

		// A file stored before chunking has no manifest and therefore no recorded sha256, so the span
		// carries none either; the deliberately wrong checksum here must simply never be consulted.
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			parts: []objectSpan{{
				Key:    "legacy-object",
				Size:   int64(len(partBytes)),
				SHA256: "0000000000000000000000000000000000000000000000000000000000000000",
			}},
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, partBytes, reassembledBytes)
	})

	t.Run("returns the closed error when read after close", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
		}
		reader := newTestReassemblingReader(t, transport, reassemblingReaderSpec{
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
		})

		buffer := make([]byte, 4096)
		_, err := reader.Read(buffer)
		require.NoError(t, err)

		require.NoError(t, reader.Close())

		_, err = reader.Read(buffer)
		assert.ErrorIs(t, err, io_utils.ErrReaderClosed, "a closed reader must not silently re-open the chunk")
	})

	t.Run("retries without panicking when the spec has no logger", func(t *testing.T) {
		t.Parallel()

		partBytes := generateBytes(8192)

		transport := &stubS3Transport{
			isLengthDeclared: true,
			objects:          map[string][]byte{"file.part000001": partBytes},
			bodyForAttempt:   severBodyOnAttempts(severSpec{throughAttempt: 1, breakAfterBytes: 3000}),
		}

		reader := newReassemblingReader(reassemblingReaderSpec{
			streamCtx:           t.Context(),
			coreClient:          newStubbedCoreClient(t, transport),
			bucket:              "test-bucket",
			hasManifestChecksum: true,
			parts:               objectSpansOf(transport.objects, "file.part000001"),
			retryBaseDelay:      time.Millisecond,
		})

		reassembledBytes, err := io.ReadAll(reader)
		require.NoError(t, err)

		assert.Equal(t, partBytes, reassembledBytes)
	})
}

func newTestReassemblingReader(
	t *testing.T,
	transport *stubS3Transport,
	spec reassemblingReaderSpec,
) *reassemblingReader {
	t.Helper()

	if spec.streamCtx == nil {
		spec.streamCtx = t.Context()
	}

	spec.logger = providertest.NewDiscardLogger()
	spec.coreClient = newStubbedCoreClient(t, transport)
	spec.bucket = "test-bucket"
	spec.retryBaseDelay = time.Millisecond

	return newReassemblingReader(spec)
}

// Region is set explicitly so the client never issues a GetBucketLocation the stub would have to
// answer, and path lookup keeps the request URL predictable.
func newStubbedCoreClient(t *testing.T, transport *stubS3Transport) *minio.Core {
	t.Helper()

	core, err := minio.NewCore("s3.stub.invalid", &minio.Options{
		Creds:        credentials.NewStaticV4("access-key", "secret-key", ""),
		Secure:       false,
		Region:       "us-east-1",
		BucketLookup: minio.BucketLookupPath,
		Transport:    transport,
	})
	require.NoError(t, err)

	return core
}

func objectSpansOf(objects map[string][]byte, keys ...string) []objectSpan {
	parts := make([]objectSpan, 0, len(keys))

	for _, key := range keys {
		checksum := sha256.Sum256(objects[key])
		parts = append(parts, objectSpan{
			Key:    key,
			Size:   int64(len(objects[key])),
			SHA256: hex.EncodeToString(checksum[:]),
		})
	}

	return parts
}

// The ranges are recorded so a test can assert where a resumed read picked up.
type stubS3Transport struct {
	objects            map[string][]byte
	isRangeIgnored     bool
	isContentRangeSent bool
	isLengthDeclared   bool
	statusForAttempt   func(attempt int) int
	bodyForAttempt     func(attempt int, payload []byte, request *http.Request) io.ReadCloser

	attempts       int
	observedRanges []string
}

func (transport *stubS3Transport) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.attempts++
	requestedRange := request.Header.Get("Range")

	transport.observedRanges = append(transport.observedRanges, requestedRange)

	header := http.Header{}
	header.Set("Last-Modified", time.Unix(0, 0).UTC().Format(http.TimeFormat))
	header.Set("ETag", `"stub-etag"`)

	respondWithStatus := func(status int) *http.Response {
		header.Set("Content-Length", "0")

		return &http.Response{
			StatusCode: status,
			Header:     header,
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}
	}

	if failureStatus := transport.GetFailureStatus(transport.attempts); failureStatus != 0 {
		return respondWithStatus(failureStatus), nil
	}

	objectKey := strings.TrimPrefix(request.URL.Path, "/test-bucket/")
	payload := transport.objects[objectKey]

	responseStatus := http.StatusOK

	if requestedRange != "" && !transport.isRangeIgnored {
		rangeStart, isParsed := rangeStartOf(requestedRange)
		if !isParsed {
			return respondWithStatus(http.StatusRequestedRangeNotSatisfiable), nil
		}

		payload = payload[rangeStart:]
		responseStatus = http.StatusPartialContent

		if transport.isContentRangeSent {
			header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d",
				rangeStart, rangeStart+int64(len(payload))-1, len(transport.objects[objectKey])))
		}
	}

	contentLength := int64(len(payload))

	if transport.isLengthDeclared {
		header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	} else {
		contentLength = -1
	}

	body := wholeBody(payload, request)
	if transport.bodyForAttempt != nil {
		body = transport.bodyForAttempt(transport.attempts, payload, request)
	}

	return &http.Response{
		StatusCode:    responseStatus,
		Header:        header,
		Body:          body,
		ContentLength: contentLength,
		Request:       request,
	}, nil
}

func (transport *stubS3Transport) GetFailureStatus(attempt int) int {
	if transport.statusForAttempt == nil {
		return 0
	}

	return transport.statusForAttempt(attempt)
}

func rangeStartOf(requestedRange string) (int64, bool) {
	span := strings.TrimPrefix(requestedRange, "bytes=")
	start, _, _ := strings.Cut(span, "-")

	parsed, err := strconv.ParseInt(start, 10, 64)
	if err != nil {
		return 0, false
	}

	return parsed, true
}

type severSpec struct {
	throughAttempt  int
	breakAfterBytes int
}

func severBodyOnAttempts(spec severSpec) func(int, []byte, *http.Request) io.ReadCloser {
	return func(attempt int, payload []byte, request *http.Request) io.ReadCloser {
		if attempt > spec.throughAttempt {
			return wholeBody(payload, request)
		}

		return &connectionBody{
			payload:         payload,
			breakAfterBytes: spec.breakAfterBytes,
			requestCtx:      request.Context(),
		}
	}
}

func wholeBody(payload []byte, request *http.Request) io.ReadCloser {
	return &connectionBody{
		payload:         payload,
		breakAfterBytes: len(payload),
		requestCtx:      request.Context(),
	}
}

type connectionBody struct {
	payload         []byte
	breakAfterBytes int
	position        int
	requestCtx      context.Context
}

func (b *connectionBody) Read(p []byte) (int, error) {
	if err := b.requestCtx.Err(); err != nil {
		return 0, err
	}

	severAt := min(b.breakAfterBytes, len(b.payload))

	if b.position >= severAt {
		if severAt < len(b.payload) {
			return 0, io.ErrUnexpectedEOF
		}

		return 0, io.EOF
	}

	n := copy(p, b.payload[b.position:severAt])
	b.position += n

	return n, nil
}

func (b *connectionBody) Close() error { return nil }

// stallingBody never delivers a byte and unblocks only when the request context is cancelled, which
// is what a real transport does to a pending read once the request is cancelled.
type stallingBody struct {
	requestCtx context.Context
}

func (b *stallingBody) Read(_ []byte) (int, error) {
	<-b.requestCtx.Done()

	return 0, b.requestCtx.Err()
}

func (b *stallingBody) Close() error { return nil }
