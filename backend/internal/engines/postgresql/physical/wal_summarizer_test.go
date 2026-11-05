package physical

import (
	"context"
	"fmt"
	"testing"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // Each version's server stops before the next version starts, which bounds the memory the servers of one test hold.
func TestGetWalSummarizerState(t *testing.T) {
	t.Parallel()

	for _, image := range getSupportedVersionImages() {
		t.Run(image, func(t *testing.T) {
			for _, isEnabled := range []bool{true, false} {
				t.Run(
					fmt.Sprintf("reports process state with summarize_wal enabled %t", isEnabled),
					func(t *testing.T) {
						options := []containers.PhysicalPostgresOption{}
						if !isEnabled {
							options = append(options, containers.WithoutSummarizer())
						}
						server := startTestServer(t, image, options...)
						inspection := server.openInspection(t, getAdminRole())
						testwait.Until(t, "summarizer process matches settings", func(ctx context.Context) error {
							state, err := inspection.GetWalSummarizerState(ctx)
							if err != nil {
								return err
							}
							if state.IsEnabled != isEnabled || state.IsRunning != isEnabled {
								return fmt.Errorf("summarizer state: %+v", state)
							}
							return nil
						})
					},
				)
			}
		})
	}
}
