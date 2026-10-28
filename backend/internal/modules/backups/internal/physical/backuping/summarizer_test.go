package backuping

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_WaitWindowForCadence_WhenCronIntervalHasNoPeriod_FallsBackToCap(t *testing.T) {
	require.Equal(t, summarizerWaitCap, waitWindowForCadence(0),
		"a cron interval reports no period, and a zero window would skip the wait entirely")
}

func Test_WaitWindowForCadence_WhenHourlyCadence_UsesQuarterOfCadence(t *testing.T) {
	require.Equal(t, 15*time.Minute, waitWindowForCadence(time.Hour))
}

func Test_WaitWindowForCadence_WhenDailyCadence_ClampsToCap(t *testing.T) {
	require.Equal(t, summarizerWaitCap, waitWindowForCadence(24*time.Hour))
}
