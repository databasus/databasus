package backuping_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/app/apptest/enginetest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalFullHistoryFailure(t *testing.T) {
	t.Parallel()

	t.Run(
		"keeps a valid full completed when source login fails during separate history acquisition",
		func(t *testing.T) {
			t.Parallel()

			_, firstStandby, standby := containers.StartPhysicalCascadingStandbys(t, "postgres:17")
			source, err := pgx.Connect(
				t.Context(),
				fmt.Sprintf(
					"host=%s port=%d user=testuser password=testpassword dbname=postgres sslmode=disable",
					firstStandby.Host,
					firstStandby.Port,
				),
			)
			require.NoError(t, err)
			firstSource := source
			t.Cleanup(func() { _ = firstSource.Close(context.WithoutCancel(t.Context())) })

			var isPromoted bool
			require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
			require.True(t, isPromoted)
			team := newBackupTeam(t, backupTeamSpec{SourceEndpoint: &firstStandby})

			engineTeam := enginetest.Team{App: team.App, Owner: team.Owner}
			user := engineTeam.CreateLimitedUser(t, team.Database)
			settings, err := team.Database.Settings.AsPostgresqlPhysicalSettings()
			require.NoError(t, err)
			settings.Username = user.Username
			settings.Password = &user.Password
			team.Database = engineTeam.UpdateDatabaseSettings(t, team.Database, settings)
			require.Equal(t, "COMPLETED", team.runBackup(t, "FULL").Status)

			cascading, err := pgx.Connect(
				t.Context(),
				fmt.Sprintf(
					"host=%s port=%d user=testuser password=testpassword dbname=postgres sslmode=disable",
					standby.Host,
					standby.Port,
				),
			)
			require.NoError(t, err)
			t.Cleanup(func() { _ = cascading.Close(context.WithoutCancel(t.Context())) })

			var roleLSN string
			require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_current_wal_insert_lsn()::text").Scan(&roleLSN))
			testwait.Until(t, "the next standby replays the backup role", func(ctx context.Context) error {
				var hasReplayed bool
				if err := cascading.QueryRow(ctx, "SELECT pg_last_wal_replay_lsn()>=$1::pg_lsn", roleLSN).
					Scan(&hasReplayed); err != nil {
					return err
				}

				if !hasReplayed {
					return fmt.Errorf("role WAL has not replayed")
				}

				return nil
			})

			require.NoError(t, cascading.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
			require.True(t, isPromoted)

			source = cascading
			team.Source = standby
			settings.Host = standby.Host
			settings.Port = standby.Port
			team.Database = engineTeam.UpdateDatabaseSettings(t, team.Database, settings)

			barrier := holdBackupSidecar(t, team, ".manifest")

			team.trigger(t, "FULL")
			team.App.RunJobPass(t, "backups_physical_scheduler")
			barrier.Wait(t)

			_, err = source.Exec(t.Context(), "ALTER ROLE "+pgx.Identifier{user.Username}.Sanitize()+" NOLOGIN")
			require.NoError(t, err)
			barrier.Release(t)
			testwait.Until(
				t,
				"the valid full completes despite unavailable history credentials",
				func(context.Context) error {
					backups := team.listBackups(t)
					if len(backups) != 2 || backups[0].Status == "IN_PROGRESS" {
						return fmt.Errorf("full has not completed")
					}

					require.Equal(t, "COMPLETED", backups[0].Status)

					return nil
				},
			)
			team.App.WaitForBackupRuns(t)

			var historyCount int
			team.App.QuerySQL(
				t,
				&historyCount,
				"SELECT count(*) FROM backups_physical_wal_history_files WHERE database_id=?",
				team.Database.Id,
			)
			assert.Equal(t, 1, historyCount)

			backup := team.listBackups(t)[0]
			status, body := team.App.CallPhysicalAPI(t, team.Owner, apptest.PhysicalBackupTokenRequest(backup.ID))
			require.Equal(t, http.StatusCreated, status, "%s", body)

			team.assertNoRunningBackup(t)
		},
	)
}
