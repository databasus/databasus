package patroni_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

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
		}
	})

	testwait.Until(t, "the child finishes starting its owned fixture", func(context.Context) error {
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
