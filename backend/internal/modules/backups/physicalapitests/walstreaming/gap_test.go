package walstreaming_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/lib/testwait"
)

func TestIncrementalPhysicalBackupAfterWalGap(t *testing.T) {
	t.Parallel()

	t.Run(
		"completes an incremental after a missing streamed segment while reporting the catalog gap",
		func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{})
			team.enable(t)
			team.waitForReceiver(t)

			var full backupRow
			testwait.Until(t, "the initial full backup completes", func(context.Context) error {
				for _, backup := range team.listBackups(t) {
					if backup.Type == "FULL" && backup.Status == "COMPLETED" {
						full = backup
						return nil
					}
				}
				return fmt.Errorf("no completed full")
			})

			source := team.openSource(t)
			for range 4 {
				rotateWal(t, source)
			}

			var segmentIDs []string
			testwait.Until(t, "at least three committed WAL segments follow the full", func(context.Context) error {
				team.App.QuerySQL(
					t,
					&segmentIDs,
					"SELECT id FROM backups_physical_wal_segments WHERE database_id=? AND file_name IS NOT NULL AND end_lsn>?::pg_lsn ORDER BY start_lsn",
					team.Database.Id,
					full.StopLSN,
				)
				if len(segmentIDs) < 3 {
					return fmt.Errorf("too few post-full segments")
				}
				return nil
			})

			team.App.ExecuteSQL(
				t,
				"DELETE FROM backups_physical_wal_segments WHERE id=?",
				segmentIDs[len(segmentIDs)/2],
			)

			var page struct {
				Backups []struct {
					HasGapBefore bool `json:"hasGapBefore"`
				} `json:"items"`
			}
			team.call(
				t,
				apptest.PhysicalListRequest(team.Database.Id, "?type=WAL&limit=1000"),
				http.StatusOK,
				&page,
			)

			hasGap := false
			for _, backup := range page.Backups {
				hasGap = hasGap || backup.HasGapBefore
			}
			require.True(t, hasGap)

			previousBackups := team.listBackups(t)
			team.call(
				t,
				apptest.PhysicalStartRequest(team.Database.Id, "INCREMENTAL"),
				http.StatusAccepted,
				nil,
			)
			team.App.RunJobPass(t, "backups_physical_scheduler")
			testwait.Until(t, "the requested incremental backup completes", func(context.Context) error {
				for _, backup := range team.listBackups(t) {
					if backup.Type != "INCREMENTAL" ||
						slices.ContainsFunc(previousBackups, func(previous backupRow) bool {
							return previous.ID == backup.ID
						}) {
						continue
					}

					if backup.Status == "IN_PROGRESS" {
						return fmt.Errorf("incremental backup %s is still running", backup.ID)
					}

					require.Equal(t, "COMPLETED", backup.Status)
					return nil
				}

				return fmt.Errorf("the requested incremental backup has not started")
			})
		},
	)
}
