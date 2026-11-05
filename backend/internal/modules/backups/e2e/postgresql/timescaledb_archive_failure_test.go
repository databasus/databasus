package postgresql

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

//nolint:tparallel // The subtests share one source server.
func TestTimescaleArchiveFailure(t *testing.T) {
	t.Parallel()
	endpoint := containers.StartTimescaleDB(t, timescaledbImage)
	for _, scenario := range []struct {
		name       string
		isShutdown bool
	}{
		{name: "preserves the archive failure when user cancellation arrives during cleanup", isShutdown: false},
		{name: "preserves the archive failure when shutdown arrives during cleanup", isShutdown: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			fixture := startBlockedTimescaleRestore(t, endpoint)
			fixture.proxy.isHolding.Store(true)
			require.NoError(t, fixture.archiveLock.Rollback())

			testwait.Until(t, "the archive fails and cleanup connects", func(context.Context) error {
				if fixture.proxy.heldCount.Load() == 0 {
					return fmt.Errorf("cleanup has not connected")
				}
				return nil
			})

			if scenario.isShutdown {
				fixture.process.Stop(t)
			} else {
				response, err := fixture.client.CancelRestoreWithResponse(t.Context(), fixture.restoreID)
				require.NoError(t, err)
				require.Equal(t, http.StatusAccepted, response.StatusCode())
				fixture.proxy.Release()
			}

			var finishedRestore apiclient.Restore
			testwait.Until(t, "the restore records its archive failure", func(ctx context.Context) error {
				response, err := fixture.team.Owner.Client.GetRestoreWithResponse(ctx, fixture.restoreID)
				if err != nil {
					return err
				}
				if response.JSON200.Status == apiclient.RestoreStatusINPROGRESS {
					return fmt.Errorf("restore is still in progress")
				}
				finishedRestore = *response.JSON200
				return nil
			})

			assert.Equal(t, apiclient.RestoreStatusFAILED, finishedRestore.Status)
			require.NotNil(t, finishedRestore.FailMessage)
			assert.Contains(t, *finishedRestore.FailMessage, "cleanup_rows")
			assert.Contains(t, *finishedRestore.FailMessage, "already exists")
			assert.NotContains(t, *finishedRestore.FailMessage, "restore interrupted by application shutdown")
			assert.Equal(
				t,
				scenario.isShutdown,
				isTimescaleTargetRestoring(t, fixture.server, fixture.targetDatabaseName),
			)

			if !scenario.isShutdown {
				fixture.process.Stop(t)
			}
			fixture.proxy.Release()
		})
	}
}
