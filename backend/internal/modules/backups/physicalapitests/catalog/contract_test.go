package catalog_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestPhysicalBackupProblems(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name    string
		request func(*catalogFixture) apptest.PhysicalRequest
		status  int
		code    apiclient.ProblemCode
	}{
		{"returns invalid_request when the page limit is zero", func(catalog *catalogFixture) apptest.PhysicalRequest {
			return apptest.PhysicalListRequest(catalog.DatabaseID, "?limit=0")
		}, 400, "invalid_request"},
		{"returns invalid_request when the page limit exceeds 1000", func(catalog *catalogFixture) apptest.PhysicalRequest {
			return apptest.PhysicalListRequest(catalog.DatabaseID, "?limit=1001")
		}, 400, "invalid_request"},
		{"returns invalid_request when the cursor is malformed", func(catalog *catalogFixture) apptest.PhysicalRequest {
			return apptest.PhysicalListRequest(catalog.DatabaseID, "?after=malformed")
		}, 400, "invalid_request"},
		{"returns no_extendable_chain when an incremental has no full backup", func(catalog *catalogFixture) apptest.PhysicalRequest {
			return apptest.PhysicalStartRequest(catalog.DatabaseID, "INCREMENTAL")
		}, 409, "no_extendable_chain"},
		{"returns invalid_request when the requested kind is unknown", func(catalog *catalogFixture) apptest.PhysicalRequest {
			return apptest.PhysicalStartRequest(catalog.DatabaseID, "UNKNOWN")
		}, 400, "invalid_request"},
		{"returns not_found when a cancelled backup does not exist", func(*catalogFixture) apptest.PhysicalRequest { return apptest.PhysicalCancelRequest(uuid.New()) }, 404, "not_found"},
		{"returns not_found when a deleted backup does not exist", func(*catalogFixture) apptest.PhysicalRequest { return apptest.PhysicalDeleteRequest(uuid.New()) }, 404, "not_found"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			status, body := catalog.App.CallPhysicalAPI(t, catalog.Owner, scenario.request(catalog))
			require.Equal(t, scenario.status, status, "%s", body)
			var problem apiclient.Problem
			require.NoError(t, json.Unmarshal(body, &problem))
			assert.Equal(t, scenario.code, problem.Code)
		})
	}
}

func TestPhysicalBackupSession(t *testing.T) {
	t.Parallel()

	t.Run("returns unauthenticated for each protected operation when no session is supplied", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		for _, request := range []apptest.PhysicalRequest{
			apptest.PhysicalListRequest(catalog.DatabaseID, ""),
			apptest.PhysicalStartRequest(catalog.DatabaseID, "FULL"),
			apptest.PhysicalCancelRequest(uuid.New()),
			apptest.PhysicalDeleteRequest(uuid.New()),
			apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil),
			apptest.PhysicalBackupTokenRequest(uuid.New()),
			apptest.PhysicalStatisticsRequest(catalog.DatabaseID, "HOUR"),
			apptest.PhysicalLiveRequest(catalog.DatabaseID),
		} {
			status, body := catalog.App.CallPhysicalAPI(t, apptest.Session{}, request)
			require.Equal(t, http.StatusUnauthorized, status, "%s", body)
			var problem apiclient.Problem
			require.NoError(t, json.Unmarshal(body, &problem))
			assert.Equal(t, apiclient.ProblemCodeUnauthenticated, problem.Code)
		}
	})
}

func TestDeletePhysicalWalSegment(t *testing.T) {
	t.Parallel()

	t.Run("returns not_found and preserves the range when deletion names a WAL segment", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		_, _, segment := seedOneOfEachType(t, catalog)
		before := getBackups(t, catalog, "")

		status, body := catalog.App.CallPhysicalAPI(t, catalog.Owner, apptest.PhysicalDeleteRequest(segment.ID))
		require.Equal(t, http.StatusNotFound, status, "%s", body)
		var problem apiclient.Problem
		require.NoError(t, json.Unmarshal(body, &problem))
		assert.Equal(t, apiclient.ProblemCodeNotFound, problem.Code)

		after := getBackups(t, catalog, "")
		assert.Equal(t, before, after)
	})
}

