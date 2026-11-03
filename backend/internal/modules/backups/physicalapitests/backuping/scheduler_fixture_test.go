package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func newBackupState(t *testing.T) *backupTeam {
	t.Helper()
	testApp := apptest.New(t)
	owner := testApp.CreateAdmin(t)
	workspace := apptest.CreateWorkspace(t, owner, "Physical scheduler states")
	databaseID := testApp.InsertPhysicalDatabase(t, workspace.Id)
	storageID := testApp.CreateLocalStorage(t, owner, workspace.Id)

	response, err := owner.Client.GetDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())
	team := &backupTeam{App: testApp, Owner: owner, Database: *response.JSON200, StorageID: storageID}
	apptest.SaveBackupConfigWithStorage(t, owner, apptest.StorageUse{DatabaseID: databaseID, StorageID: storageID})

	return team
}

// addDatabase returns a second team on the same application, workspace and storage, so one scheduler
// pass evaluates both databases.
func (team *backupTeam) addDatabase(t *testing.T) *backupTeam {
	t.Helper()
	databaseID := team.App.InsertPhysicalDatabase(t, team.Database.WorkspaceId)

	response, err := team.Owner.Client.GetDatabaseWithResponse(t.Context(), databaseID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode())

	apptest.SaveBackupConfigWithStorage(
		t,
		team.Owner,
		apptest.StorageUse{DatabaseID: databaseID, StorageID: team.StorageID},
	)

	return &backupTeam{App: team.App, Owner: team.Owner, Database: *response.JSON200, StorageID: team.StorageID}
}

func newIdleBackupState(t *testing.T) *backupTeam {
	t.Helper()
	team := newBackupState(t)
	team.makeIdle(t)

	return team
}

func (team *backupTeam) addIdleDatabase(t *testing.T) *backupTeam {
	t.Helper()
	addedTeam := team.addDatabase(t)
	addedTeam.makeIdle(t)

	return addedTeam
}

func (team *backupTeam) makeIdle(t *testing.T) {
	t.Helper()
	team.setSchedules(t, true)
	team.seedFull(t, schedulerFullSpec{Age: 2 * time.Hour})
}

func (team *backupTeam) listNewBackups(t *testing.T, listedBefore []backupRow) []backupRow {
	t.Helper()
	knownBackupIDs := make(map[uuid.UUID]bool, len(listedBefore))
	for _, backup := range listedBefore {
		knownBackupIDs[backup.ID] = true
	}

	var newBackups []backupRow
	for _, backup := range team.listBackups(t) {
		if !knownBackupIDs[backup.ID] {
			newBackups = append(newBackups, backup)
		}
	}

	return newBackups
}

type schedulerFullSpec struct {
	Status           string
	Reason           *string
	Age              time.Duration
	Segment          int
	ManifestFileName *string
}

func (team *backupTeam) seedFull(t *testing.T, spec schedulerFullSpec) apptest.PhysicalFullBackupSeed {
	t.Helper()
	status := spec.Status
	if status == "" {
		status = "COMPLETED"
	}
	timestamp := time.Now().UTC().Add(-spec.Age)
	return team.App.InsertPhysicalFullBackup(
		t,
		apptest.PhysicalFullBackupSeed{
			DatabaseID:       team.Database.Id,
			StorageID:        team.StorageID,
			TimelineID:       1,
			Status:           status,
			ErrorReason:      spec.Reason,
			StartLSN:         new(fmt.Sprintf("0/%X", spec.Segment*16*1024*1024)),
			StopLSN:          new(fmt.Sprintf("0/%X", (spec.Segment+1)*16*1024*1024)),
			CreatedAt:        timestamp,
			CompletedAt:      &timestamp,
			FileName:         new(uuid.NewString() + ".zst"),
			ManifestFileName: spec.ManifestFileName,
		},
	)
}

