package walstreaming_test

import (
	"log/slog"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
)

type publishedFork struct {
	ID               uuid.UUID
	ParentTimelineID *int
	ForkLSN          *string
	IsConflicting    bool
}

//nolint:tparallel // Each scenario changes the published row the previous scenario left behind.
func TestTimelineHistoryForkPublication(t *testing.T) {
	t.Parallel()

	_, standby := containers.StartPhysicalPrimaryWithStandby(t, "postgres:17")
	observer := historyObserver{
		Handler:          slog.DiscardHandler,
		events:           make(chan string, 1024),
		conflictWarnings: make(chan []string, 16),
	}
	team := newWalTeam(t, walTeamSpec{SourceEndpoint: &standby, LogSink: observer})
	team.enable(t)
	team.waitForReceiver(t)
	source := team.openSource(t)
	var isPromoted bool
	require.NoError(t, source.QueryRow(t.Context(), "SELECT pg_promote()").Scan(&isPromoted))
	require.True(t, isPromoted)
	observer.Wait(t, "timeline switch observed via .history")

	var sourceHistory string
	require.NoError(
		t,
		source.QueryRow(t.Context(), "SELECT convert_from(pg_read_binary_file('pg_wal/00000002.history'), 'UTF8')").
			Scan(&sourceHistory),
	)
	sourceForkLSN := strings.Fields(strings.TrimSpace(sourceHistory))[1]
	published := team.getPublishedFork(t)

	t.Run("records the parent timeline and fork point when history is published", func(t *testing.T) {
		require.NotNil(t, published.ParentTimelineID)
		require.Equal(t, 1, *published.ParentTimelineID)
		require.NotNil(t, published.ForkLSN)
		require.Equal(t, sourceForkLSN, *published.ForkLSN)
		require.False(t, published.IsConflicting)
	})

	t.Run("reuses the published history when the same history arrives again", func(t *testing.T) {
		team.reportHistoryAgain(t, observer)

		require.Equal(t, published, team.getPublishedFork(t))
	})

	t.Run("marks the history conflicting when the fork point differs", func(t *testing.T) {
		team.App.ExecuteSQL(t, "UPDATE backups_physical_wal_history_files SET fork_lsn='0/1' WHERE id=?", published.ID)

		team.reportHistoryAgain(t, observer)

		warningKeys := observer.WaitForConflictWarning(t)
		slices.Sort(warningKeys)
		require.Equal(t, []string{"database_id", "timeline_id"}, slices.Compact(warningKeys))
		conflicting := team.getPublishedFork(t)
		require.Equal(t, published.ID, conflicting.ID)
		require.Equal(t, "0/1", *conflicting.ForkLSN)
		require.True(t, conflicting.IsConflicting)
	})

	t.Run("keeps the conflict mark when a matching history arrives later", func(t *testing.T) {
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_history_files SET fork_lsn=?::pg_lsn WHERE id=?",
			sourceForkLSN,
			published.ID,
		)

		team.reportHistoryAgain(t, observer)

		require.True(t, team.getPublishedFork(t).IsConflicting)
	})

	t.Run("reuses a history published without a fork point unchanged", func(t *testing.T) {
		team.App.ExecuteSQL(
			t,
			"UPDATE backups_physical_wal_history_files "+
				"SET parent_timeline_id=NULL, fork_lsn=NULL, is_conflicting=false WHERE id=?",
			published.ID,
		)

		team.reportHistoryAgain(t, observer)

		legacy := team.getPublishedFork(t)
		require.Equal(t, published.ID, legacy.ID)
		require.Nil(t, legacy.ParentTimelineID)
		require.Nil(t, legacy.ForkLSN)
		require.False(t, legacy.IsConflicting)
	})
}

func (team *walTeam) getPublishedFork(t *testing.T) publishedFork {
	t.Helper()

	var forks []publishedFork
	team.App.QuerySQL(
		t,
		&forks,
		"SELECT id, parent_timeline_id, fork_lsn::text AS fork_lsn, is_conflicting "+
			"FROM backups_physical_wal_history_files WHERE database_id=? AND timeline_id=2",
		team.Database.Id,
	)
	require.Len(t, forks, 1)

	return forks[0]
}

// reportHistoryAgain puts the history notification back into the receiver's queue, as the receiver
// does when the source reports the timeline's history again.
func (team *walTeam) reportHistoryAgain(t *testing.T, observer historyObserver) {
	t.Helper()

	historyPath := team.App.GetWalQueueFilePath(team.Database.Id, "00000002.history")
	require.NoError(t, os.WriteFile(historyPath, []byte("notification\n"), 0o600))
	observer.Wait(t, "timeline switch observed via .history")
}
