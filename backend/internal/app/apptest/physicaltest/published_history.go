package physicaltest

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

func (team Team) requirePublishedTimelineHistory(t *testing.T, source *pgx.Conn) {
	t.Helper()

	var histories []struct {
		ID        uuid.UUID
		FileName  string
		StorageID uuid.UUID
	}
	team.App.QuerySQL(
		t,
		&histories,
		"SELECT id,file_name,storage_id FROM backups_physical_wal_history_files WHERE database_id=? AND timeline_id=2",
		team.Database.Id,
	)
	require.Len(t, histories, 1)
	require.Equal(t, team.StorageID, histories[0].StorageID)

	var sourceHistory []byte
	require.NoError(
		t,
		source.QueryRow(t.Context(), "SELECT pg_read_binary_file('pg_wal/00000002.history')").Scan(&sourceHistory),
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
	require.Equal(t, 2, metadata.TimelineID)
	require.Equal(t, "00000002.history", metadata.HistoryFilename)
}