func (team *backupTeam) seedIncremental(
	t *testing.T,
	fullID uuid.UUID,
	reason string,
) apptest.PhysicalIncrementalBackupSeed {
	t.Helper()
	timestamp := time.Now().UTC().Add(-2 * time.Hour)
	return team.App.InsertPhysicalIncrementalBackup(
		t,
		apptest.PhysicalIncrementalBackupSeed{
			DatabaseID:       team.Database.Id,
			StorageID:        team.StorageID,
			RootFullBackupID: fullID,
			TimelineID:       1,
			Status:           "CHAIN_BROKEN",
			ErrorReason:      &reason,
			StartLSN:         new("0/1000000"),
			StopLSN:          new("0/2000000"),
			CreatedAt:        timestamp,
			CompletedAt:      &timestamp,
		},
	)
}

func (team *backupTeam) setSchedules(t *testing.T, isFullOnly bool) {
	t.Helper()
	if isFullOnly {
		team.App.ExecuteSQL(
			t,
			"UPDATE databases_postgresql_physical SET backup_type='FULL' WHERE database_id=?",
			team.Database.Id,
		)
	}

	config := apptest.GetBackupConfig(t, team.Owner, team.Database.Id)
	settings, err := config.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	settings.FullBackupSchedule = apiclient.Schedule{
		Type:      apiclient.ScheduleTypeDAILY,
		TimeOfDay: new(fmt.Sprintf("%02d:00", (time.Now().UTC().Hour()+12)%24)),
	}

	if !isFullOnly {
		settings.IncrementalBackupSchedule = &apiclient.Schedule{Type: apiclient.ScheduleTypeHOURLY}
	} else {
		settings.IncrementalBackupSchedule = nil
		settings.WalLagThresholdBytes = 0
		settings.Retention.Policy = apiclient.PhysicalRetentionPolicyFULLBACKUPS
		settings.Retention.ChainsCount = 0
		settings.Retention.FullBackups = &apiclient.FullBackupsRetention{
			Policy: apiclient.FullBackupsRetentionPolicyLASTN,
			Count:  7,
		}
	}

	var backupSettings apiclient.BackupSettings
	require.NoError(t, backupSettings.FromPhysicalBackupSettings(settings))

	response, err := team.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		team.Database.Id,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &team.StorageID,
			Encryption:       config.Encryption,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}

// storeUndatedIncrementalCronWithDailySlotTwoHoursAgo places what the API refuses. The full schedule's time
// of day becomes the time of the daily schedule the backend builds in place of the stored expression.
func (team *backupTeam) storeUndatedIncrementalCronWithDailySlotTwoHoursAgo(t *testing.T) {
	t.Helper()

	team.App.ExecuteSQL(t, `UPDATE backups_configs_physical
		SET full_interval_type = 'CRON', full_cron_expression = '0 0 29 2 *', full_time_of_day = ?,
			incremental_interval_type = 'CRON', incremental_cron_expression = '0 0 31 2 *'
		WHERE database_id = ?`, time.Now().UTC().Add(-2*time.Hour).Format("15:04"), team.Database.Id)
}

func (team *backupTeam) runAutomaticBackup(t *testing.T) backupRow {
	t.Helper()
	before := map[uuid.UUID]bool{}
	for _, backup := range team.listBackups(t) {
		before[backup.ID] = true
	}

	team.App.RunJobPass(t, "backups_physical_scheduler")

	var terminal backupRow
	testwait.Until(t, "the automatic scheduler backup reaches a terminal state", func(context.Context) error {
		for _, backup := range team.listBackups(t) {
			if !before[backup.ID] && backup.Status != "IN_PROGRESS" {
				terminal = backup
				return nil
			}
		}
		return fmt.Errorf("no new terminal backup")
	})

	return terminal
}

func (team *backupTeam) assertNoScheduledBackup(t *testing.T) {
	t.Helper()
	before := team.listBackups(t)
	for range 3 {
		team.App.RunJobPass(t, "backups_physical_scheduler")
	}
	assert.Equal(t, before, team.listBackups(t))
}
