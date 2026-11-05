package catalog_test

import (
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func (catalog *catalogFixture) addViewer(t *testing.T) apptest.Session {
	t.Helper()
	viewer := catalog.App.CreateNonAdminUser(t, catalog.Admin)
	apptest.AddWorkspaceMember(
		t,
		catalog.Owner,
		catalog.WorkspaceID,
		viewer.Email,
		apiclient.AssignableWorkspaceRoleWORKSPACEVIEWER,
	)
	return viewer
}

func (catalog *catalogFixture) getRunningBackupID(t *testing.T) *uuid.UUID {
	t.Helper()
	var backupIDs []uuid.UUID
	catalog.App.QuerySQL(
		t,
		&backupIDs,
		"SELECT id FROM backups_physical WHERE status = 'IN_PROGRESS' AND database_id = ?",
		catalog.DatabaseID,
	)
	if len(backupIDs) == 0 {
		return nil
	}
	require.Len(t, backupIDs, 1)
	return &backupIDs[0]
}

func (catalog *catalogFixture) assertFullGone(t *testing.T, backupID uuid.UUID) {
	t.Helper()
	assert.NotContains(t, collectBackupIDs(getBackups(t, catalog, "?type=FULL&limit=1000")), backupID)
}

func (catalog *catalogFixture) assertFullPresent(t *testing.T, backupID uuid.UUID) {
	t.Helper()
	assert.Contains(t, collectBackupIDs(getBackups(t, catalog, "?type=FULL&limit=1000")), backupID)
}

func (catalog *catalogFixture) assertIncrementalGone(t *testing.T, backupID uuid.UUID) {
	t.Helper()
	assert.NotContains(t, collectBackupIDs(getBackups(t, catalog, "?type=INCREMENTAL&limit=1000")), backupID)
}

func (catalog *catalogFixture) assertIncrementalPresent(t *testing.T, backupID uuid.UUID) {
	t.Helper()
	assert.Contains(t, collectBackupIDs(getBackups(t, catalog, "?type=INCREMENTAL&limit=1000")), backupID)
}

func (catalog *catalogFixture) assertWalGone(t *testing.T, segmentID uuid.UUID) {
	t.Helper()
	var count int64
	catalog.App.QuerySQL(t, &count, "SELECT count(*) FROM backups_physical_wal_segments WHERE id = ?", segmentID)
	assert.Zero(t, count)
}

func (catalog *catalogFixture) assertWalPresent(t *testing.T, segmentID uuid.UUID) {
	t.Helper()
	var count int64
	catalog.App.QuerySQL(t, &count, "SELECT count(*) FROM backups_physical_wal_segments WHERE id = ?", segmentID)
	assert.Equal(t, int64(1), count)
}

func (catalog *catalogFixture) assertAuditLogContains(t *testing.T, messageFragment, databaseName string) {
	t.Helper()
	response, err := catalog.Owner.Client.ListAuditEventsWithResponse(
		t.Context(),
		&apiclient.ListAuditEventsParams{WorkspaceId: &catalog.WorkspaceID},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
	assert.True(t, slices.ContainsFunc(response.JSON200.Items, func(event apiclient.AuditEvent) bool {
		return strings.Contains(event.Message, messageFragment) && strings.Contains(event.Message, databaseName)
	}), "audit events must include %q and %q", messageFragment, databaseName)
}
