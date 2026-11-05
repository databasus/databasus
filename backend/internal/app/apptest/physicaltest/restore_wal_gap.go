package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func RunWalGap(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM},
	)
	conn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	team.enableBackups(t, true)
	chain := team.waitForChain(t, 0)
	team.streamPostFullSegments(t, conn, walArchiveSpec{Chain: chain, Count: 3})

	var segmentIDs []uuid.UUID
	team.App.QuerySQL(
		t,
		&segmentIDs,
		`SELECT id FROM backups_physical_wal_segments WHERE database_id = ? AND timeline_id = ? AND file_name IS NOT NULL AND end_lsn > ? ORDER BY start_lsn`,
		team.Database.Id,
		chain[len(chain)-1].TimelineID,
		chain[len(chain)-1].StopLSN,
	)
	require.GreaterOrEqual(t, len(segmentIDs), 3)
	team.App.ExecuteSQL(t, `DELETE FROM backups_physical_wal_segments WHERE id = ?`, segmentIDs[len(segmentIDs)/2])

	testwait.Until(t, "the removed WAL segment appears as a gap", func(context.Context) error {
		for _, wal := range team.listBackups(t, "WAL") {
			if wal.HasGapBefore {
				return nil
			}
		}
		return fmt.Errorf("no WAL range reports a gap")
	})

	targetTime := sourceTime(t, conn)
	var response apiclient.Problem
	team.call(
		t,
		physicalResponseSpec{
			Call:        team.restoreTokenCall(&targetTime),
			Status:      http.StatusConflict,
			Destination: &response,
		},
	)
	assert.Equal(t, apiclient.ProblemCodeTargetTimeUnreachable, response.Code)
	assert.Contains(t, response.Detail, "WAL gap")
}