func TestOldPhysicalBackupPath(t *testing.T) {
	t.Parallel()

	t.Run("returns not_found when a client requests the removed physical list path", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			catalog.App.URL+"/api/v1/backups/physical/database/"+catalog.DatabaseID.String()+"/backups",
			nil,
		)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer "+catalog.Owner.Token)

		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { _ = response.Body.Close() }()
		require.Equal(t, http.StatusNotFound, response.StatusCode)

		var problem apiclient.Problem
		require.NoError(t, json.NewDecoder(response.Body).Decode(&problem))
		assert.Equal(t, apiclient.ProblemCodeNotFound, problem.Code)
	})
}

func TestPhysicalBackupWorkspacePrivacy(t *testing.T) {
	t.Parallel()

	t.Run(
		"returns the missing-resource problem when an outsider names an existing database or backup",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			full, _, _ := seedOneOfEachType(t, catalog)
			apptest.SaveBackupConfigWithStorage(t, catalog.Owner,
				apptest.StorageUse{DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID})
			running := catalog.insertFull(t, newInProgressFull(inProgressFullSpec{
				DatabaseID: catalog.DatabaseID, StorageID: catalog.StorageID, TimelineID: 1,
			}))

			outsider := catalog.App.CreateNonAdminUser(t, catalog.Admin)
			for _, requests := range [][2]apptest.PhysicalRequest{
				{apptest.PhysicalListRequest(catalog.DatabaseID, ""), apptest.PhysicalListRequest(uuid.New(), "")},
				{apptest.PhysicalStartRequest(catalog.DatabaseID, "FULL"), apptest.PhysicalStartRequest(uuid.New(), "FULL")},
				{apptest.PhysicalCancelRequest(running.ID), apptest.PhysicalCancelRequest(uuid.New())},
				{apptest.PhysicalDeleteRequest(full.ID), apptest.PhysicalDeleteRequest(uuid.New())},
				{apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil), apptest.PhysicalPitrTokenRequest(uuid.New(), nil)},
				{apptest.PhysicalBackupTokenRequest(full.ID), apptest.PhysicalBackupTokenRequest(uuid.New())},
				{apptest.PhysicalStatisticsRequest(catalog.DatabaseID, "HOUR"), apptest.PhysicalStatisticsRequest(uuid.New(), "HOUR")},
				{apptest.PhysicalLiveRequest(catalog.DatabaseID), apptest.PhysicalLiveRequest(uuid.New())},
			} {
				status, body := catalog.App.CallPhysicalAPI(t, outsider, requests[0])
				missingStatus, missingBody := catalog.App.CallPhysicalAPI(t, outsider, requests[1])
				assert.Equal(t, http.StatusNotFound, status, "%s", body)
				assert.Equal(t, status, missingStatus)
				assert.JSONEq(t, string(missingBody), string(body))
			}

			assert.Equal(t, backupRequests{}, catalog.getBackupRequests(t))
			runningBackupID := catalog.getRunningBackupID(t)
			require.NotNil(t, runningBackupID)
			assert.Equal(t, running.ID, *runningBackupID)

			page := getBackups(t, catalog, "")
			require.Len(t, page.Backups, 4)
			assert.Contains(t, collectBackupIDs(page), full.ID)
			assert.Contains(t, collectBackupIDs(page), running.ID)
			for _, backup := range page.Backups {
				if backup.ID == running.ID {
					assert.Equal(t, "IN_PROGRESS", backup.Status)
				}
				if backup.ID == full.ID {
					assert.Equal(t, "COMPLETED", backup.Status)
				}
			}
		},
	)
	t.Run("returns invalid_request when database operations name a logical database", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		databaseID := catalog.App.InsertDatabase(t, catalog.WorkspaceID)
		for _, request := range []apptest.PhysicalRequest{
			apptest.PhysicalListRequest(databaseID, ""), apptest.PhysicalStartRequest(databaseID, "FULL"),
			apptest.PhysicalStartRequest(databaseID, "AUTO"), apptest.PhysicalStartRequest(databaseID, "INCREMENTAL"),
			apptest.PhysicalPitrTokenRequest(databaseID, nil), apptest.PhysicalStatisticsRequest(databaseID, "HOUR"), apptest.PhysicalLiveRequest(databaseID),
		} {
			status, body := catalog.App.CallPhysicalAPI(t, catalog.Owner, request)
			require.Equal(t, http.StatusBadRequest, status, "%s", body)
			var problem apiclient.Problem
			require.NoError(t, json.Unmarshal(body, &problem))
			assert.Equal(t, apiclient.ProblemCodeInvalidRequest, problem.Code)
		}
	})
}

