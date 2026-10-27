package physical

import (
	"context"
	"fmt"
	"testing"

	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/util/testing/containers"
)

func TestGetWalSummarizerState(t *testing.T) {
	for _, image := range supportedVersionImages {
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
						inspection := server.openInspection(t, adminRole)
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
