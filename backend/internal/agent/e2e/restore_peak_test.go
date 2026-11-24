package e2e_test

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/moby/moby/api/pkg/stdcopy"
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

const (
	// restorePeakReserveMb is the reserve the restore peak holds for WAL and the empty cluster.
	restorePeakReserveMb = 1536
	// The sort of each index outgrows the default maintenance_work_mem of 64 MB, so each build spills
	// to disk. The tables are unlogged so the source's 512 MB tmpfs holds them without their WAL.
	restorePeakIndexedRows = 1_500_000
	restorePeakTableCount  = 4
	// The index builds of a four-job restore overlap for a second or two, so the disk is sampled
	// faster than testwait polls.
	restorePeakSampleInterval = 100 * time.Millisecond
)

func TestRestorePeak(t *testing.T) {
	t.Parallel()

	t.Run("stays within the restore peak of the largest indexes when restoring with four jobs", func(t *testing.T) {
		t.Parallel()

		endpoint := containers.StartPostgres(t, "postgres:18")
		team := enginetest.NewTeam(t, apptest.WithAgentBinariesDir(currentBinaryDirectory))
		database := team.CreateDatabase(t, seedSourceWithStatements(t, endpoint, getIndexedTablesStatements()))
		backup := team.BackUp(t, database.Id, apiclient.BackupEncryptionNONE)
		largestIndexSizesMb := getRecordedLargestIndexSizesMb(t, team.App, backup)
		require.Len(t, largestIndexSizesMb, restorePeakTableCount)

		credentials := team.App.CreateVerificationAgent(t, team.Admin)
		engine := newEngine(t, credentials.Agent.ID)
		gate := newGateway(t, team.App.URL, gatewayOptions{isClaimHeld: false, isStreamHeld: false, diskCapMb: 0})
		agent := newAgentProcess(t, currentBinaryDirectory)
		agent.capacity.maxCPU = restorePeakTableCount
		agent.capacity.maxRAMMb = 4096
		verificationID := team.App.EnqueueVerification(t, team.Owner, backup.Id)
		agent.start(t, gate.url, credentials)

		observedPeakBytes := sampleRestoreDiskPeakBytes(t, restoreDiskSampling{
			app:            team.App,
			member:         team.Owner,
			engine:         engine,
			agentID:        credentials.Agent.ID,
			verificationID: verificationID,
		})

		requireCompleted(t, team.App.GetVerification(t, team.Owner, verificationID))

		observedPeakMb := float64(observedPeakBytes) / (1024 * 1024)
		indexesMb := sumFloats(largestIndexSizesMb)
		restorePeakMb := backup.SizeMb + backup.RawDatabaseSizeMb + restorePeakReserveMb + indexesMb
		t.Logf(
			"observed peak %.1f MB: archive %.1f MB, database %.1f MB, four largest indexes %.1f MB, "+
				"above archive and database %.1f MB",
			observedPeakMb, backup.SizeMb, backup.RawDatabaseSizeMb, indexesMb,
			observedPeakMb-backup.SizeMb-backup.RawDatabaseSizeMb,
		)
		assert.LessOrEqual(t, observedPeakMb, restorePeakMb)
	})
}

func getIndexedTablesStatements() string {
	var statements strings.Builder

	for tableNumber := 1; tableNumber <= restorePeakTableCount; tableNumber++ {
		fmt.Fprintf(&statements, `CREATE UNLOGGED TABLE public.indexed_%[1]d (payload integer NOT NULL);
INSERT INTO public.indexed_%[1]d SELECT (random() * 2000000000)::integer FROM generate_series(1, %[2]d);
CREATE INDEX indexed_%[1]d_payload ON public.indexed_%[1]d (payload);
`, tableNumber, restorePeakIndexedRows)
	}

	return statements.String()
}

func getRecordedLargestIndexSizesMb(t *testing.T, app *apptest.App, backup apiclient.LogicalBackup) []float64 {
	t.Helper()

	var recordedIndexSizesMb []pq.Float64Array
	app.QuerySQL(t, &recordedIndexSizesMb,
		"SELECT backup_largest_index_sizes_mb FROM backups_logical WHERE id = ?", backup.Id)
	require.Len(t, recordedIndexSizesMb, 1)

	return recordedIndexSizesMb[0]
}

type restoreDiskSampling struct {
	app            *apptest.App
	member         apptest.Session
	engine         *client.Client
	agentID        uuid.UUID
	verificationID uuid.UUID
}

// sampleRestoreDiskPeakBytes measures the restore container's disk the way the agent's disk watcher
// does, until the verification reaches an outcome, and returns the largest sample.
func sampleRestoreDiskPeakBytes(t *testing.T, sampling restoreDiskSampling) int64 {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), testwait.DefaultLimit)
	defer cancel()

	ticker := time.NewTicker(restorePeakSampleInterval)
	defer ticker.Stop()

	var peakBytes int64

	for {
		select {
		case <-ctx.Done():
			require.FailNow(t, "the verification did not reach an outcome while its disk was sampled")
		case <-ticker.C:
		}

		if usedBytes, isSampled := readRestoreDiskUsageBytes(ctx, sampling); isSampled {
			peakBytes = max(peakBytes, usedBytes)
		}

		status := sampling.app.GetVerification(t, sampling.member, sampling.verificationID).Status
		if status == "COMPLETED" || status == "FAILED" {
			require.Positive(t, peakBytes, "no disk sample was taken while the restore ran")

			return peakBytes
		}
	}
}

// readRestoreDiskUsageBytes reports false while the container does not exist yet or is going away,
// because those moments say nothing about the restore's peak.
func readRestoreDiskUsageBytes(ctx context.Context, sampling restoreDiskSampling) (int64, bool) {
	filters := agentFilter(sampling.agentID)
	filters.Add("label", agent_container.LabelVerificationID+"="+sampling.verificationID.String())

	instances, err := sampling.engine.ContainerList(ctx, client.ContainerListOptions{Filters: filters})
	if err != nil || len(instances.Items) != 1 {
		return 0, false
	}

	execution, err := sampling.engine.ExecCreate(ctx, instances.Items[0].ID, client.ExecCreateOptions{
		Cmd:          []string{"du", "-sb", "/var/lib/postgresql", "/tmp"},
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return 0, false
	}

	attached, err := sampling.engine.ExecAttach(ctx, execution.ID, client.ExecAttachOptions{})
	if err != nil {
		return 0, false
	}
	defer attached.Close()

	var output, errorOutput bytes.Buffer
	if _, err := stdcopy.StdCopy(&output, &errorOutput, attached.Reader); err != nil {
		return 0, false
	}

	var usedBytes int64

	for line := range strings.SplitSeq(strings.TrimSpace(output.String()), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			return 0, false
		}

		directoryBytes, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			return 0, false
		}

		usedBytes += directoryBytes
	}

	return usedBytes, true
}

func sumFloats(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}

	return total
}
