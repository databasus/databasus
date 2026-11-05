package physicaltest

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
	"databasus-backend/internal/lib/testwait"
)

func getDatabaseSlots(t *testing.T, conn *pgx.Conn, isWalOnly bool) []string {
	t.Helper()

	rows, err := conn.Query(
		t.Context(),
		`SELECT slot_name FROM pg_replication_slots WHERE starts_with(slot_name, 'databasus_slot_') OR ($1 = false AND starts_with(slot_name, 'databasus_basebackup_'))`,
		isWalOnly,
	)
	require.NoError(t, err)
	defer rows.Close()
	slots, err := pgx.CollectRows(rows, pgx.RowTo[string])
	require.NoError(t, err)
	return slots
}

func waitForSlotPresent(t *testing.T, conn *pgx.Conn) string {
	t.Helper()

	var slots []string
	testwait.Until(t, "the WAL replication slot appears", func(context.Context) error {
		slots = getDatabaseSlots(t, conn, true)
		if len(slots) == 0 {
			return fmt.Errorf("no WAL replication slot exists")
		}
		return nil
	})
	require.Len(t, slots, 1)
	return slots[0]
}

func requireDatabaseSlotsGone(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	testwait.Until(t, "the deleted database leaves no replication slots", func(context.Context) error {
		slots := getDatabaseSlots(t, conn, false)
		if len(slots) > 0 {
			return fmt.Errorf("replication slots still pin WAL: %v", slots)
		}
		return nil
	})
}

func RunSlotLifecycle(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM},
	)
	conn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	require.Empty(t, getDatabaseSlots(t, conn, true))
	team.enableBackups(t, true)
	waitForSlotPresent(t, conn)

	team.deleteDatabase(t)
	requireDatabaseSlotsGone(t, conn)
}

func RunDeleteStreamingDatabase(t *testing.T, image string) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTALWALSTREAM},
	)
	conn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	team.enableBackups(t, true)
	team.waitForChain(t, 0)
	waitForSlotPresent(t, conn)
	generateWalActivity(t, conn, 64*1024*1024)

	team.deleteDatabase(t)
	requireDatabaseSlotsGone(t, conn)
}
