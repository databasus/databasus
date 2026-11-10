package physicaltest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func RunForeignSourceQueuedRecovery(t *testing.T, image string) {
	t.Helper()
	team := newTeam(t)
	original := containers.StartPhysicalPostgres(t, image)
	foreign := containers.StartPhysicalPostgres(t, image)
	proxy := containers.StartPostgresProxy(t, original)
	team.createDatabase(
		t,
		physicalDatabaseSpec{
			Endpoint:   proxy.Endpoint,
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		},
	)
	source := openConnAt(t, original.Host, original.Port)
	foreignConnection := openConnAt(t, foreign.Host, foreign.Port)
	createMarkerTable(t, source)
	createMarkerTable(t, foreignConnection)
	insertMarker(t, foreignConnection, markerRow{Phase: "foreign-cluster", Payload: "must never be restored"})
	insertMarker(t, source, markerRow{Phase: "before-full", Payload: "original base backup"})
	originalIdentity := sourceSystemIdentifier(t, source)
	require.NotEqual(t, originalIdentity, sourceSystemIdentifier(t, foreignConnection))
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	storageDirectory := team.App.GetLocalStorageFilePath("")
	require.NoError(t, os.Chmod(storageDirectory, 0o500))
	t.Cleanup(func() { require.NoError(t, os.Chmod(storageDirectory, 0o755)) })
	insertMarker(t, source, markerRow{Phase: "queued-before-refusal", Payload: "requires original queued WAL"})
	generateWalActivity(t, source, 32*1024*1024)
	targetTime := targetTimeWithMargin(t, source)
	insertMarker(t, source, markerRow{Phase: "after-target", Payload: "must not be restored"})
	throughLSN := currentLSN(t, source)
	forceWalRotation(t, source)
	queueDirectory := team.App.GetWalQueueFilePath(team.Database.Id, "")
	var queuedFiles map[string][32]byte
	testwait.Until(t, "original WAL remains queued while storage refuses writes", func(context.Context) error {
		queuedFiles = readQueuedSourceWal(t, queueDirectory)
		targetWal := fmt.Sprintf("%08X%08X%08X", 1, throughLSN>>32, (throughLSN&0xffffffff)/(16*1024*1024))
		_, hasTargetWal := queuedFiles[targetWal]
		if len(queuedFiles) < 2 || !hasTargetWal {
			return fmt.Errorf("only %d completed queued segments", len(queuedFiles))
		}
		return nil
	})
	var slotName string
	team.App.QuerySQL(
		t,
		&slotName,
		"SELECT replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
		team.Database.Id,
	)
	_, err := foreignConnection.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slotName)
	require.NoError(t, err)
	generateWalActivity(t, foreignConnection, 64*1024*1024)
	// Keep SQL on A so the refusal must come from the actual replication session on B.
	proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: foreign})
	proxy.CutConnections()
	team.App.RestartModules(t)
	team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
	testwait.Until(t, "the restarted stream refuses the authenticated foreign source", func(context.Context) error {
		var reason string
		team.App.QuerySQL(
			t,
			&reason,
			"SELECT COALESCE(error_reason,'') FROM backups_physical_wal_streamers WHERE database_id=?",
			team.Database.Id,
		)
		if reason != "SYSTEM_IDENTIFIER_MISMATCH" {
			return fmt.Errorf("reason: %s", reason)
		}
		return nil
	})
	hasForeignIdentity := false
	for _, event := range proxy.GetEvents() {
		if event.Destination == foreign && event.Kind == "identified" {
			hasForeignIdentity = true
		}
		if event.Destination == foreign {
			require.NotEqual(t, "streaming", event.Kind)
		}
	}
	require.True(t, hasForeignIdentity)
	retainedFiles := readQueuedSourceWal(t, queueDirectory)
	for name, checksum := range queuedFiles {
		require.Equal(t, checksum, retainedFiles[name], name)
	}
	require.Len(t, team.listBackups(t, "FULL"), 1)
	require.NoError(t, os.Chmod(storageDirectory, 0o755))
	proxy.Route(containers.PostgresProxyRoute{SQLDestination: original, ReplicationDestination: original})
	proxy.CutConnections()
	team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
	team.waitForReplayableThroughLSN(t, chain, throughLSN)
	for _, wal := range team.listBackups(t, "WAL") {
		require.Equal(t, 1, wal.TimelineID)
	}
	var savedIdentity string
	team.App.QuerySQL(
		t,
		&savedIdentity,
		"SELECT system_identifier FROM databases_postgresql_physical WHERE database_id=?",
		team.Database.Id,
	)
	require.Equal(t, originalIdentity, savedIdentity)
	bundle := team.downloadRestoreBundle(t, &targetTime)
	target := prepareRestoreTarget(t, image)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: &targetTime})
	startRestoredCluster(t, target, image)
	require.ElementsMatch(t, []string{"before-full", "queued-before-refusal"}, queryRestoredMarkerRows(t, target))
}

