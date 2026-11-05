package physicaltest

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func setupBastionedTeam(t *testing.T, image string) (Team, containers.BastionedDatabase) {
	t.Helper()

	team := newTeam(t)
	topology := containers.StartPhysicalPostgresBehindSSHBastion(t, image)
	team.createDatabase(
		t,
		physicalDatabaseSpec{
			Endpoint:   topology.Database,
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			Tunnel:     enginetest.NewBastionSSHTunnelSettings(topology),
		},
	)
	return team, topology
}

func RunTunnelRestore(t *testing.T, image string) {
	t.Helper()

	team, topology := setupBastionedTeam(t, image)
	target := prepareRestoreTarget(t, image)
	endpoint := enginetest.OpenForwardedEndpoint(t, topology)
	conn := openConnAt(t, endpoint.Host, endpoint.Port)
	targetTime, survivingPhases := team.seedChainAndStreamPastTarget(t, conn)
	bundle := team.downloadRestoreBundle(t, &targetTime)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: &targetTime})
	startRestoredCluster(t, target, image)

	assert.ElementsMatch(t, survivingPhases, queryRestoredMarkerRows(t, target))
}

type streamStatistics struct {
	Streamer struct {
		State            string `json:"state"`
		RestartsLastHour int    `json:"restartsLastHour"`
	} `json:"streamer"`
	Current *struct {
		SourceState      string `json:"sourceState"`
		ReceiverRestarts int    `json:"receiverRestarts"`
	} `json:"current"`
}

func (team Team) getStreamStatistics(t *testing.T) streamStatistics {
	t.Helper()

	var statistics streamStatistics
	team.call(
		t,
		physicalResponseSpec{
			Call:        apptest.PhysicalStatisticsRequest(team.Database.Id, "HOUR"),
			Status:      http.StatusOK,
			Destination: &statistics,
		},
	)
	return statistics
}

func (team Team) getCommittedWalSegmentCount(t *testing.T) int {
	t.Helper()

	segmentCount := 0
	for _, wal := range team.listBackups(t, "WAL") {
		if wal.TimelineID == 1 && wal.WalSegmentsCount != nil {
			segmentCount += *wal.WalSegmentsCount
		}
	}
	return segmentCount
}

func RunBastionOutage(t *testing.T, image string) {
	t.Helper()

	team, topology := setupBastionedTeam(t, image)
	team.enableBackups(t, true)
	team.waitForChain(t, 0)
	endpoint := enginetest.OpenForwardedEndpoint(t, topology)
	conn := openConnAt(t, endpoint.Host, endpoint.Port)
	slotName := waitForSlotPresent(t, conn)
	backupsBeforeOutage := len(team.waitForChain(t, 0))
	statisticsBeforeOutage := team.getStreamStatistics(t)

	stopTimeout := 10 * time.Second
	require.NoError(t, topology.BastionContainer.Stop(t.Context(), &stopTimeout))
	stoppedAt := time.Now()
	team.watchBastionOutage(t, stoppedAt, statisticsBeforeOutage)

	require.NoError(t, topology.BastionContainer.Start(t.Context()))
	waitForBastionAuthenticating(t, topology.Bastion)
	resumedEndpoint := enginetest.OpenForwardedEndpoint(t, topology)
	resumedConn := openConnAt(t, resumedEndpoint.Host, resumedEndpoint.Port)
	require.Contains(
		t,
		getDatabaseSlots(t, resumedConn, true),
		slotName,
		"the original WAL slot must survive the outage",
	)

	segmentsBeforeResume := team.getCommittedWalSegmentCount(t)
	generateWalActivity(t, resumedConn, 64*1024*1024)
	testwait.Until(t, "WAL archiving resumes through the bastion", func(context.Context) error {
		if team.getCommittedWalSegmentCount(t) <= segmentsBeforeResume {
			return fmt.Errorf("no new segment since the bastion resumed")
		}
		return nil
	})
	assert.Len(t, team.waitForChain(t, 0), backupsBeforeOutage, "a transport blip must not trigger another full backup")
}

func (team Team) watchBastionOutage(t *testing.T, stoppedAt time.Time, before streamStatistics) {
	t.Helper()

	observation := time.NewTimer(time.Until(stoppedAt.Add(75 * time.Second)))
	defer observation.Stop()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	hasNoticedOutage := false

	for {
		statistics := team.getStreamStatistics(t)
		require.Equal(t, "RUNNING", statistics.Streamer.State, "the streamer must stay owned throughout the outage")
		if statistics.Streamer.RestartsLastHour > before.Streamer.RestartsLastHour ||
			(statistics.Current != nil && (statistics.Current.SourceState == "UNREACHABLE" || (before.Current != nil && statistics.Current.ReceiverRestarts > before.Current.ReceiverRestarts))) {
			hasNoticedOutage = true
		}

		select {
		case <-observation.C:
			// Statistics can lag an outage until a sample completes, so the observation
			// window starts when the bastion stops.
			if !hasNoticedOutage {
				t.Log("statistics did not report the outage; observed the window from the bastion stop")
			}
			return
		case <-ticker.C:
		case <-t.Context().Done():
			t.Fatal("test ended during the bastion outage observation")
		}
	}
}

func waitForBastionAuthenticating(t *testing.T, endpoint containers.Endpoint) {
	t.Helper()

	address := net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port))
	testwait.Until(t, "the restarted bastion authenticates SSH", func(ctx context.Context) error {
		dialer := net.Dialer{Timeout: 5 * time.Second}
		conn, err := dialer.DialContext(ctx, "tcp", address)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close() }()
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return err
		}
		clientConn, channels, requests, err := ssh.NewClientConn(conn, address, &ssh.ClientConfig{
			User: containers.SSHBastionUsername, Auth: []ssh.AuthMethod{ssh.Password(containers.SSHBastionPassword)},
			// Each disposable bastion generates a new host key.
			// The disposable bastion generates a new host key for each test.
			// nosemgrep: go.lang.security.audit.crypto.insecure_ssh.avoid-ssh-insecure-ignore-host-key
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		})
		if err != nil {
			return err
		}
		return ssh.NewClient(clientConn, channels, requests).Close()
	})
}
