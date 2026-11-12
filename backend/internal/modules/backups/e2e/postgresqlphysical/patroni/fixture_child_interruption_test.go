package patroni_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

type patroniChildInterruption struct {
	Command   *exec.Cmd
	Cancel    context.CancelFunc
	ReadyPath string
}

func interruptPatroniChild(t *testing.T, spec patroniChildInterruption) ([]byte, error) {
	t.Helper()

	var output bytes.Buffer
	spec.Command.Stdout = &output
	spec.Command.Stderr = &output
	require.NoError(t, spec.Command.Start())

	hasWaited := false
	t.Cleanup(func() {
		spec.Cancel()
		if !hasWaited {
			_ = spec.Command.Wait()
			t.Logf("child output:\n%s", output.String())
		}
	})

	// The child builds and starts a whole Patroni cluster, which takes longer on slower CI runners.
	testwait.Within(4*time.Minute).
		Until(t, "the child finishes starting its owned fixture", func(context.Context) error {
			if _, err := os.Stat(spec.ReadyPath); err != nil {
				return fmt.Errorf("child readiness: %w", err)
			}

			return nil
		})

	spec.Cancel()
	err := spec.Command.Wait()
	hasWaited = true

	return output.Bytes(), err
}
