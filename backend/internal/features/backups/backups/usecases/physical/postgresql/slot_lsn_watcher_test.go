package usecases_physical_postgresql

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
)

func streamingSample(restartLSN walmath.LSN, observedAt time.Time) slotLivenessSample {
	return slotLivenessSample{
		RestartLSN:        restartLSN,
		LagBytes:          16 * 1024 * 1024,
		IsReceiverRunning: true,
		ObservedAt:        observedAt,
	}
}

func Test_RecordSampleAndDetectStall_WhenFirstSample_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute),
		"the first sample only arms the clock; it can never be a stall")
}

func Test_RecordSampleAndDetectStall_WhenRestartLsnAdvances_ReArmsAndDoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.False(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(200), base.Add(2*time.Minute)), time.Minute),
		"a changed restart_lsn means progress — the advance clock must reset")
}

func Test_RecordSampleAndDetectStall_WhenFrozenWithinTimeout_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.False(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(30*time.Second)), time.Minute),
		"a frozen restart_lsn within the stall timeout is not yet a stall")
}

func Test_RecordSampleAndDetectStall_WhenFrozenPastTimeout_DetectsStallThenReArms(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.True(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(90*time.Second)), time.Minute),
		"a frozen restart_lsn past the stall timeout must trigger a restart")

	require.False(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(2*time.Minute)), time.Minute),
		"after firing, the clock re-arms so we restart at most once per window")
	require.True(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(4*time.Minute)), time.Minute),
		"a sustained stall fires again only after another full timeout window")
}

func Test_RecordSampleAndDetectStall_WhenLsnFrozenAndNoLag_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	idleSample := func(observedAt time.Time) slotLivenessSample {
		return slotLivenessSample{
			RestartLSN:        walmath.LSN(100),
			LagBytes:          0,
			IsReceiverRunning: true,
			ObservedAt:        observedAt,
		}
	}

	require.False(t, tracker.recordSampleAndDetectStall(idleSample(base), time.Minute))
	require.False(t, tracker.recordSampleAndDetectStall(idleSample(base.Add(10*time.Minute)), time.Minute),
		"an idle database parks restart_lsn legitimately — there is nothing left to flush")
}

func Test_RecordSampleAndDetectStall_WhenLsnFrozenWithLag_DetectsStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	require.False(t, tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base), time.Minute))
	require.True(t,
		tracker.recordSampleAndDetectStall(streamingSample(walmath.LSN(100), base.Add(90*time.Second)), time.Minute),
		"WAL is waiting and restart_lsn has not moved — the receiver is wedged")
}

func Test_RecordSampleAndDetectStall_WhenReceiverNotRunning_DoesNotDetectStall(t *testing.T) {
	var tracker stallTracker

	base := time.Now().UTC()

	stoppedSample := func(observedAt time.Time) slotLivenessSample {
		return slotLivenessSample{
			RestartLSN:        walmath.LSN(100),
			LagBytes:          16 * 1024 * 1024,
			IsReceiverRunning: false,
			ObservedAt:        observedAt,
		}
	}

	require.False(t, tracker.recordSampleAndDetectStall(stoppedSample(base), time.Minute))
	require.False(t, tracker.recordSampleAndDetectStall(stoppedSample(base.Add(10*time.Minute)), time.Minute),
		"a receiver we are deliberately holding down is not a stalled one")
}

func newProbeOnlySupervisor(
	t *testing.T,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
) *WalStreamSupervisor {
	t.Helper()

	return NewWalStreamSupervisor(WalStreamSpec{
		DatabaseID:   uuid.New(),
		SourceDB:     sourceDB,
		WatchDirRoot: t.TempDir(),
		Logger:       logger.GetLogger(),
	})
}

