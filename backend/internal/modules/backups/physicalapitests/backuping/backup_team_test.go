package backuping_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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
)

type (
	backupTeam struct {
		App       *apptest.App
		Owner     apptest.Session
		Database  apiclient.Database
		StorageID uuid.UUID
		Source    containers.Endpoint
		IsMtls    bool
	}
	backupTeamSpec struct {
		Image                       string
		ClientToolsDir              string
		BackupType                  apiclient.PhysicalBackupType
		IsSummarizerDisabled        bool
		IsEncrypted                 bool
		IsMtls                      bool
		IsBastioned                 bool
		IsPhysicalReplicationDenied bool
	}
)

func newBackupTeam(t *testing.T, spec backupTeamSpec, options ...apptest.Option) *backupTeam {
	t.Helper()
	image := spec.Image
	if image == "" {
		image = "postgres:17"
	}

	var source containers.Endpoint
	if spec.IsMtls {
		source = containers.StartPhysicalPostgresMtls(t, mtlsDirectory(t))
	} else if !spec.IsBastioned {
		source = containers.StartPhysicalPostgres(t, image)
	}

	connectionEndpoint := source
	tunnel := apiclient.SshTunnelSettings{IsEnabled: false}
	if spec.IsBastioned {
		topology := containers.StartPhysicalPostgresBehindSSHBastion(t, image)
		source = topology.Database
		connectionEndpoint = enginetest.OpenForwardedEndpoint(t, topology)
		tunnel = enginetest.NewBastionSSHTunnelSettings(topology)
	}

	options = append(options, apptest.WithClientToolsDir(spec.ClientToolsDir))
	testApp := apptest.New(t, options...)
	owner := testApp.CreateAdmin(t)
	workspace := apptest.CreateWorkspace(t, owner, "Physical backup API")
	storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)

	backupType := spec.BackupType
	if backupType == "" {
		backupType = apiclient.PhysicalBackupTypeFULLINCREMENTAL
	}

	password := "testpassword"
	var databaseSettings apiclient.DatabaseSettings
	physicalSettings := apiclient.PostgresqlPhysicalSettings{
		Type:       apiclient.PostgresqlPhysicalSettingsTypePOSTGRESPHYSICAL,
		Host:       source.Host,
		Port:       source.Port,
		Username:   "testuser",
		Password:   &password,
		SslMode:    apiclient.PostgresqlSslModeDisable,
		SshTunnel:  tunnel,
		BackupType: backupType,
	}

	if spec.IsMtls {
		physicalSettings.SslMode = apiclient.PostgresqlSslModeVerifyCa
		physicalSettings.SslClientCert = new(readMtlsFile(t, "client.crt"))
		physicalSettings.SslClientKey = new(readMtlsFile(t, "client.key"))
		physicalSettings.SslRootCert = new(readMtlsFile(t, "ca.crt"))
	}

	require.NoError(t, databaseSettings.FromPostgresqlPhysicalSettings(physicalSettings))
	database := apptest.CreateDatabase(t, owner, workspace.Id, databaseSettings)

	team := &backupTeam{
		App:       testApp,
		Owner:     owner,
		Database:  database,
		StorageID: storageID,
		Source:    connectionEndpoint,
		IsMtls:    spec.IsMtls,
	}

	config := apptest.GetBackupConfig(t, owner, database.Id)
	settings, err := config.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	settings.FullBackupSchedule = apiclient.Schedule{Type: apiclient.ScheduleTypeDAILY, TimeOfDay: new("04:00")}
	if backupType != apiclient.PhysicalBackupTypeFULL {
		settings.IncrementalBackupSchedule = &apiclient.Schedule{Type: apiclient.ScheduleTypeHOURLY}
	}

	var backupSettings apiclient.BackupSettings
	require.NoError(t, backupSettings.FromPhysicalBackupSettings(settings))

	encryption := apiclient.BackupEncryptionNONE
	if spec.IsEncrypted {
		encryption = apiclient.BackupEncryptionENCRYPTED
	}

	response, err := owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &storageID,
			Encryption:       encryption,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)

	if spec.IsSummarizerDisabled {
		connection := team.openSource(t)
		_, err := connection.Exec(t.Context(), "ALTER SYSTEM SET summarize_wal=off")
		require.NoError(t, err)
		_, err = connection.Exec(t.Context(), "SELECT pg_reload_conf()")
		require.NoError(t, err)
	}
	if spec.IsPhysicalReplicationDenied {
		team.denyPhysicalReplication(t)
	}

	return team
}

func (team *backupTeam) openSource(t *testing.T) *pgx.Conn {
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
	t.Cleanup(func() { _ = connection.Close(t.Context()) })

	return connection
}

type backupRow struct {
	CreatedAt        time.Time  `json:"createdAt"`
	ID               uuid.UUID  `json:"id"`
	Type             string     `json:"type"`
	Status           string     `json:"status"`
	StartLSN         string     `json:"startLsn"`
	StopLSN          string     `json:"stopLsn"`
	RootFullBackupID *uuid.UUID `json:"rootFullBackupId"`
	FailMessage      *string    `json:"failMessage"`
}

func (team *backupTeam) listBackups(t *testing.T) []backupRow {
	t.Helper()
	status, body := team.App.CallPhysicalAPI(
		t,
		team.Owner,
		apptest.PhysicalListRequest(team.Database.Id, "?limit=1000"),
	)
	require.Equal(t, http.StatusOK, status, "%s", body)
	var page struct {
		Backups []backupRow `json:"items"`
	}
	require.NoError(t, json.Unmarshal(body, &page))
	return page.Backups
}

func (team *backupTeam) trigger(t *testing.T, backupType string) {
	t.Helper()
	status, body := team.App.CallPhysicalAPI(
		t,
		team.Owner,
		apptest.PhysicalStartRequest(team.Database.Id, strings.ToUpper(backupType)),
	)
	require.Equal(t, http.StatusAccepted, status, "%s", body)
}

func (team *backupTeam) runBackup(t *testing.T, backupType string) backupRow {
	t.Helper()
	before := map[uuid.UUID]bool{}
	for _, backup := range team.listBackups(t) {
		before[backup.ID] = true
	}

	team.trigger(t, backupType)
	team.App.RunJobPass(t, "backups_physical_scheduler")

	var terminal backupRow
	testwait.Until(t, "the scheduler publishes a terminal backup", func(context.Context) error {
		for _, backup := range team.listBackups(t) {
			if !before[backup.ID] && backup.Status != "IN_PROGRESS" {
				terminal = backup
				return nil
			}
		}
		return fmt.Errorf("no new terminal backup")
	})

	team.App.WaitForBackupRuns(t)

	return terminal
}

func (team *backupTeam) assertNoRunningBackup(t *testing.T) {
	t.Helper()
	var count int64
	team.App.QuerySQL(
		t,
		&count,
		"SELECT count(*) FROM backups_physical WHERE status = 'IN_PROGRESS' AND database_id=?",
		team.Database.Id,
	)
	require.Zero(t, count)
	if team.Source.Port != 0 {
		source := team.openSource(t)
		testwait.Until(t, "the source drops the backup replication slots", func(ctx context.Context) error {
			var slots int
			if err := source.QueryRow(ctx, "SELECT count(*) FROM pg_replication_slots WHERE starts_with(slot_name,'databasus_basebackup_')").
				Scan(&slots); err != nil {
				return err
			}
			if slots != 0 {
				return fmt.Errorf("%d backup replication slots are still present", slots)
			}
			return nil
		})
	}
}
