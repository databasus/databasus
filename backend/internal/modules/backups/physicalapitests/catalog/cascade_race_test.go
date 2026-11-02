package catalog_test

import (
	"context"
	"errors"
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

func TestCascadePhysicalBackupDuringClaim(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name           string
		shouldKeepFull bool
	}{
		{name: "keeps WAL after a concurrent running full commits while deletion waits", shouldKeepFull: false},
		{name: "keeps WAL after a concurrent running full commits while retained full cleanup waits", shouldKeepFull: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			full := catalog.seedCascadeFull(t, cascadeFullSpec{Age: 80 * time.Hour})
			wal := catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1, Age: 70 * time.Hour})
			catalog.publishNames(t, *wal.FileName)
			if testCase.shouldKeepFull {
				catalog.seedCascadeFull(t, cascadeFullSpec{Segment: 10, Age: 60 * time.Hour})
				catalog.configureFullRetention(t)
			}

			transaction := catalog.App.BeginSQLTransaction(t)
			transaction.ExecuteSQL(
				t,
				"SELECT pg_advisory_xact_lock(hashtextextended('physical-backup:' || ?::text,0))",
				catalog.DatabaseID,
			)
			type cascadeOutcome struct {
				status int
				err    error
			}
			outcomes := make(chan cascadeOutcome, 1)
			go func() {
				if testCase.shouldKeepFull {
					catalog.App.RunJobPass(t, "backups_physical_cleaner")
					outcomes <- cascadeOutcome{}
					return
				}
				response, err := catalog.Owner.Client.DeletePhysicalBackup(t.Context(), full.ID)
				if err != nil {
					outcomes <- cascadeOutcome{err: err}
					return
				}
				_ = response.Body.Close()
				outcomes <- cascadeOutcome{status: response.StatusCode}
			}()

			testwait.Until(t, "the cascade waits on the held backup lock", func(context.Context) error {
				var waiting int64
				catalog.App.QuerySQL(
					t,
					&waiting,
					`SELECT count(*) FROM pg_locks waiting JOIN pg_locks held ON waiting.locktype=held.locktype AND waiting.database=held.database AND waiting.classid=held.classid AND waiting.objid=held.objid AND waiting.objsubid=held.objsubid WHERE NOT waiting.granted AND held.granted AND held.pid IN (SELECT pid FROM pg_stat_activity WHERE datname=current_database()) AND waiting.locktype='advisory'`,
				)
				if waiting == 0 {
					return errors.New("the cascade has not reached the lock")
				}
				return nil
			})

			select {
			case <-outcomes:
				t.Fatal("the cascade finished before the running full committed")
			default:
			}

			activeFullID := uuid.New()
			transaction.ExecuteSQL(
				t,
				"INSERT INTO backups_physical(id,database_id,storage_id,timeline_id,status,created_at,kind) VALUES(?,?,?,1,'IN_PROGRESS',?,'FULL')",
				activeFullID,
				catalog.DatabaseID,
				catalog.StorageID,
				time.Now().UTC(),
			)
			transaction.Commit(t)

			select {
			case outcome := <-outcomes:
				require.NoError(t, outcome.err)
				if !testCase.shouldKeepFull {
					require.Equal(t, http.StatusNoContent, outcome.status)
				}
			case <-t.Context().Done():
				t.Fatal("the cascade did not finish after the running full committed")
			}

			catalog.assertWalPresent(t, wal.ID)
			assert.True(t, catalog.App.IsLocalStorageFilePresent(t, *wal.FileName))
			if testCase.shouldKeepFull {
				catalog.assertFullPresent(t, full.ID)
			} else {
				catalog.assertFullGone(t, full.ID)
			}
		})
	}
}

func (catalog *catalogFixture) configureFullRetention(t *testing.T) {
	t.Helper()
	catalog.App.ExecuteSQL(
		t,
		"UPDATE databases_postgresql_physical SET backup_type='FULL' WHERE database_id=?",
		catalog.DatabaseID,
	)

	config := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)
	settings, err := config.Settings.AsPhysicalBackupSettings()
	require.NoError(t, err)
	configureRetentionSchedule(&settings)
	settings.IncrementalBackupSchedule = nil
	settings.WalLagThresholdBytes = 0
	settings.Retention.Policy = apiclient.PhysicalRetentionPolicyFULLBACKUPS
	settings.Retention.ChainsCount = 0
	settings.Retention.FullBackups = &apiclient.FullBackupsRetention{
		Policy: apiclient.FullBackupsRetentionPolicyLASTN,
		Count:  20,
	}

	var backupSettings apiclient.BackupSettings
	require.NoError(t, backupSettings.FromPhysicalBackupSettings(settings))
	response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		catalog.DatabaseID,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: true,
			StorageId:        &catalog.StorageID,
			Encryption:       apiclient.BackupEncryptionNONE,
			Settings:         backupSettings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}
