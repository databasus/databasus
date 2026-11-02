package walstreaming_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
)

func TestPhysicalWalSourceInRecovery(t *testing.T) {
	t.Parallel()

	for _, version := range []string{"17", "18"} {
		t.Run(
			"reports unavailable source positions when PostgreSQL "+version+" is replaced by its standby",
			func(t *testing.T) {
				t.Parallel()

				primary, standby := containers.StartPhysicalPrimaryWithStandby(t, "postgres:"+version)
				team := newWalTeam(t, walTeamSpec{SourceEndpoint: &primary})
				team.App.ExecuteSQL(
					t,
					"UPDATE databases_postgresql_physical SET host=?,port=? WHERE database_id=?",
					standby.Host,
					standby.Port,
					team.Database.Id,
				)

				team.enable(t)
				testwait.Until(
					t,
					"the standby is reported unreachable instead of producing a primary WAL position",
					func(context.Context) error {
						reading := team.live(t)
						if !reading.IsStreamerRunning || reading.Current == nil || reading.Current.SourceState == nil ||
							*reading.Current.SourceState != "UNREACHABLE" {
							return fmt.Errorf("no standby probe yet")
						}
						assert.Nil(t, reading.Current.SourceCurrentLSN)
						assert.Nil(t, reading.Current.SlotLagBytes)
						assert.Nil(t, reading.Current.SlotWalStatus)
						return nil
					},
				)

				statistics := team.statistics(t)
				require.NotEmpty(t, statistics.Streamer.State)
			},
		)
	}
}
