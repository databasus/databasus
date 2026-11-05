package iox

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type truncatingReader struct {
	payload       string
	bytesServed   int
	truncateAfter int
	failure       error
}

func (tr *truncatingReader) Read(p []byte) (int, error) {
	if tr.bytesServed >= tr.truncateAfter {
		return 0, tr.failure
	}

	n := copy(p, tr.payload[tr.bytesServed:tr.truncateAfter])
	tr.bytesServed += n

	return n, nil
}

func Test_FailureTrackingReader_WhenSourceFailsMidStream_FirstReadErrorRecorded(t *testing.T) {
	networkFailure := errors.New("connection reset by peer")
	source := &truncatingReader{
		payload:       "INSERT INTO orders VALUES (1);",
		truncateAfter: 10,
		failure:       networkFailure,
	}

	trackingReader := NewFailureTrackingReader(source)
	copiedBytes, copyErr := io.Copy(io.Discard, trackingReader)

	require.ErrorIs(t, copyErr, networkFailure)
	assert.EqualValues(t, 10, copiedBytes)
	assert.ErrorIs(t, trackingReader.GetFirstReadError(), networkFailure)
}

func Test_FailureTrackingReader_WhenSourceEndsCleanly_NoReadErrorRecorded(t *testing.T) {
	payload := "INSERT INTO orders VALUES (1);"

	trackingReader := NewFailureTrackingReader(strings.NewReader(payload))
	copiedBytes, copyErr := io.Copy(io.Discard, trackingReader)

	require.NoError(t, copyErr)
	assert.EqualValues(t, len(payload), copiedBytes)
	assert.NoError(t, trackingReader.GetFirstReadError())
}

func Test_FailureTrackingReader_WhenSourceFailsRepeatedly_KeepsFirstReadError(t *testing.T) {
	originalFailure := errors.New("connection reset by peer")
	source := &truncatingReader{truncateAfter: 0, failure: originalFailure}

	trackingReader := NewFailureTrackingReader(source)

	_, firstErr := trackingReader.Read(make([]byte, 8))
	require.ErrorIs(t, firstErr, originalFailure)

	source.failure = errors.New("file already closed")
	_, secondErr := trackingReader.Read(make([]byte, 8))
	require.Error(t, secondErr)

	assert.ErrorIs(t, trackingReader.GetFirstReadError(), originalFailure)
}
