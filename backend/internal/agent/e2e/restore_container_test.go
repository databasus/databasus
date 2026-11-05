package e2e_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/moby/moby/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	agent_container "databasus-backend/internal/agent/container"
	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestRestoreContainer(t *testing.T) {
	t.Parallel()

	t.Run("runs without root or capabilities when restoring a backup", func(t *testing.T) {
		t.Parallel()

		endpoint := containers.StartPostgres(t, "postgres:18")
		team := enginetest.NewTeam(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		database := team.CreateDatabase(t, seedSource(t, endpoint, false))
		backup := team.BackUp(t, database.Id, apiclient.BackupEncryptionENCRYPTED)

		credentials := team.App.CreateVerificationAgent(t, team.Admin)
		engine := newEngine(t, credentials.Agent.ID)
		gate := newGateway(t, team.App.URL, gatewayOptions{isStreamHeld: true})
		agent := newAgentProcess(t, currentBinaryDirectory)
		verificationID := team.App.EnqueueVerification(t, team.Owner, backup.Id)
		agent.start(t, gate.url, credentials)
		agent.getPID(t)

		testwait.Until(
			t,
			"the agent requests the backup stream after starting its container",
			func(context.Context) error {
				select {
				case <-gate.streamRequested:
					return nil
				default:
				}

				// A verification that ended without its container, as when Docker has no network
				// left, never requests the stream, so waiting further would last until the timeout.
				verification := team.App.GetVerification(t, team.Owner, verificationID)
				if verification.Status == "COMPLETED" || verification.Status == "FAILED" {
					require.FailNow(
						t,
						"the verification ended before requesting the stream",
						fmt.Sprint(verification.FailMessage),
					)
				}

				return fmt.Errorf("the stream has not been requested")
			},
		)

		filters := agentFilter(credentials.Agent.ID)
		filters.Add("label", agent_container.LabelVerificationID+"="+verificationID.String())
		instances, err := engine.ContainerList(t.Context(), client.ContainerListOptions{Filters: filters})
		require.NoError(t, err)
		require.Len(t, instances.Items, 1)
		status := readContainerStatus(t, engine, instances.Items[0].ID)

		var hasUID, hasCapabilities bool
		for line := range strings.SplitSeq(status, "\n") {
			fields := strings.Fields(line)
			if strings.HasPrefix(line, "Uid:") {
				require.Len(t, fields, 5)
				for _, uid := range fields[1:] {
					assert.NotEqual(t, "0", uid)
				}
				hasUID = true
			}
			if strings.HasPrefix(line, "CapBnd:") {
				require.Len(t, fields, 2)
				assert.Equal(t, "0000000000000000", fields[1])
				hasCapabilities = true
			}
		}

		require.True(t, hasUID)
		require.True(t, hasCapabilities)

		gate.releaseStream()
		requireCompleted(t, waitForVerification(t, team.App, team.Owner, verificationID))
		requireNoLeftovers(t, engine, credentials.Agent.ID)
	})
}
