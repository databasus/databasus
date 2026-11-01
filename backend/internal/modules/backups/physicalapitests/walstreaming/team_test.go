package walstreaming_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	apiclient "databasus-backend/internal/lib/apiclient"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/backups"
)

type (
	walTeam struct {
		App       *apptest.App
		Owner     apptest.Session
		Workspace apiclient.Workspace
		Database  apiclient.Database
		StorageID uuid.UUID
		Source    containers.Endpoint
		IsMtls    bool
	}
	walTeamSpec struct {
		SourceOptions  []containers.PhysicalPostgresOption
		SourceEndpoint *containers.Endpoint
		Image          string
		Timings        *backups.WalStreamTimings
		IsWithoutLoops bool
		IsMtls         bool
		IsBastioned    bool
	}
)

func newWalTeam(t *testing.T, spec walTeamSpec) *walTeam {
	t.Helper()
	image := spec.Image
	if image == "" {
		image = "postgres:17"
	}

	var source containers.Endpoint
	tunnel := apiclient.SshTunnelSettings{IsEnabled: false}
	var connectionEndpoint containers.Endpoint

	switch {
	case spec.SourceEndpoint != nil:
		source = *spec.SourceEndpoint
		connectionEndpoint = source
	case spec.IsMtls:
		source = containers.StartPhysicalPostgresMtls(t, mtlsDirectory(t))
		connectionEndpoint = source
	case spec.IsBastioned:
		topology := containers.StartPhysicalPostgresBehindSSHBastion(t, image)
		source = topology.Database
		connectionEndpoint = enginetest.OpenForwardedEndpoint(t, topology)
		tunnel = enginetest.NewBastionSSHTunnelSettings(topology)
	default:
		source = containers.StartPhysicalPostgres(t, image, spec.SourceOptions...)
		connectionEndpoint = source
	}

	timings := backups.ProductionWalStreamTimings()
	timings.ForcedRotationInterval = 2 * time.Second
	if spec.Timings != nil {
		timings = *spec.Timings
	}

	options := []apptest.Option{apptest.WithWalStreamTimings(timings)}
	if !spec.IsWithoutLoops {
		options = append(options, apptest.WithPhysicalBackupLoops())
	}

	testApp := apptest.New(t, options...)
	owner := testApp.CreateAdmin(t)
	workspace := apptest.CreateWorkspace(t, owner, "WAL streaming API")
	storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)

	var settings apiclient.DatabaseSettings
	physicalSettings := apiclient.PostgresqlPhysicalSettings{
		Type:       apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Host:       source.Host,
		Port:       source.Port,
		Username:   "testuser",
		Password:   new("testpassword"),
		SslMode:    apiclient.PostgresqlSslModeDisable,
		SshTunnel:  tunnel,
		BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM,
	}

	if spec.IsMtls {
		physicalSettings.SslMode = apiclient.PostgresqlSslModeVerifyCa
		physicalSettings.SslClientCert = new(readMtlsFile(t, "client.crt"))
		physicalSettings.SslClientKey = new(readMtlsFile(t, "client.key"))
		physicalSettings.SslRootCert = new(readMtlsFile(t, "ca.crt"))
	}

	require.NoError(t, settings.FromPostgresqlPhysicalSettings(physicalSettings))
	database := apptest.CreateDatabase(t, owner, workspace.Id, settings)
	return &walTeam{
		App:       testApp,
		Owner:     owner,
		Workspace: workspace,
		Database:  database,
		StorageID: storageID,
		Source:    connectionEndpoint,
		IsMtls:    spec.IsMtls,
	}
}

func (team *walTeam) enable(t *testing.T) {
	t.Helper()
	apptest.SaveBackupConfigWithStorage(
		t,
		team.Owner,
		apptest.StorageUse{DatabaseID: team.Database.Id, StorageID: team.StorageID},
	)
}

func (team *walTeam) disable(t *testing.T) {
	t.Helper()
	config := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: false,
			StorageId:        &team.StorageID,
			Encryption:       config.Encryption,
			Settings:         config.Settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}

