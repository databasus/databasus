package containers

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func (fixture *Patroni) GetCompletedCycles(t *testing.T, node PatroniNode) int {
	t.Helper()

	logs, err := node.Container.Logs(t.Context())
	require.NoError(t, err)
	defer func() { _ = logs.Close() }()

	body, err := io.ReadAll(io.LimitReader(logs, 1024*1024))
	require.NoError(t, err)

	return strings.Count(string(body), "no action. I am (")
}

func (fixture *Patroni) RequireCycles(t *testing.T, node PatroniNode, count int) {
	t.Helper()

	baseline := fixture.GetCompletedCycles(t, node)
	testwait.Until(t, "Patroni completes fresh management cycles", func(context.Context) error {
		cycles := fixture.GetCompletedCycles(t, node) - baseline
		if cycles < count {
			return fmt.Errorf("%s completed %d of %d cycles", node.Name, cycles, count)
		}

		return nil
	})
}