func RunStableEndpointPromotion(t *testing.T, image string) {
	t.Helper()
	team := newTeam(t)
	primary, standby := containers.StartPhysicalPrimaryWithStandby(t, image)
	proxy := containers.StartPostgresProxy(t, primary)
	team.createDatabase(
		t,
		physicalDatabaseSpec{
			Endpoint:   proxy.Endpoint,
			BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
			Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		},
	)
	team.useReplicationOnlyUser(t)
	primaryConnection := openConnAt(t, primary.Host, primary.Port)
	standbyConnection := openConnAt(t, standby.Host, standby.Port)
	_, err := primaryConnection.Exec(t.Context(), "CREATE TABLE backup_permission_probe(id integer)")
	require.NoError(t, err)
	createMarkerTable(t, primaryConnection)
	insertMarker(t, primaryConnection, markerRow{Phase: "before-promotion", Payload: "original primary"})
	waitForMarkerOnStandby(t, standbyConnection, "before-promotion")
	identity := sourceSystemIdentifier(t, primaryConnection)
	require.Equal(t, identity, sourceSystemIdentifier(t, standbyConnection))
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	team.streamPostFullSegments(t, primaryConnection, walArchiveSpec{Chain: chain, Count: 1})
	promoteStandby(t, standbyConnection)
	require.Equal(t, identity, sourceSystemIdentifier(t, standbyConnection))
	proxy.Route(containers.PostgresProxyRoute{SQLDestination: standby, ReplicationDestination: standby})
	proxy.CutConnections()
	team.App.RestartModules(t)
	team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
	insertMarker(t, standbyConnection, markerRow{Phase: "after-promotion", Payload: "promoted source"})
	generateWalActivity(t, standbyConnection, 32*1024*1024)
	team.triggerIncremental(t)
	team.waitForReanchoredFull(t, chain[0].ID, 2)
	team.waitForTimelineHistory(t, 2)
	team.requireRestrictedPromotionRole(t)
	require.Equal(t, proxy.Endpoint.Host, team.Settings.Host)
	require.Equal(t, proxy.Endpoint.Port, team.Settings.Port)
	testwait.Until(t, "the promoted source enters WAL replication with the same identity", func(context.Context) error {
		for _, event := range proxy.GetEvents() {
			if event.Destination == standby && event.Kind == "streaming" {
				return nil
			}
		}
		return fmt.Errorf("no streaming session on the promoted source")
	})
	insertMarker(t, standbyConnection, markerRow{Phase: "after-replacement-full", Payload: "new timeline WAL"})
	replacement := []backupRow{team.listBackups(t, "FULL")[0]}
	team.streamPostFullSegments(t, standbyConnection, walArchiveSpec{Chain: replacement, Count: 2})
	team.waitForReplayableThroughLSN(t, replacement, currentLSN(t, standbyConnection))
	bundle := team.downloadRestoreBundle(t, nil)
	target := prepareRestoreTarget(t, image)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})
	startRestoredCluster(t, target, image)
	require.ElementsMatch(
		t,
		[]string{"before-promotion", "after-promotion", "after-replacement-full"},
		queryRestoredMarkerRows(t, target),
	)
	var live struct {
		Current *struct {
			IsReceiverRunning bool `json:"isReceiverRunning"`
		} `json:"current"`
	}
	body := team.call(
		t,
		physicalResponseSpec{Call: apptest.PhysicalLiveRequest(team.Database.Id), Status: 200, Destination: nil},
	)
	require.NoError(t, json.Unmarshal(body, &live))
	require.NotNil(t, live.Current)
	require.True(t, live.Current.IsReceiverRunning)
}

func sourceSystemIdentifier(t *testing.T, connection *pgx.Conn) string {
	t.Helper()
	var identity string
	require.NoError(
		t,
		connection.QueryRow(t.Context(), "SELECT system_identifier::text FROM pg_control_system()").Scan(&identity),
	)
	return identity
}

func readQueuedSourceWal(t *testing.T, directory string) map[string][32]byte {
	t.Helper()
	segments := make(map[string][32]byte)
	require.NoError(t, filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || len(entry.Name()) != 24 {
			return nil
		}
		body, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		segments[entry.Name()] = sha256.Sum256(body)
		return nil
	}))
	return segments
}