func Test_ProbeSlot_WhenSlotRemovedOnReachableSource_ReportsSlotMissing(t *testing.T) {
	fixture := SetupPhysicalDBForBackup(t)
	adminConn := OpenAdminConn(t, fixture)

	sourceDB := fixture.SourceDB
	sourceDB.WalStreamSlotName = "databasus_probe_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	_, err := adminConn.Exec(
		t.Context(),
		"SELECT pg_create_physical_replication_slot($1, true)",
		sourceDB.WalStreamSlotName,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = adminConn.Exec(
			context.Background(),
			"SELECT pg_drop_replication_slot($1)",
			sourceDB.WalStreamSlotName,
		)
	})

	supervisor := newProbeOnlySupervisor(t, sourceDB)

	probe := supervisor.probeSlot(t.Context(), logger.GetLogger())
	require.Equal(t, slotProbeOutcomeReachable, probe.Outcome)
	require.NotNil(t, probe.State)

	DropReplicationSlotExternally(t, adminConn, sourceDB.WalStreamSlotName)

	probe = supervisor.probeSlot(t.Context(), logger.GetLogger())
	require.Equal(t, slotProbeOutcomeSlotMissing, probe.Outcome,
		"a source that answers without the slot is not unreachable: the chain is already broken")
	require.Nil(t, probe.State)
}

func Test_ProbeSlot_WhenSourceAcceptsButNeverAnswers_ReportsUnreachableWithinTimeout(t *testing.T) {
	silentListener := startSilentListener(t)

	supervisor := newProbeOnlySupervisor(t, databases.PostgresqlPhysicalConnectionSettings{
		Engine: pgphysical.Settings{
			Host:     "127.0.0.1",
			Port:     silentListener.Addr().(*net.TCPAddr).Port,
			Username: "streamer",
			Password: "streamer",
		},
		WalStreamSlotName: "databasus_silent_source",
	})

	probeStartedAt := time.Now().UTC()
	probe := supervisor.probeSlot(t.Context(), logger.GetLogger())

	require.Equal(t, slotProbeOutcomeUnreachable, probe.Outcome)
	require.Less(t, time.Since(probeStartedAt), 11*time.Second,
		"a silent source must not hold the lag monitor's tick past the probe timeout")
}

// Accepts every connection and holds it open without ever writing a byte, the
// way a hung server or a black-holing middlebox behaves.
func startSilentListener(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var (
		acceptedConnsMutex sync.Mutex
		acceptedConns      []net.Conn
	)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			acceptedConnsMutex.Lock()
			acceptedConns = append(acceptedConns, conn)
			acceptedConnsMutex.Unlock()
		}
	}()

	t.Cleanup(func() {
		_ = listener.Close()

		acceptedConnsMutex.Lock()
		defer acceptedConnsMutex.Unlock()

		for _, conn := range acceptedConns {
			_ = conn.Close()
		}
	})

	return listener
}

func TestStandbySlotInspection(t *testing.T) {
	for _, image := range []string{"postgres:17", "postgres:18"} {
		t.Run(image, func(t *testing.T) {
			t.Run("reports unreachable when the source is in recovery", func(t *testing.T) {
				_, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
				sourceDB := databases.PostgresqlPhysicalConnectionSettings{
					Engine: pgphysical.Settings{
						Host: standby.Host, Port: standby.Port,
						Username: containers.PostgresUsername, Password: containers.PostgresPassword,
						TLS: pgshared.TLS{Mode: pgshared.SslModeDisable},
					},
					WalStreamSlotName: "databasus_standby_" + strings.ReplaceAll(uuid.NewString(), "-", ""),
				}
				inspection := OpenTestInspection(t, sourceDB)
				require.NoError(t, inspection.CreateSlot(t.Context(), sourceDB.WalStreamSlotName))
				slot, err := inspection.GetSlot(t.Context(), sourceDB.WalStreamSlotName)
				require.NoError(t, err)
				require.NotNil(t, slot)
				require.Nil(t, slot.CurrentLSN)

				supervisor := newProbeOnlySupervisor(t, sourceDB)
				probe := supervisor.probeSlot(t.Context(), logger.GetLogger())

				require.Equal(t, slotProbeOutcomeUnreachable, probe.Outcome)
				require.Nil(t, probe.State)

				isTerminated, err := supervisor.terminateOwnedSlotBackend(t.Context(), logger.GetLogger())
				require.ErrorIs(t, err, errSourceWalPositionUnavailable)
				require.False(t, isTerminated)

				isActive, canInspect := supervisor.isSlotActive(t.Context(), logger.GetLogger())
				require.False(t, isActive)
				require.False(t, canInspect)
			})
		})
	}
}
