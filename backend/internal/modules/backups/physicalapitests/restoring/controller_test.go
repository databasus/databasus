package restoring_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	apiclient "databasus-backend/internal/lib/apiclient"
)

func TestRestorePhysicalBackup(t *testing.T) {
	t.Parallel()

	t.Run("issues a token and expiry time when a full backup can be restored", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   fixtureLSN(0),
					StopLSN:    fixtureLSN(segmentBytes),
				},
			),
		)

		var response restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &response,
			},
		)

		assert.NotEmpty(t, response.Token)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), response.ExpiresAt, time.Minute)
	})
	t.Run("returns 409 when the database has no restorable chain", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, nil),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusConflict,
				Destination:   &problem,
			},
		)
		assert.Equal(t, apiclient.ProblemCodeNoRestorableBackup, problem.Code)
	})
	t.Run("issues a token when the requested restore time is reachable", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-2 * time.Hour)

		full := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		full.CompletedAt = new(base)
		catalog.insertFull(t, full)

		incr := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  fixtureLSN(segmentBytes),
				StopLSN:                   fixtureLSN(2 * segmentBytes),
			},
		)
		incr.CompletedAt = new(base.Add(10 * time.Minute))
		catalog.insertIncremental(t, incr)

		walSegment := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000002",
				StartLSN:    fixtureLSN(2 * segmentBytes),
				EndLSN:      fixtureLSN(3 * segmentBytes),
			},
		)
		walSegment.ReceivedAt = base.Add(12 * time.Minute)
		catalog.insertWal(t, walSegment)

		targetTime := base.Add(11 * time.Minute)

		var response restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, &targetTime),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &response,
			},
		)

		assert.NotEmpty(t, response.Token)
	})
	t.Run("issues a token when the target time lies during a newer full backup", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)
		base := time.Now().UTC().Add(-2 * time.Hour)

		olderFull := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		olderFull.CreatedAt = base.Add(-time.Minute)
		olderFull.CompletedAt = new(base)
		catalog.insertFull(t, olderFull)

		newerFull := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(2 * segmentBytes),
				StopLSN:    fixtureLSN(2*segmentBytes + 0x100),
			},
		)
		newerFull.CreatedAt = base.Add(5 * time.Minute)
		newerFull.CompletedAt = new(base.Add(10 * time.Minute))
		catalog.insertFull(t, newerFull)

		for walIndex := 1; walIndex <= 3; walIndex++ {
			walSegment := newWalSeed(
				walSeedSpec{
					DatabaseID:  catalog.DatabaseID,
					StorageID:   catalog.StorageID,
					TimelineID:  1,
					WalFilename: "00000001000000000000000" + string(rune('0'+walIndex)),
					StartLSN:    fixtureLSN(walIndex * segmentBytes),
					EndLSN:      fixtureLSN((walIndex + 1) * segmentBytes),
				},
			)
			walSegment.ReceivedAt = base.Add(time.Duration(3+walIndex*2) * time.Minute)
			catalog.insertWal(t, walSegment)
		}

		targetTime := base.Add(7 * time.Minute)
		var response restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, &targetTime),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &response,
			},
		)

		assert.NotEmpty(t, response.Token)
	})
	t.Run("returns 409 before issuing a token when the target lies beyond a WAL gap", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		base := time.Now().UTC().Add(-2 * time.Hour)

		full := newFullSeed(
			fullSeedSpec{
				DatabaseID: catalog.DatabaseID,
				StorageID:  catalog.StorageID,
				TimelineID: 1,
				StartLSN:   fixtureLSN(0),
				StopLSN:    fixtureLSN(segmentBytes),
			},
		)
		full.CompletedAt = new(base)
		catalog.insertFull(t, full)

		incr := newIncrementalSeed(
			incrementalSeedSpec{
				DatabaseID:                catalog.DatabaseID,
				StorageID:                 catalog.StorageID,
				RootFullBackupID:          full.ID,
				ParentIncrementalBackupID: nil,
				TimelineID:                1,
				StartLSN:                  fixtureLSN(segmentBytes),
				StopLSN:                   fixtureLSN(2 * segmentBytes),
			},
		)
		incr.CompletedAt = new(base.Add(10 * time.Minute))
		catalog.insertIncremental(t, incr)

		w2 := newWalSeed(
			walSeedSpec{
				DatabaseID:  catalog.DatabaseID,
				StorageID:   catalog.StorageID,
				TimelineID:  1,
				WalFilename: "000000010000000000000002",
				StartLSN:    fixtureLSN(2 * segmentBytes),
				EndLSN:      fixtureLSN(3 * segmentBytes),
			},
		)
		w2.ReceivedAt = base.Add(12 * time.Minute)
		catalog.insertWal(t, w2)

		targetTime := base.Add(60 * time.Minute)

		response := catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, &targetTime),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusConflict,
			},
		)

		var problem apiclient.Problem
		require.NoError(t, json.Unmarshal(response.Body, &problem))
		assert.Equal(t, apiclient.ProblemCodeTargetTimeUnreachable, problem.Code)
		assert.Contains(t, problem.Detail, "WAL gap")
	})
	t.Run("issues a token and expiry time when a full backup is selected", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   fixtureLSN(0),
					StopLSN:    fixtureLSN(segmentBytes),
				},
			),
		)

		var response restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &response,
			},
		)

		assert.NotEmpty(t, response.Token)
		assert.WithinDuration(t, time.Now().Add(15*time.Minute), response.ExpiresAt, time.Minute)
	})
	t.Run("returns 404 when the selected backup does not exist", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(uuid.New()),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusNotFound,
			},
		)
	})
	t.Run("returns 401 when the download token is invalid", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalDownloadRequest("not-a-real-token"),
				Authorization: "",
				StatusCode:    http.StatusUnauthorized,
			},
		)
	})
	t.Run("streams a full and its manifest without WAL when a backup token is used", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := createStreamableFullBackup(t, catalog)

		var tokenResponse restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &tokenResponse,
			},
		)

		response := catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalDownloadRequest(tokenResponse.Token),
				Authorization: "",
				StatusCode:    http.StatusOK,
			},
		)

		assert.Equal(t, "application/x-tar", response.Headers.Get("Content-Type"))
		assert.Contains(t, response.Headers.Get("Content-Disposition"), "restore-")

		names := readTarEntryNames(t, response.Body)
		assert.Contains(t, names, "full/base.tar.zst")
		assert.Contains(t, names, "full/backup_manifest")
		for _, name := range names {
			assert.False(t, strings.HasPrefix(name, "wal/"),
				"a per-backup restore must carry no WAL, got %q", name)
		}
	})
	t.Run("returns a runnable POSIX script when the public recovery script is requested", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalScriptRequest(), Authorization: "", StatusCode: http.StatusOK},
		)

		assert.Contains(t, response.Headers.Get("Content-Type"), "text/x-shellscript")
		assert.True(t, strings.HasPrefix(string(response.Body), "#!/bin/sh"),
			"the recovery script must be a runnable POSIX sh script")
		assert.Contains(t, string(response.Body), "pg_combinebackup",
			"the recovery script must reconstruct the cluster")
		assert.Contains(t, string(response.Body), "--pg-bin",
			"the recovery script must accept a PostgreSQL bin path")
		assert.Contains(t, string(response.Body), "--target-time",
			"the recovery script must accept the PITR target time as an argument")
		assert.NotContains(t, string(response.Body), "ZSTD_BIN",
			"WAL is decompressed on the host up front; the script must not wire zstd into restore_command")
	})
	t.Run("explains how to restore configuration when the restored cluster has no config files", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		response := catalog.call(
			t,
			catalogCall{Request: apptest.PhysicalScriptRequest(), Authorization: "", StatusCode: http.StatusOK},
		)

		servedScript := string(response.Body)

		assert.Contains(t, servedScript, "ACTION REQUIRED: the restored cluster has no configuration files",
			"the script must name the failure instead of leaving it to the server's startup error")
		assert.Contains(t, servedScript, "/etc/postgresql/",
			"the instructions must name where Debian/Ubuntu keeps the originals")
		assert.Contains(t, servedScript, "SHOW config_file",
			"the instructions must show how to ask the source cluster for the real paths")
		assert.Contains(t, servedScript, "pg_lsclusters",
			"the instructions must show how to tell two clusters on one host apart")

		assert.NotContains(t, servedScript, "trust",
			"the script must never emit a trust rule for the restored cluster")
	})
	t.Run("rejects a second download when its single-use token was consumed", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := createStreamableFullBackup(t, catalog)

		var tokenResponse restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &tokenResponse,
			},
		)

		streamRequest := apptest.PhysicalDownloadRequest(tokenResponse.Token)

		catalog.call(t, catalogCall{Request: streamRequest, Authorization: "", StatusCode: http.StatusOK})
		catalog.call(t, catalogCall{Request: streamRequest, Authorization: "", StatusCode: http.StatusUnauthorized})
	})
	t.Run("returns 409 when a restore download is already open for the same user", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   fixtureLSN(0),
					StopLSN:    fixtureLSN(segmentBytes),
				},
			),
		)

		var tokenResponse restoreTokenResponse
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusCreated,
				Destination:   &tokenResponse,
			},
		)

		catalog.holdRestoreStream(t)

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalDownloadRequest(tokenResponse.Token),
				Authorization: "",
				StatusCode:    http.StatusConflict,
				Destination:   &problem,
			},
		)
		assert.Equal(t, apiclient.ProblemCodeAlreadyRunning, problem.Code)
	})
	t.Run("refuses another token with 409 when a restore download is already open", func(t *testing.T) {
		t.Parallel()

		catalog := newCatalog(t)

		full := catalog.insertFull(
			t,
			newFullSeed(
				fullSeedSpec{
					DatabaseID: catalog.DatabaseID,
					StorageID:  catalog.StorageID,
					TimelineID: 1,
					StartLSN:   fixtureLSN(0),
					StopLSN:    fixtureLSN(segmentBytes),
				},
			),
		)

		catalog.holdRestoreStream(t)

		var problem apiclient.Problem
		catalog.call(
			t,
			catalogCall{
				Request:       apptest.PhysicalBackupTokenRequest(full.ID),
				Authorization: "Bearer " + catalog.Owner.Token,
				StatusCode:    http.StatusConflict,
				Destination:   &problem,
			},
		)
		assert.Equal(t, apiclient.ProblemCodeAlreadyRunning, problem.Code)
	})
}
