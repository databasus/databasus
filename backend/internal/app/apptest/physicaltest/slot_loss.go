package physicaltest

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

// requireMissingSlotResponse waits until Databasus either recovers its missing slot and archives new
// WAL through it, or reports BROKEN/SLOT_MISSING. Resuming new WAL does not prove that an older
// unarchived interval is recoverable; callers check their target independently.
func (source failoverSource) requireMissingSlotResponse(t *testing.T) (hasRecovered bool) {
	t.Helper()

	connection := source.connect(t)
	testwait.Until(t, "the missing slot is recovered or diagnosed as broken", func(ctx context.Context) error {
		response, err := source.Team.Owner.Client.GetWalStreamStatisticsWithResponse(
			ctx,
			source.Team.Database.Id,
			&apiclient.GetWalStreamStatisticsParams{Period: apiclient.WalStreamStatisticsPeriodHOUR},
		)
		if err != nil {
			return err
		}

		if response.JSON200 == nil {
			return fmt.Errorf("stream statistics: %d %s", response.StatusCode(), response.Body)
		}

		health := response.JSON200.Health
		for _, reason := range health.Reasons {
			if reason.Code == apiclient.WalStreamReasonCodeSLOTMISSING {
				require.Equal(t, apiclient.WalStreamHealthLevelBROKEN, health.Level)
				t.Logf("missing slot %s is diagnosed as BROKEN/SLOT_MISSING", source.SlotName)
				return nil
			}
		}

		var isActive bool
		if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_replication_slots WHERE slot_name=$1 AND active)", source.SlotName).
			Scan(&isActive); err != nil {
			return err
		}

		if isActive {
			hasRecovered = true
			t.Logf("Databasus recovered missing slot %s", source.SlotName)
			return nil
		}

		return fmt.Errorf("missing slot health: %+v", health)
	})

	if hasRecovered {
		resumedSegment := getCurrentSegment(t, connection)
		generateWalActivity(t, connection, 16*1024*1024)
		forceWalRotation(t, connection)
		source.Team.waitForArchivedSegment(t, resumedSegment)
	}

	return hasRecovered
}

// recycledSegment is a source WAL segment Databasus needed and never archived, and a target inside it.
type recycledSegment struct {
	Segment string
	Target  time.Time
}

// requireRecycledSegmentRefused waits until the connected primary recycles the segment and checks
// that no restore token reaches the target, before and after the slot and a FULL exist again.
func (source failoverSource) requireRecycledSegmentRefused(t *testing.T, connection *pgx.Conn, spec recycledSegment) {
	t.Helper()

	requiredSegment := spec.Segment
	target := spec.Target
	source.waitForRecycledSegment(t, connection, requiredSegment)

	require.False(t, source.Team.hasArchivedSegment(t, requiredSegment))
	source.requireUnreachableTarget(t, target)
	source.Proxy.SetReplicationBlocked(false)
	source.requireMissingSlotResponse(t)
	source.Team.App.RunJobPass(t, "backups_physical_wal_stream_manager")
	source.requireUnreachableTarget(t, target)
	backups := source.Team.listBackups(t, "FULL")
	require.NotEmpty(t, backups)
	source.Team.call(
		t,
		physicalResponseSpec{
			Call:   apptest.PhysicalStartRequest(source.Team.Database.Id, "FULL"),
			Status: http.StatusAccepted,
		},
	)
	source.waitForNewFull(t, backups[0].ID)
	source.requireUnreachableTarget(t, target)
	t.Logf("required segment %s was recycled without archival; target %s remains refused", requiredSegment, target)
}

func (source failoverSource) waitForRecycledSegment(t *testing.T, connection *pgx.Conn, requiredSegment string) {
	t.Helper()

	testwait.Until(t, "PostgreSQL recycles the identified required source WAL", func(ctx context.Context) error {
		generateWalActivity(t, connection, 32*1024*1024)
		forceWalRotation(t, connection)
		_, err := connection.Exec(ctx, "CHECKPOINT")
		if err != nil {
			return err
		}

		var hasSegment bool
		if err := connection.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_ls_waldir() WHERE name=$1)", requiredSegment).
			Scan(&hasSegment); err != nil {
			return err
		}

		if hasSegment {
			return fmt.Errorf("required segment %s is still retained", requiredSegment)
		}

		return nil
	})
}
