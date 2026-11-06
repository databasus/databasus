package physicaltest

import (
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func RunRLSRestore(t *testing.T, image string, isIncremental bool) {
	t.Helper()

	team := setupReplicationOnlyTeam(
		t,
		physicalSourceSpec{Image: image, BackupType: apiclient.PhysicalBackupTypeFULLINCREMENTAL},
	)
	target := prepareRestoreTarget(t, image)
	sourceConn := openConnAt(t, team.Settings.Host, team.Settings.Port)

	createMarkerTable(t, sourceConn)
	insertMarker(t, sourceConn, markerRow{Phase: "tenant-one-before-full", Payload: "first tenant before full"})
	insertMarker(t, sourceConn, markerRow{Phase: "tenant-two-before-full", Payload: "second tenant before full"})

	if !isIncremental {
		enableMarkerRLS(t, sourceConn)
	}

	requireReplicationWithoutRowAccess(t, team)
	requirePhysicalRLSConnection(t, team)

	team.enableBackups(t, false)
	chain := team.waitForChain(t, 0)

	if isIncremental {
		enableMarkerRLS(t, sourceConn)
		requireReplicationWithoutRowAccess(t, team)
		requirePhysicalRLSConnection(t, team)
		insertMarker(t, sourceConn, markerRow{Phase: "tenant-one-after-full", Payload: "first tenant after full"})
		insertMarker(t, sourceConn, markerRow{Phase: "tenant-two-after-full", Payload: "second tenant after full"})
		team.buildIncremental(t, sourceConn, incrementalSpec{ParentStopLSN: chainTipStopLSN(t, chain), Count: 1})
	}

	sourceMarkers := getRLSMarkerRows(t, sourceConn)
	sourceRLS := getMarkerRLSConfiguration(t, sourceConn)

	bundle := team.downloadRestoreBundle(t, nil)
	reconstructCluster(t, target, restoreSpec{App: team.App, Image: image, BundlePath: bundle, TargetTime: nil})
	startRestoredCluster(t, target, image)

	restoredConn := openConnAt(t, target.Host(), target.MappedPort())
	assert.Equal(t, sourceMarkers, getRLSMarkerRows(t, restoredConn))
	assert.Equal(t, sourceRLS, getMarkerRLSConfiguration(t, restoredConn))
}

func enableMarkerRLS(t *testing.T, conn *pgx.Conn) {
	t.Helper()

	_, err := conn.Exec(t.Context(), `ALTER TABLE restore_marker ENABLE ROW LEVEL SECURITY;
		ALTER TABLE restore_marker FORCE ROW LEVEL SECURITY;
		CREATE POLICY tenant_access ON restore_marker FOR SELECT USING (phase LIKE 'tenant-one-%')`)
	require.NoError(t, err)
}

func requireReplicationWithoutRowAccess(t *testing.T, team Team) {
	t.Helper()

	conn := openConnAt(t, team.Settings.Host, team.Settings.Port)
	var hasSelect, isSuperuser, canBypassRLS, canReplicate bool
	err := conn.QueryRow(t.Context(), `SELECT has_table_privilege(rolname, 'restore_marker', 'SELECT'),
		rolsuper, rolbypassrls, rolreplication FROM pg_roles WHERE rolname = $1`, team.Settings.Username).
		Scan(&hasSelect, &isSuperuser, &canBypassRLS, &canReplicate)
	require.NoError(t, err)

	require.False(t, hasSelect)
	require.False(t, isSuperuser)
	require.False(t, canBypassRLS)
	require.True(t, canReplicate)
}

func requirePhysicalRLSConnection(t *testing.T, team Team) {
	t.Helper()

	response, err := team.Owner.Client.TestDatabaseWithResponse(t.Context(), team.Database.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	require.True(t, response.JSON200.IsConnected, "%s", response.Body)
}

func getRLSMarkerRows(t *testing.T, conn *pgx.Conn) []markerRow {
	t.Helper()

	rows, err := conn.Query(t.Context(), `SELECT phase, payload FROM restore_marker ORDER BY phase`)
	require.NoError(t, err)
	markers, err := pgx.CollectRows(rows, pgx.RowToStructByName[markerRow])
	require.NoError(t, err)

	return markers
}

func getMarkerRLSConfiguration(t *testing.T, conn *pgx.Conn) string {
	t.Helper()

	var configuration string
	err := conn.QueryRow(t.Context(), `SELECT json_build_object(
		'enabled', relrowsecurity, 'forced', relforcerowsecurity,
		'policies', (SELECT json_agg(policy ORDER BY policyname) FROM pg_policies policy
		WHERE schemaname = 'public' AND tablename = 'restore_marker'))::text
		FROM pg_class WHERE oid = 'public.restore_marker'::regclass`).Scan(&configuration)
	require.NoError(t, err)

	return configuration
}
