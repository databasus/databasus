package containers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEgressDelay = 50 * time.Millisecond

func Test_AddEgressDelay_WhenAppliedToMinio_TcpHandshakeTakesAtLeastTheDelay(t *testing.T) {
	minio := StartMinioContainer(t)

	handshakeBeforeDelay, err := MeasureTCPHandshake(t, minio)
	require.NoError(t, err)
	assert.Less(t, handshakeBeforeDelay, testEgressDelay)

	AddEgressDelay(t, minio, testEgressDelay)

	handshakeAfterDelay, err := MeasureTCPHandshake(t, minio)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, handshakeAfterDelay, testEgressDelay)
}

func Test_SetEgressLoss_WhenApplied_DialTimesOut(t *testing.T) {
	minio := StartMinioContainer(t)

	SetEgressLoss(t, minio)

	_, err := MeasureTCPHandshake(t, minio)
	assert.Error(t, err)
}

func Test_ClearEgressShaping_AfterLoss_DialSucceedsAgain(t *testing.T) {
	minio := StartMinioContainer(t)
	SetEgressLoss(t, minio)

	ClearEgressShaping(t, minio)

	handshakeDuration, err := MeasureTCPHandshake(t, minio)
	require.NoError(t, err)
	assert.Less(t, handshakeDuration, testEgressDelay)
}

func Test_ClearEgressShaping_WhenCalledTwice_Succeeds(t *testing.T) {
	minio := StartMinioContainer(t)
	AddEgressDelay(t, minio, testEgressDelay)

	ClearEgressShaping(t, minio)
	ClearEgressShaping(t, minio)

	_, err := MeasureTCPHandshake(t, minio)
	assert.NoError(t, err)
}
