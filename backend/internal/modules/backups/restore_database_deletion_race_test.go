package backups_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

//nolint:tparallel // The subtests share one source server.
func TestDeleteDatabaseRacingRestoreStart(t *testing.T) {
	t.Parallel()

	server := startRestoreServer(t, restoreServerImage)

	const roundCount = 3

	for round := range roundCount {
		t.Run(fmt.Sprintf("never deletes the database next to a restore it accepted, round %d", round),
			func(t *testing.T) {
				tm := newTeam(t)
				database := newBackedUpDatabase(t, tm, server)
				// The target stays locked, so an accepted restore cannot finish before the deletion reads it: a
				// restore that has already ended does not stand in the deletion's way.
				blockedTargetSpec, _ := server.newBlockedTargetSpec(t)
				restoreTarget := server.newRestoreTarget(t, blockedTargetSpec)

				restoreStatus := make(chan int, 1)
				startSignal := make(chan struct{})

				go func() {
					<-startSignal

					response, err := tm.owner.Client.StartRestoreWithResponse(t.Context(),
						apiclient.RestoreStart{BackupId: database.backup.Id, Target: restoreTarget})
					if err != nil {
						restoreStatus <- 0

						return
					}

					restoreStatus <- response.StatusCode()
				}()

				close(startSignal)
				deletion, err := tm.owner.Client.DeleteDatabaseWithResponse(t.Context(), database.databaseID)
				require.NoError(t, err)

				roundRestoreStatus := <-restoreStatus

				switch deletion.StatusCode() {
				case http.StatusNoContent:
					assert.Equal(t, http.StatusNotFound, roundRestoreStatus)
				case http.StatusConflict:
					requireProblem(t, deletion.StatusCode(), deletion.Body, http.StatusConflict,
						apiclient.ProblemCodeInUse)
					assert.Equal(t, http.StatusAccepted, roundRestoreStatus)
				default:
					t.Fatalf("the deletion answered %d: %s", deletion.StatusCode(), deletion.Body)
				}
			})
	}
}
