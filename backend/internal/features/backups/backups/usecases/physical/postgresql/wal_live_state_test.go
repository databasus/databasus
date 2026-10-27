package usecases_physical_postgresql

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/util/logger"
)

const liveStateQueuedStartLSN = walmath.LSN(0x40000000)

func Test_GetLiveState_WhenSourceProbed_ReportsValuesOfNewestProbe(t *testing.T) {
	supervisor := newLiveStateTestSupervisor(t)
	probedAt := time.Now().UTC().Add(-7 * time.Second)

	supervisor.latestProbe.Store(&timedSlotProbe{
		probe: slotProbe{Outcome: slotProbeOutcomeReachable, State: &pgphysical.Slot{
			WalStatus:  "reserved",
			LagBytes:   new(int64(1024 * 1024)),
			CurrentLSN: new(walmath.LSN(liveStateQueuedStartLSN + 5*1024*1024)),
		}},
		probedAt: probedAt,
	})

	liveState := supervisor.GetLiveState(time.Now().UTC())

	require.NotNil(t, liveState.SourceProbedAt)
	assert.Equal(t, probedAt, *liveState.SourceProbedAt, "source values carry the time of the probe they come from")
	assert.Equal(t, physical_enums.PhysicalWalStreamSourceStateReachable, *liveState.SourceState)
	assert.Equal(t, int64(1024*1024), *liveState.SlotLagBytes)
	assert.Equal(t, "reserved", *liveState.SlotWalStatus)
	assert.Equal(t, liveStateQueuedStartLSN, *liveState.ArchiveProgressLSN,
		"the lowest queued segment holds the progress point back, as in a sample")
	assert.Equal(t, int64(5*1024*1024), *liveState.UnarchivedWalBytes)
}

func Test_GetLiveState_WhenArchiveProgressPassedOlderProbe_ReportsNothingUnsaved(t *testing.T) {
	supervisor := newLiveStateTestSupervisor(t)
	supervisor.latestProbe.Store(&timedSlotProbe{
		probe: slotProbe{Outcome: slotProbeOutcomeReachable, State: &pgphysical.Slot{
			LagBytes:   new(int64(0)),
			WalStatus:  "reserved",
			CurrentLSN: new(walmath.LSN(liveStateQueuedStartLSN - 1024)),
		}},
		probedAt: time.Now().UTC().Add(-9 * time.Second),
	})

	liveState := supervisor.GetLiveState(time.Now().UTC())

	require.NotNil(t, liveState.UnarchivedWalBytes,
		"segments saved after the probe must not make the unsaved amount unknown")
	assert.Equal(t, int64(0), *liveState.UnarchivedWalBytes)
}

func Test_GetLiveState_WhenSourceUnreachable_ReportsStateWithoutSourceValues(t *testing.T) {
	supervisor := newLiveStateTestSupervisor(t)
	supervisor.latestProbe.Store(&timedSlotProbe{
		probe:    slotProbe{Outcome: slotProbeOutcomeUnreachable},
		probedAt: time.Now().UTC(),
	})

	liveState := supervisor.GetLiveState(time.Now().UTC())

	assert.Equal(t, physical_enums.PhysicalWalStreamSourceStateUnreachable, *liveState.SourceState)
	assert.Nil(t, liveState.SlotLagBytes)
	assert.Nil(t, liveState.SourceCurrentLSN)
	assert.Nil(t, liveState.UnarchivedWalBytes)
}

func Test_GetLiveState_WhenRead_LeavesRestartCountsForTheNextSample(t *testing.T) {
	supervisor := newLiveStateTestSupervisor(t)
	supervisor.receiverRespawns.recordRespawn(false)

	supervisor.GetLiveState(time.Now().UTC())

	restarts, _ := supervisor.receiverRespawns.takeCounts()
	assert.Equal(t, 1, restarts, "a live reading must not consume what the sample reports")
}

func newLiveStateTestSupervisor(t *testing.T) *WalStreamSupervisor {
	t.Helper()

	supervisor := NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID: uuid.New(),
		SourceDB: databases.PostgresqlPhysicalConnectionSettings{
			Engine:            pgphysical.Settings{},
			WalStreamSlotName: "databasus_live_test",
		},
		WalSegmentRepo: physical_repositories.GetWalSegmentRepository(),
		WatchDirRoot:   t.TempDir(),
		Logger:         logger.GetLogger(),
	})
	supervisor.setLowestQueuedSegment([]queuedWalSegment{{timelineID: 1, startLSN: liveStateQueuedStartLSN}})

	return supervisor
}
