package physicaltest

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type patroniWalPublication struct {
	ID       uuid.UUID
	FileName string
}

func (source patroniSource) delayWalPublication(t *testing.T, timelineID int) patroniWalPublication {
	t.Helper()

	var publication patroniWalPublication
	source.Team.App.QuerySQL(
		t,
		&publication,
		"SELECT id, file_name FROM backups_physical_wal_segments WHERE database_id=? AND timeline_id=? AND file_name IS NOT NULL ORDER BY end_lsn DESC LIMIT 1",
		source.Team.Database.Id,
		timelineID,
	)

	require.NotEqual(t, uuid.Nil, publication.ID)
	require.NotEmpty(t, publication.FileName)

	source.Team.App.ExecuteSQL(t, "UPDATE backups_physical_wal_segments SET file_name=NULL WHERE id=?", publication.ID)

	return publication
}

func (source patroniSource) publishDelayedWal(t *testing.T, publication patroniWalPublication, receivedAt time.Time) {
	t.Helper()

	source.Team.App.ExecuteSQL(t,
		"UPDATE backups_physical_wal_segments SET file_name=?, received_at=? WHERE id=?",
		publication.FileName, receivedAt, publication.ID)
}
