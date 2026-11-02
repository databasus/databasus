package backuping_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/testwait"
)

func TestIncrementalBackupWithPendingWalSummaries(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"17", "18"} {
		for _, isWriting := range []bool{false, true} {
			scenario := "completes an incremental on an idle PostgreSQL " + version + " source while summaries trail its full"
			if isWriting {
				scenario = "completes an incremental after 64 MiB of new WAL on PostgreSQL " + version + " without a manual checkpoint"
			}
			t.Run(scenario, func(t *testing.T) {
				t.Parallel()

				team := newBackupTeam(t, backupTeamSpec{Image: "postgres:" + version})
				full := team.runBackup(t, "full")
				require.Equal(t, "COMPLETED", full.Status)
				source := team.openSource(t)

				if isWriting {
					_, err := source.Exec(t.Context(), "CREATE TABLE pending_summary_writes(payload text)")
					require.NoError(t, err)
					var start string
					require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_current_wal_lsn()::text").Scan(&start))

					for {
						_, err = source.Exec(
							t.Context(),
							"INSERT INTO pending_summary_writes SELECT repeat('x',1024) FROM generate_series(1,1000)",
						)
						require.NoError(t, err)
						var bytes int64
						require.NoError(
							t,
							source.QueryRow(t.Context(), "SELECT pg_wal_lsn_diff(pg_current_wal_lsn(),$1::pg_lsn)::bigint", start).
								Scan(&bytes),
						)
						if bytes >= 64*1024*1024 {
							break
						}
					}
				}

				incremental := team.runBackup(t, "incremental")
				require.Equal(t, "COMPLETED", incremental.Status)
				team.assertArtifacts(t, incremental)
				team.assertNoRunningBackup(t)
			})
		}
	}
	t.Run("breaks the incremental chain when its parent stop position precedes the oldest summary", func(t *testing.T) {
		t.Parallel()

		team := newBackupTeam(t, backupTeamSpec{})
		full := team.runBackup(t, "full")
		require.Equal(t, "COMPLETED", full.Status)
		source := team.openSource(t)

		testwait.Until(t, "the source publishes its first WAL summary", func(ctx context.Context) error {
			var count int
			err := source.QueryRow(ctx, "SELECT count(*) FROM pg_available_wal_summaries()").Scan(&count)
			if err != nil {
				return err
			}
			if count == 0 {
				return fmt.Errorf("no summaries yet")
			}
			return nil
		})

		team.App.ExecuteSQL(t, "UPDATE backups_physical SET stop_lsn='0/1' WHERE id=?", full.ID)
		incremental := team.runBackup(t, "incremental")
		assert.Equal(t, "CHAIN_BROKEN", incremental.Status)
		team.assertTerminalDetails(t, incremental, "SUMMARIES_EXPIRED")
		team.assertNoRunningBackup(t)
	})
}
