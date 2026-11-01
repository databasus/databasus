package physical

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/clock"
)

func TestBaseBackupProgress(t *testing.T) {
	t.Parallel()
	t.Run("reports advancing size and elapsed time before the stream ends", func(t *testing.T) {
		t.Parallel()
		spec := newBaseBackupTestSpec(t)
		manualClock := clock.NewManual(time.Now())
		spec.Clock = manualClock
		gateDir := t.TempDir()
		pidPath := filepath.Join(gateDir, "pid")
		gates := []string{
			filepath.Join(gateDir, "first"),
			filepath.Join(gateDir, "second"),
			filepath.Join(gateDir, "finish"),
		}
		tarPath := writeBaseBackupTar(t)

		var script strings.Builder
		_, err := fmt.Fprintf(&script, "echo $$ > %q\n", pidPath)
		require.NoError(t, err)
		for block := range 3 {
			_, err := fmt.Fprintf(&script,
				"while ! test -f %q; do sleep 0.01; done\ndd if=%q bs=512 skip=%d count=1 2>/dev/null\n",
				gates[block],
				tarPath,
				block,
			)
			require.NoError(t, err)
		}
		_, err = script.WriteString(
			"echo 'write-ahead log start point: 0/3000060 on timeline 1' >&2\necho 'write-ahead log end point: 0/3000220' >&2",
		)
		require.NoError(t, err)

		writeBaseBackupStub(t, spec, script.String())
		type progressReport struct {
			sizeMb    float64
			elapsedMs int64
		}
		reports := make(chan progressReport, 10)
		spec.ProgressListener = func(sizeMb float64, elapsedMs int64) { reports <- progressReport{sizeMb: sizeMb, elapsedMs: elapsedMs} }

		completed := make(chan BaseBackupResult, 1)
		failures := make(chan error, 1)
		go func() {
			backup, err := NewSource(Settings{}).StreamBaseBackup(t.Context(), spec, io.Discard)
			completed <- backup
			failures <- err
		}()

		waitForToolPID(t, pidPath)
		manualClock.Advance(time.Second)
		require.NoError(t, os.WriteFile(gates[0], nil, 0o600))
		firstReport := <-reports
		assert.InEpsilon(t, float64(512)/(1024*1024), firstReport.sizeMb, 1e-9)
		assert.Equal(t, int64(1000), firstReport.elapsedMs)
		manualClock.Advance(2 * time.Second)
		require.NoError(t, os.WriteFile(gates[1], nil, 0o600))
		secondReport := <-reports
		assert.InEpsilon(t, 2*firstReport.sizeMb, secondReport.sizeMb, 1e-9)
		assert.Equal(t, int64(3000), secondReport.elapsedMs)
		require.NoError(t, os.WriteFile(gates[2], nil, 0o600))

		backup := <-completed
		require.NoError(t, <-failures)
		assert.Equal(t, int64(1536), backup.StreamedSize)

		for len(reports) > 0 {
			<-reports
		}
		manualClock.Advance(time.Hour)
		select {
		case <-reports:
			t.Fatal("progress was reported after the stream returned")
		default:
		}
	})
}
