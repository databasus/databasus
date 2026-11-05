package catalog_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestStartPhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run("sets the full request time when a full backup is requested", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		apptest.SaveBackupConfigWithStorage(
			t,
			catalog.Owner,
			apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID},
		)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, "FULL"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusAccepted,
			},
		)

		var requested []struct{ ForceFullRequestedAt *time.Time }
		catalog.App.QuerySQL(
			t,
			&requested,
			"SELECT force_full_requested_at FROM backups_configs_physical WHERE database_id = ?",
			catalog.DatabaseID,
		)
		assert.NotNil(t, requested[0].ForceFullRequestedAt)
	})

	t.Run("returns 409 when an incremental has no extendable chain", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		apptest.SaveBackupConfigWithStorage(t, catalog.Owner,
			apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID})

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, "INCREMENTAL"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusConflict,
				Destination:   &problem,
			},
		)

		assert.Equal(t, apiclient.ProblemCodeNoExtendableChain, problem.Code)
		assert.Equal(t, backupRequests{}, catalog.getBackupRequests(t))
	})

	t.Run("accepts an automatic request when no backup is running", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, "AUTO"),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusAccepted,
			},
		)
	})

	t.Run("returns invalid_request when backups are turned off", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.turnOffBackups(t)

		catalog.assertStartRefused(t, startRefusal{
			Kind: "FULL", Token: catalog.Owner.Token,
			StatusCode: http.StatusBadRequest, Code: apiclient.ProblemCodeInvalidRequest,
		})
	})

	t.Run("returns invalid_request when backups are turned off and the scheduler would choose", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.turnOffBackups(t)

		catalog.assertStartRefused(t, startRefusal{
			Kind: "AUTO", Token: catalog.Owner.Token,
			StatusCode: http.StatusBadRequest, Code: apiclient.ProblemCodeInvalidRequest,
		})
	})

	t.Run("returns invalid_request when backups are turned off and an incremental has no chain", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.turnOffBackups(t)

		catalog.assertStartRefused(t, startRefusal{
			Kind: "INCREMENTAL", Token: catalog.Owner.Token,
			StatusCode: http.StatusBadRequest, Code: apiclient.ProblemCodeInvalidRequest,
		})
	})

	t.Run("requests no incremental when backups are turned off and a chain exists", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.seedCascadeFull(t, cascadeFullSpec{})
		catalog.seedCascadeWal(t, cascadeWalSpec{Segment: 1})
		catalog.turnOffBackups(t)

		catalog.assertStartRefused(t, startRefusal{
			Kind: "INCREMENTAL", Token: catalog.Owner.Token,
			StatusCode: http.StatusBadRequest, Code: apiclient.ProblemCodeInvalidRequest,
		})

		catalog.saveStorageConfig(t, catalog.StorageID)
		catalog.App.RunJobPass(t, "backups_physical_scheduler")
		catalog.App.WaitForPhysicalBackups(t)

		assert.Empty(t, getBackups(t, catalog, "?type=INCREMENTAL").Backups)
	})

	t.Run("returns invalid_request when the backup config was never saved", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		unsavedDatabase := createSecondPhysicalDatabase(t, catalog)

		var problem apiclient.Problem
		catalog.call(t, catalogCall{
			Request:       apptest.PhysicalStartRequest(unsavedDatabase.ID, "FULL"),
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    http.StatusBadRequest,
			Destination:   &problem,
		})

		assert.Equal(t, apiclient.ProblemCodeInvalidRequest, problem.Code)
	})

	for _, kind := range []string{"FULL", "INCREMENTAL", "AUTO"} {
		t.Run(
			"returns invalid_request when backups are on and the config names no storage with kind "+kind,
			func(t *testing.T) {
				t.Parallel()

				catalog := newCatalog(t)
				cleared := catalog.saveStorageSelection(t, nil)
				require.Equal(t, http.StatusOK, cleared.StatusCode(), "%s", cleared.Body)
				require.True(t, apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID).IsBackupsEnabled)

				catalog.assertStartRefused(t, startRefusal{
					Kind: kind, Token: catalog.Owner.Token,
					StatusCode: http.StatusBadRequest, Code: apiclient.ProblemCodeInvalidRequest,
				})
			},
		)
	}

	t.Run("returns permission_denied to a viewer when backups are turned off", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		viewer := catalog.addViewer(t)
		catalog.turnOffBackups(t)

		catalog.assertStartRefused(t, startRefusal{
			Kind: "FULL", Token: viewer.Token,
			StatusCode: http.StatusForbidden, Code: apiclient.ProblemCodePermissionDenied,
		})
	})

	t.Run("returns not_found to an outsider when backups are turned off", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		outsider := catalog.App.CreateNonAdminUser(t, catalog.Admin)
		catalog.turnOffBackups(t)

		catalog.assertStartRefused(t, startRefusal{
			Kind: "FULL", Token: outsider.Token,
			StatusCode: http.StatusNotFound, Code: apiclient.ProblemCodeNotFound,
		})
	})
}

type startRefusal struct {
	Kind       string
	Token      string
	StatusCode int
	Code       apiclient.ProblemCode
}

func (catalog *catalogFixture) assertStartRefused(t *testing.T, refusal startRefusal) {
	t.Helper()

	var problem apiclient.Problem
	catalog.call(t, catalogCall{
		Request:       apptest.PhysicalStartRequest(catalog.DatabaseID, refusal.Kind),
		Authorization: "Bearer " + refusal.Token,
		StatusCode:    refusal.StatusCode,
		Destination:   &problem,
	})
	assert.Equal(t, refusal.Code, problem.Code)

	assert.Equal(t, backupRequests{}, catalog.getBackupRequests(t))

	var statistics apiclient.WalStreamStatistics
	catalog.call(t, catalogCall{
		Request:       apptest.PhysicalStatisticsRequest(catalog.DatabaseID, "HOUR"),
		Authorization: "Bearer " + catalog.Owner.Token,
		StatusCode:    http.StatusOK,
		Destination:   &statistics,
	})
	assert.False(t, statistics.Health.IsFullBackupPending)
}

func (catalog *catalogFixture) turnOffBackups(t *testing.T) {
	t.Helper()

	config := apptest.GetBackupConfig(t, catalog.Owner, catalog.DatabaseID)

	response, err := catalog.Owner.Client.UpdateBackupConfigWithResponse(
		t.Context(),
		catalog.DatabaseID,
		apiclient.BackupConfigInput{
			IsBackupsEnabled: false,
			StorageId:        &catalog.StorageID,
			Encryption:       config.Encryption,
			Settings:         config.Settings,
		},
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode(), "%s", response.Body)
}