func (team *walTeam) openSource(t *testing.T) *pgx.Conn {
	t.Helper()
	dsn := fmt.Sprintf(
		"postgres://testuser:testpassword@%s:%d/postgres?sslmode=disable",
		team.Source.Host,
		team.Source.Port,
	)
	if team.IsMtls {
		directory := mtlsDirectory(t)
		dsn = fmt.Sprintf(
			"host=%s port=%d user=testuser password=testpassword dbname=postgres sslmode=verify-ca sslcert=%s/client.crt sslkey=%s/client.key sslrootcert=%s/ca.crt",
			team.Source.Host,
			team.Source.Port,
			directory,
			directory,
			directory,
		)
	}

	connection, err := pgx.Connect(t.Context(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = connection.Close(context.WithoutCancel(t.Context())) })
	return connection
}

func (team *walTeam) call(t *testing.T, call apptest.PhysicalRequest, status int, destination any) {
	t.Helper()
	code, body := team.App.CallPhysicalAPI(t, team.Owner, call)
	require.Equal(t, status, code, "%s", body)
	if destination != nil {
		require.NoError(t, json.Unmarshal(body, destination))
	}
}

type liveReading struct {
	IsStreamerRunning bool `json:"isStreamerRunning"`
	Current           *struct {
		MeasuredAt           time.Time  `json:"measuredAt"`
		SourceState          *string    `json:"sourceState"`
		SourceCurrentLSN     *string    `json:"sourceCurrentLsn"`
		SlotWalStatus        *string    `json:"slotWalStatus"`
		SlotLagBytes         *int64     `json:"slotLagBytes"`
		ArchiveProgressLSN   *string    `json:"archiveProgressLsn"`
		LastArchivedWalAt    *time.Time `json:"lastArchivedWalAt"`
		UnarchivedWalBytes   *int64     `json:"unarchivedWalBytes"`
		LocalQueueBytes      int64      `json:"localQueueBytes"`
		LocalQueuePauseBytes int64      `json:"localQueuePauseBytes"`
		IsReceiverRunning    bool       `json:"isReceiverRunning"`
	} `json:"current"`
}

func (team *walTeam) live(t *testing.T) liveReading {
	t.Helper()
	var reading liveReading
	team.call(
		t,
		apptest.PhysicalLiveRequest(team.Database.Id),
		http.StatusOK,
		&reading,
	)
	return reading
}

func (team *walTeam) waitForReceiver(t *testing.T) {
	t.Helper()
	testwait.Until(t, "the live route reports a running WAL receiver", func(context.Context) error {
		reading := team.live(t)
		if !reading.IsStreamerRunning || reading.Current == nil || !reading.Current.IsReceiverRunning {
			return fmt.Errorf("receiver is not running")
		}
		return nil
	})
}

type backupRow struct {
	ID               uuid.UUID `json:"id"`
	Type             string    `json:"type"`
	Status           string    `json:"status"`
	StopLSN          string    `json:"stopLsn"`
	WalSegmentsCount *int      `json:"walSegmentsCount"`
}

func (team *walTeam) listBackups(t *testing.T) []backupRow {
	t.Helper()
	var page struct {
		Backups []backupRow `json:"items"`
	}
	team.call(
		t,
		apptest.PhysicalListRequest(team.Database.Id, "?limit=1000"),
		http.StatusOK,
		&page,
	)
	return page.Backups
}

func (team *walTeam) waitForWal(t *testing.T, minimum int) {
	t.Helper()
	testwait.Until(t, "the API reports committed WAL segments", func(context.Context) error {
		count := 0
		for _, backup := range team.listBackups(t) {
			if backup.Type == "WAL" && backup.WalSegmentsCount != nil {
				count += *backup.WalSegmentsCount
			}
		}
		if count < minimum {
			return fmt.Errorf("%d WAL segments, need %d", count, minimum)
		}
		return nil
	})
}

func generateWal(t *testing.T, source *pgx.Conn, minimum int64) {
	t.Helper()
	_, err := source.Exec(t.Context(), "CREATE TABLE IF NOT EXISTS wal_api_activity(payload text)")
	require.NoError(t, err)

	var start string
	require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&start))

	for {
		_, err = source.Exec(
			t.Context(),
			"INSERT INTO wal_api_activity SELECT repeat('x',1024) FROM generate_series(1,1000)",
		)
		require.NoError(t, err)
		var bytes int64
		require.NoError(
			t,
			source.QueryRow(t.Context(), "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(),$1::pg_lsn)::bigint", start).
				Scan(&bytes),
		)
		if bytes >= minimum {
			break
		}
	}
}

func rotateWal(t *testing.T, source *pgx.Conn) {
	t.Helper()
	generateWal(t, source, 1)
	_, err := source.Exec(t.Context(), "SELECT pg_switch_wal()")
	require.NoError(t, err)
}
