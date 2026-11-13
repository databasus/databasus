package physicaltest

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

// CNPGCertificateRefusal selects an invalid source certificate and the diagnostic each operation reports.
type CNPGCertificateRefusal struct {
	Mode              containers.CNPGCertificateMode
	ConnectionFailure string
	BackupFailure     string
}

// RunCNPGRefusedCertificate saves a source while its certificate verifies, rotates it to one Databasus
// cannot verify and checks that the source is then never connected, backed up or streamed from.
func RunCNPGRefusedCertificate(t *testing.T, spec CNPGManagerSpec, refusal CNPGCertificateRefusal) {
	t.Helper()

	cluster := spec.Pool.StartCluster(t, containers.CNPGClusterSpec{Image: spec.ClusterImage})
	authority := cluster.Authority.CertificatePEM
	events := make(chan string, 1)
	observer := walCertificateObserver{Handler: slog.DiscardHandler, Events: events}
	team := newTeam(t, apptest.WithLogSink(observer))
	team.createDatabase(t, physicalDatabaseSpec{
		Endpoint:   cluster.Endpoint,
		BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
		Tunnel:     enginetest.NewDisabledSSHTunnelSettings(),
		Access: sourceAccess{
			Username:        containers.CNPGBackupRole,
			Password:        containers.PostgresPassword,
			SslMode:         apiclient.PostgresqlSslModeVerifyFull,
			RootCertificate: &authority,
		},
	})

	cluster.RotateServerCertificate(t, refusal.Mode)

	response, err := team.Owner.Client.TestDatabaseWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	require.False(t, response.JSON200.IsConnected)
	require.NotNil(t, response.JSON200.Error)
	t.Logf("connection test error: %s", *response.JSON200.Error)
	require.Contains(t, *response.JSON200.Error, refusal.ConnectionFailure)

	team.enableBackups(t, true)
	team.call(t, physicalResponseSpec{
		Call:   apptest.PhysicalStartRequest(team.Database.Id, "FULL"),
		Status: http.StatusAccepted,
	})
	var failure attemptFailure
	testwait.Until(t, "the requested full fails on the refused certificate", func(context.Context) error {
		team.App.QuerySQL(
			t,
			&failure,
			"SELECT status, COALESCE(error_reason,'') AS reason, COALESCE(fail_message,'') AS message FROM backups_physical WHERE database_id=? AND kind='FULL' ORDER BY created_at DESC LIMIT 1",
			team.Database.Id,
		)
		if failure.Status != "ERROR" {
			return fmt.Errorf("full backup: %+v", failure)
		}

		return nil
	})
	t.Logf("full backup failure: %s", failure.Message)
	require.Contains(t, failure.Message, refusal.BackupFailure)

	team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
	testwait.Until(t, "the WAL worker refuses the rotated certificate", func(context.Context) error {
		select {
		case failure := <-events:
			require.True(
				t,
				strings.Contains(failure, refusal.ConnectionFailure) ||
					strings.Contains(failure, refusal.BackupFailure),
				failure,
			)
			return nil
		default:
			return fmt.Errorf("no WAL connection failure observed")
		}
	})

	var archivedSegments int
	team.App.QuerySQL(
		t,
		&archivedSegments,
		"SELECT count(*) FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL",
		team.Database.Id,
	)
	require.Zero(t, archivedSegments)

	var completedBackups int
	team.App.QuerySQL(
		t,
		&completedBackups,
		"SELECT count(*) FROM backups_physical WHERE database_id=? AND status='COMPLETED'",
		team.Database.Id,
	)
	require.Zero(t, completedBackups)

	statistics, err := team.Owner.Client.GetWalStreamStatisticsWithResponse(
		t.Context(),
		team.Database.Id,
		&apiclient.GetWalStreamStatisticsParams{Period: apiclient.WalStreamStatisticsPeriodHOUR},
	)
	require.NoError(t, err)
	require.NotNil(t, statistics.JSON200, "%s", statistics.Body)
	require.NotEqual(t, apiclient.WalStreamHealthLevelHEALTHY, statistics.JSON200.Health.Level)
}
