package physicaltest

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func (team Team) requirePublishedTimelineHistory(t *testing.T, source *pgx.Conn) {
	t.Helper()

	var timelineID int
	require.NoError(
		t,
		source.QueryRow(t.Context(), "SELECT ('x'||substr(pg_walfile_name(pg_current_wal_lsn()),1,8))::bit(32)::int").
			Scan(&timelineID),
	)
	historyFilename := fmt.Sprintf("%08X.history", timelineID)

	var histories []struct {
		ID        uuid.UUID
		FileName  string
		StorageID uuid.UUID
	}
	team.App.QuerySQL(
		t,
		&histories,
		"SELECT id,file_name,storage_id FROM backups_physical_wal_history_files WHERE database_id=? AND timeline_id=?",
		team.Database.Id,
		timelineID,
	)
	require.Len(t, histories, 1)
	configuration, err := team.Owner.Client.GetBackupConfigWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.NotNil(t, configuration.JSON200)
	require.NotNil(t, configuration.JSON200.StorageId)
	require.Equal(t, *configuration.JSON200.StorageId, histories[0].StorageID)

	var sourceHistory []byte
	require.NoError(
		t,
		// The SQL is constant and the history file path is a bound parameter.
		// nosemgrep: go.lang.security.audit.sqli.pgx-sqli.pgx-sqli
		source.QueryRow(t.Context(), "SELECT pg_read_binary_file($1)", "pg_wal/"+historyFilename).Scan(&sourceHistory),
	)

	compressed, err := os.ReadFile(team.App.GetLocalStorageFilePath(histories[0].FileName))
	require.NoError(t, err)

	decoder, err := zstd.NewReader(nil)
	require.NoError(t, err)
	defer decoder.Close()

	body, err := decoder.DecodeAll(compressed, nil)
	require.NoError(t, err)
	require.Equal(t, sourceHistory, body)

	metadataBytes, err := os.ReadFile(team.App.GetLocalStorageFilePath(histories[0].FileName + ".metadata"))
	require.NoError(t, err)
	var metadata struct {
		WalHistoryFileID uuid.UUID `json:"walHistoryFileId"`
		DatabaseID       uuid.UUID `json:"databaseId"`
		TimelineID       int       `json:"timelineId"`
		HistoryFilename  string    `json:"historyFilename"`
	}
	require.NoError(t, json.Unmarshal(metadataBytes, &metadata))

	require.Equal(t, histories[0].ID, metadata.WalHistoryFileID)
	require.Equal(t, team.Database.Id, metadata.DatabaseID)
	require.Equal(t, timelineID, metadata.TimelineID)
	require.Equal(t, historyFilename, metadata.HistoryFilename)
}