func TestPhysicalFilteredPaging(t *testing.T) {
	t.Parallel()

	t.Run(
		"keeps the type filter on every page when full backups share a catalog with incrementals and WAL",
		func(t *testing.T) {
			t.Parallel()

			catalog := newCatalog(t)
			full, _, _ := seedOneOfEachType(t, catalog)
			base := catalog.App.Clock.Now().Add(-time.Hour)
			catalog.App.ExecuteSQL(
				t,
				"UPDATE backups_physical SET created_at=? WHERE id=?",
				base.Add(2*time.Minute),
				full.ID,
			)
			additional := catalog.insertFulls(t, fullsSeedSpec{Count: 2, FirstCreatedAt: base})

			first := getBackups(t, catalog, "?type=FULL&limit=2")
			require.Len(t, first.Backups, 2)
			require.NotEmpty(t, first.NextCursor)
			second := getBackups(t, catalog, "?type=FULL&limit=2&after="+first.NextCursor)
			require.Len(t, second.Backups, 1)
			assert.Empty(t, second.NextCursor)

			assert.Equal(t, []uuid.UUID{full.ID, additional[1].ID}, collectBackupIDs(first))
			assert.Equal(t, []uuid.UUID{additional[0].ID}, collectBackupIDs(second))
		},
	)
	t.Run("keeps the time cutoff on every page when newer backups lie after the cutoff", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		base := time.Now().UTC().Add(-time.Hour)
		backups := catalog.insertFulls(t, fullsSeedSpec{Count: 5, FirstCreatedAt: base})

		query := "?limit=2&createdBefore=" + url.QueryEscape(base.Add(3*time.Minute).Format(time.RFC3339Nano))
		first := getBackups(t, catalog, query)
		require.Len(t, first.Backups, 2)
		second := getBackups(t, catalog, query+"&after="+first.NextCursor)
		require.Len(t, second.Backups, 1)

		assert.Empty(t, second.NextCursor)
		assert.Equal(t, []uuid.UUID{backups[2].ID, backups[1].ID, backups[0].ID}, collectBackupIDs(first, second))
	})
	t.Run("keeps the status filter on every page when completed and failed backups are mixed", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		catalog.insertFulls(t, fullsSeedSpec{Count: 2, FirstCreatedAt: time.Now().Add(-time.Hour)})

		failedIDs := []uuid.UUID{}
		for index := range 2 {
			failed := newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StopLSN:    fixtureLSN(segmentBytes),
				},
			)
			failed.Status = "ERROR"
			failed.CreatedAt = time.Now().Add(time.Duration(index) * time.Minute)
			persisted := catalog.insertFull(t, failed)
			failedIDs = append(failedIDs, persisted.ID)
		}

		first := getBackups(t, catalog, "?status=ERROR&limit=1")
		require.Len(t, first.Backups, 1)
		second := getBackups(t, catalog, "?status=ERROR&limit=1&after="+first.NextCursor)
		require.Len(t, second.Backups, 1)
		assert.Empty(t, second.NextCursor)
		assert.ElementsMatch(t, failedIDs, collectBackupIDs(first, second))
	})
}

func TestPhysicalViewerReads(t *testing.T) {
	t.Parallel()

	t.Run("lists backups and issues both token variants when the user is a workspace viewer", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		full, _, _ := seedOneOfEachType(t, catalog)
		viewer := catalog.addViewer(t)
		for _, request := range []apptest.PhysicalRequest{
			apptest.PhysicalListRequest(catalog.DatabaseID, ""),
			apptest.PhysicalStatisticsRequest(catalog.DatabaseID, "HOUR"),
			apptest.PhysicalLiveRequest(catalog.DatabaseID),
		} {
			status, body := catalog.App.CallPhysicalAPI(t, viewer, request)
			require.Equal(t, http.StatusOK, status, "%s", body)
		}
		for _, request := range []apptest.PhysicalRequest{
			apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil), apptest.PhysicalBackupTokenRequest(full.ID),
		} {
			status, body := catalog.App.CallPhysicalAPI(t, viewer, request)
			require.Equal(t, http.StatusCreated, status, "%s", body)
			var token apiclient.PhysicalRestoreToken
			require.NoError(t, json.Unmarshal(body, &token))
			assert.NotEmpty(t, token.Token)
		}
	})
}
