package restoring_test

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/app/apptest"
	"databasus-backend/internal/modules/storages"
)

type restoreTimeline struct{ base time.Time }

func newRestoreTimeline() *restoreTimeline {
	return &restoreTimeline{base: time.Now().UTC().Add(-2 * time.Hour)}
}

func (timeline *restoreTimeline) at(minutes int) time.Time {
	return timeline.base.Add(time.Duration(minutes) * time.Minute)
}
func lsnAt(segments int) fixtureLSN { return fixtureLSN(segments * segmentBytes) }
func (catalog *catalogFixture) publish(t *testing.T, name string, body []byte) {
	t.Helper()
	catalog.App.PublishStorageFile(
		t,
		storages.StoredFileReference{StorageID: catalog.StorageID, FileName: name},
		bytes.NewReader(body),
	)
}

type restoreFullSpec struct {
	StartLSN    fixtureLSN
	StopLSN     fixtureLSN
	CompletedAt time.Time
}

func (catalog *catalogFixture) seedRestoreFull(t *testing.T, spec restoreFullSpec) *apptest.PhysicalFullBackupSeed {
	t.Helper()
	full := newFullSeed(
		fullSeedSpec{
			DatabaseID: catalog.DatabaseID,
			StorageID:  catalog.StorageID,
			TimelineID: 1,
			StartLSN:   spec.StartLSN,
			StopLSN:    spec.StopLSN,
		},
	)

	full.ID = uuid.New()
	full.CreatedAt = spec.CompletedAt.Add(-time.Minute)
	full.CompletedAt = &spec.CompletedAt
	full.ManifestFileName = new(*full.FileName + ".manifest")

	blob := zstdTar(t, map[string]string{"PG_VERSION": "17\n", "base/1/1259": "heap-bytes"})
	full.BackupSizeMb = new(float64(len(blob)) / (1024 * 1024))
	catalog.publish(t, *full.FileName, blob)
	catalog.publish(t, *full.ManifestFileName, []byte(full.ID.String()))
	return catalog.insertFull(t, full)
}

type restoreIncrementalSpec struct {
	FullID      uuid.UUID
	ParentID    *uuid.UUID
	Segment     int
	CompletedAt time.Time
}

func (catalog *catalogFixture) seedRestoreIncremental(
	t *testing.T,
	spec restoreIncrementalSpec,
) *apptest.PhysicalIncrementalBackupSeed {
	t.Helper()
	incremental := newIncrementalSeed(
		incrementalSeedSpec{
			DatabaseID:                catalog.DatabaseID,
			StorageID:                 catalog.StorageID,
			TimelineID:                1,
			RootFullBackupID:          spec.FullID,
			ParentIncrementalBackupID: spec.ParentID,
			StartLSN:                  lsnAt(spec.Segment),
			StopLSN:                   lsnAt(spec.Segment + 1),
		},
	)

	incremental.ID = uuid.New()
	incremental.CreatedAt = spec.CompletedAt.Add(-time.Minute)
	incremental.CompletedAt = &spec.CompletedAt
	incremental.ManifestFileName = new(*incremental.FileName + ".manifest")

	blob := zstdTar(t, map[string]string{"PG_VERSION": "17\n"})
	incremental.BackupSizeMb = new(float64(len(blob)) / (1024 * 1024))
	catalog.publish(t, *incremental.FileName, blob)
	catalog.publish(t, *incremental.ManifestFileName, []byte(incremental.ID.String()))
	return catalog.insertIncremental(t, incremental)
}

type restoreWalSpec struct {
	Segment    int
	ReceivedAt time.Time
}

func (catalog *catalogFixture) seedRestoreWal(t *testing.T, spec restoreWalSpec) *apptest.PhysicalWalSegmentSeed {
	t.Helper()
	segment := newWalSeed(
		walSeedSpec{
			DatabaseID:  catalog.DatabaseID,
			StorageID:   catalog.StorageID,
			TimelineID:  1,
			WalFilename: fmt.Sprintf("0000000100000000%08X", spec.Segment),
			StartLSN:    lsnAt(spec.Segment),
			EndLSN:      lsnAt(spec.Segment + 1),
		},
	)
	segment.ReceivedAt = spec.ReceivedAt
	catalog.publish(t, *segment.FileName, []byte("stored WAL "+segment.WalFilename))
	return catalog.insertWal(t, segment)
}

func seedFullTwoIncrementals(
	t *testing.T,
	catalog *catalogFixture,
	timeline *restoreTimeline,
) (*apptest.PhysicalFullBackupSeed, *apptest.PhysicalIncrementalBackupSeed, *apptest.PhysicalIncrementalBackupSeed) {
	t.Helper()
	full := catalog.seedRestoreFull(
		t,
		restoreFullSpec{StartLSN: lsnAt(0), StopLSN: lsnAt(1), CompletedAt: timeline.at(0)},
	)
	first := catalog.seedRestoreIncremental(
		t,
		restoreIncrementalSpec{FullID: full.ID, Segment: 1, CompletedAt: timeline.at(10)},
	)
	second := catalog.seedRestoreIncremental(
		t,
		restoreIncrementalSpec{FullID: full.ID, ParentID: &first.ID, Segment: 2, CompletedAt: timeline.at(20)},
	)
	return full, first, second
}

type restoreRequest struct {
	BackupID   *uuid.UUID
	TargetTime *time.Time
	StatusCode int
}

func (catalog *catalogFixture) downloadRestore(t *testing.T, spec restoreRequest) map[string]string {
	t.Helper()
	request := apptest.PhysicalPitrTokenRequest(catalog.DatabaseID, spec.TargetTime)
	if spec.BackupID != nil {
		request = apptest.PhysicalBackupTokenRequest(*spec.BackupID)
	}

	status := spec.StatusCode
	if status == 0 {
		status = http.StatusCreated
	}

	var token restoreTokenResponse
	catalog.call(
		t,
		catalogCall{
			Request:       request,
			Authorization: "Bearer " + catalog.Owner.Token,
			StatusCode:    status,
			Destination:   &token,
		},
	)

	if status != http.StatusCreated {
		return nil
	}
	response := catalog.call(
		t,
		catalogCall{Request: apptest.PhysicalDownloadRequest(token.Token), StatusCode: http.StatusOK},
	)
	return readTarEntries(t, response.Body)
}

func readTarEntries(t *testing.T, body []byte) map[string]string {
	t.Helper()
	entries := map[string]string{}
	reader := tar.NewReader(bytes.NewReader(body))
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		content, err := io.ReadAll(reader)
		require.NoError(t, err)
		entries[header.Name] = string(content)
	}
	return entries
}

func assertRestoreArtifacts(
	t *testing.T,
	entries map[string]string,
	fullID uuid.UUID,
	incrementalIDs []uuid.UUID,
	walSegments ...int,
) {
	t.Helper()
	assert.Equal(t, fullID.String(), entries["full/backup_manifest"])
	for index, incrementalID := range incrementalIDs {
		assert.Equal(t, incrementalID.String(), entries[fmt.Sprintf("incr-%d/backup_manifest", index+1)])
	}
	assert.NotContains(t, entries, fmt.Sprintf("incr-%d/backup_manifest", len(incrementalIDs)+1))

	walNames := []string{}
	for name := range entries {
		if len(name) > 4 && name[:4] == "wal/" {
			walNames = append(walNames, name)
		}
	}

	includedWalNames := []string{}
	for _, segment := range walSegments {
		includedWalNames = append(includedWalNames, fmt.Sprintf("wal/0000000100000000%08X.zst", segment))
	}
	assert.ElementsMatch(t, includedWalNames, walNames)
}
