package walstreaming_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeletePhysicalReplicationSlots(t *testing.T) {
	t.Parallel()

	for _, isUnreachable := range []bool{false, true} {
		scenario := "drops both orphan slots through the SSH bastion after database deletion commits"
		if isUnreachable {
			scenario = "deletes the database without waiting for an unreachable source to return"
		}
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()

			team := newWalTeam(t, walTeamSpec{IsWithoutLoops: true, IsBastioned: !isUnreachable})
			source := team.openSource(t)

			var names []struct {
				BackupSlotName      string
				ReplicationSlotName string
			}
			team.App.QuerySQL(
				t,
				&names,
				"SELECT backup_slot_name,replication_slot_name FROM databases_postgresql_physical WHERE database_id=?",
				team.Database.Id,
			)
			require.Len(t, names, 1)

			slots := []string{names[0].BackupSlotName, names[0].ReplicationSlotName}
			for _, slot := range slots {
				_, err := source.Exec(t.Context(), "SELECT pg_create_physical_replication_slot($1,true)", slot)
				require.NoError(t, err)
			}

			if isUnreachable {
				team.App.ExecuteSQL(
					t,
					"UPDATE databases_postgresql_physical SET host='127.0.0.1',port=1 WHERE database_id=?",
					team.Database.Id,
				)
			}

			response, err := team.Owner.Client.DeleteDatabaseWithResponse(t.Context(), team.Database.Id)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, response.StatusCode(), "%s", response.Body)

			var databases int
			team.App.QuerySQL(t, &databases, "SELECT count(*) FROM databases WHERE id=?", team.Database.Id)
			assert.Zero(t, databases)
			if !isUnreachable {
				var remaining int
				require.NoError(
					t,
					source.QueryRow(t.Context(), "SELECT count(*) FROM pg_replication_slots WHERE slot_name=ANY($1)", slots).
						Scan(&remaining),
				)
				assert.Zero(t, remaining)
			}
		})
	}
}
