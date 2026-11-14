package physicaltest

import (
	"archive/tar"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
)

func RunPgBouncerTargetTime(t *testing.T, image string) {
	t.Helper()
	pooler := containers.StartPhysicalPostgresThroughPgBouncer(t, image)
	team := newTeam(t)
	team.createDatabase(t, physicalDatabaseSpec{
		Endpoint: pooler.Endpoint, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
		Tunnel: enginetest.NewDisabledSSHTunnelSettings(), Access: sourceAccess{},
	})
	limitedUser := team.CreateLimitedUser(t, team.Database)
	pooler.AddCredential(
		t,
		containers.PgBouncerCredential{Username: limitedUser.Username, Password: limitedUser.Password},
	)
	team.Settings.Username = limitedUser.Username
	team.Settings.Password = &limitedUser.Password
	team.Database = team.UpdateDatabaseSettings(t, team.Database, team.Settings)
	source := openConnAt(t, pooler.Host, pooler.Port)
	team.requireRestrictedRoleAttributes(t, source)

	targetTime, survivingPhases := team.seedChainAndStreamPastTarget(t, source)
	chain := team.waitForChain(t, 2)

	require.Len(t, chain, 3)
	team.streamPostFullSegments(t, source, walArchiveSpec{Chain: chain, Count: 2})
	team.requirePgBouncerChainLinks(t, chain)

	bundle := team.downloadRestoreBundle(t, &targetTime)

	requirePgBouncerRestoreArtifacts(t, bundle)

	target := prepareRestoreTarget(t, image)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: &targetTime})
	startRestoredCluster(t, target, image)

	require.ElementsMatch(t, survivingPhases, queryRestoredMarkerRows(t, target))
}

func (team Team) requirePgBouncerChainLinks(t *testing.T, chain []backupRow) {
	t.Helper()
	var response struct {
		Backups []struct {
			ID                        uuid.UUID  `json:"id"`
			RootFullBackupID          *uuid.UUID `json:"rootFullBackupId"`
			ParentIncrementalBackupID *uuid.UUID `json:"parentIncrementalBackupId"`
		} `json:"items"`
	}
	team.call(t, physicalResponseSpec{
		Call:   apptest.PhysicalListRequest(team.Database.Id, "?limit=1000&type=INCREMENTAL"),
		Status: http.StatusOK, Destination: &response,
	})
	require.Len(t, response.Backups, 2)
	for _, incremental := range response.Backups {
		require.NotNil(t, incremental.RootFullBackupID)
		require.Equal(t, chain[0].ID, *incremental.RootFullBackupID)

		if incremental.ID == chain[1].ID {
			require.Nil(t, incremental.ParentIncrementalBackupID)
		} else {
			require.Equal(t, chain[2].ID, incremental.ID)
			require.NotNil(t, incremental.ParentIncrementalBackupID)
			require.Equal(t, chain[1].ID, *incremental.ParentIncrementalBackupID)
		}
	}
}

func requirePgBouncerRestoreArtifacts(t *testing.T, bundle string) {
	t.Helper()
	file, err := os.Open(bundle)
	require.NoError(t, err)
	defer func() { _ = file.Close() }()

	archive := tar.NewReader(file)
	manifests := make([]string, 0, 3)
	backups := make([]string, 0, 3)
	walCount := 0
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}

		require.NoError(t, err)

		if strings.HasSuffix(header.Name, "/backup_manifest") {
			manifests = append(manifests, header.Name)
		}

		if strings.Contains(header.Name, "/base.tar") {
			backups = append(backups, strings.Split(header.Name, "/")[0])
		}

		if strings.HasPrefix(header.Name, "wal/") {
			walCount++
		}
	}

	require.ElementsMatch(
		t,
		[]string{"full/backup_manifest", "incr-1/backup_manifest", "incr-2/backup_manifest"},
		manifests,
	)
	require.ElementsMatch(t, []string{"full", "incr-1", "incr-2"}, backups)
	require.GreaterOrEqual(t, walCount, 2)
}
