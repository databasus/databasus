package physical_service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	storage_files "databasus-backend/internal/features/storages/files"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/storage"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/walmath"
)

const segmentMB = 16

const (
	firstRecordOffset = physical_testing.FirstRecordOffset
	fullLSNSpan       = physical_testing.FullLSNSpan
)

type serviceTestPrereqs struct {
	storage  *storages.Storage
	database *databases.Database
}

func createServiceTestPrereqs(t *testing.T) *serviceTestPrereqs {
	t.Helper()

	router := workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
	)
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Phys Service "+uuid.NewString(), user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")

	t.Cleanup(func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, database.ID)
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
	})

	return &serviceTestPrereqs{storage: storage, database: database}
}

func saveObject(t *testing.T, st *storages.Storage, fileName string) {
	t.Helper()

	err := st.SaveFile(
		t.Context(),
		encryption.GetFieldEncryptor(),
		logger.GetLogger(),
		fileName,
		strings.NewReader("x"),
	)
	require.NoError(t, err)
}

func objectExists(t *testing.T, st *storages.Storage, fileName string) bool {
	t.Helper()

	reader, err := st.GetFile(t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), fileName)
	if err != nil {
		return false
	}

	_ = reader.Close()

	return true
}

func requireObjectAbsent(t *testing.T, backupStorage *storages.Storage, fileName string) {
	t.Helper()

	reader, err := backupStorage.GetFile(t.Context(), encryption.GetFieldEncryptor(), logger.GetLogger(), fileName)
	if err == nil {
		require.NoError(t, reader.Close())
		require.Failf(t, "storage object still exists", "file_name=%s", fileName)

		return
	}

	require.ErrorContains(t, err, "file not found:")
}

func lsn(segments int) walmath.LSN {
	return walmath.LSN(segments * segmentMB * 1024 * 1024)
}

func Test_DeleteFull_WithDependents_CascadesEntireChainAndObjects(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	manifestName := *fullModel.FileName + ".manifest"
	fullModel.ManifestFileName = &manifestName
	fullModel.BackupSizeMb = new(100.0)
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	firstIncrModel := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, storageID, full.ID, nil, 1, lsn(1), lsn(2))
	firstIncrManifestName := *firstIncrModel.FileName + ".manifest"
	firstIncrModel.ManifestFileName = &firstIncrManifestName
	firstIncr := physical_testing.CreateTestIncrementalBackup(t, firstIncrModel)

	secondIncrModel := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID,
		storageID,
		full.ID,
		&firstIncr.ID,
		1,
		lsn(2),
		lsn(3),
	)
	secondIncrManifestName := *secondIncrModel.FileName + ".manifest"
	secondIncrModel.ManifestFileName = &secondIncrManifestName
	secondIncr := physical_testing.CreateTestIncrementalBackup(t, secondIncrModel)
	incrementals := []*physical_models.PhysicalIncrementalBackup{firstIncr, secondIncr}

	walSegments := []*physical_models.PhysicalWalSegment{
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2)),
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000002", lsn(2), lsn(3)),
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000003", lsn(3), lsn(4)),
	}
	for _, walSegment := range walSegments {
		physical_testing.CreateTestWalSegment(t, walSegment)
	}

	history := physical_testing.CreateTestWalHistoryFile(t,
		physical_testing.NewTestWalHistoryFile(databaseID, storageID, 1))

	// A separate chain on TL2 that must survive the delete.
	successor := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 2, lsn(10), lsn(11)))

	// Materialise the storage objects the cascade is expected to remove.
	saveObject(t, prereqs.storage, *full.FileName)
	saveObject(t, prereqs.storage, manifestName)
	saveObject(t, prereqs.storage, *full.FileName+".metadata")
	for _, incremental := range incrementals {
		saveObject(t, prereqs.storage, *incremental.FileName)
		saveObject(t, prereqs.storage, *incremental.ManifestFileName)
		saveObject(t, prereqs.storage, *incremental.FileName+".metadata")
	}
	for _, walSegment := range walSegments {
		saveObject(t, prereqs.storage, *walSegment.FileName)
		saveObject(t, prereqs.storage, *walSegment.FileName+".metadata")
	}
	saveObject(t, prereqs.storage, history.FileName)
	saveObject(t, prereqs.storage, history.FileName+".metadata")

	summary, err := service.DeleteFull(t.Context(), full.ID, 1_000_000)
	require.NoError(t, err)

	assert.True(t, summary.ChainFullyDeleted)
	assert.Equal(t, 3, summary.WalSegments)
	assert.Equal(t, 2, summary.Incrementals)
	assert.Equal(t, 1, summary.HistoryFiles)

	gotFull, _ := physical_repositories.GetFullBackupRepository().FindByID(full.ID)
	assert.Nil(t, gotFull, "full row must be gone")

	remainingIncrs, _ := physical_repositories.GetIncrementalBackupRepository().FindAllByRootFull(full.ID)
	assert.Empty(t, remainingIncrs, "incr rows must be gone")

	for _, walSegment := range walSegments {
		gotSegment, _ := physical_repositories.GetWalSegmentRepository().FindByID(walSegment.ID)
		assert.Nil(t, gotSegment, "wal row must be gone")
	}

	gotHistory, _ := physical_repositories.GetWalHistoryRepository().FindByDatabaseTimeline(databaseID, 1)
	assert.Nil(t, gotHistory, "history row must be gone")

	// A cascade records the obligations and commits; the worker removes the objects
	// afterwards, so the test drives it to completion before asserting absence.
	names := []string{*full.FileName, manifestName, *full.FileName + ".metadata", history.FileName}
	for _, incremental := range incrementals {
		names = append(names,
			*incremental.FileName, *incremental.ManifestFileName, *incremental.FileName+".metadata")
	}

	for _, walSegment := range walSegments {
		names = append(names, *walSegment.FileName, *walSegment.FileName+".metadata")
	}

	references := make([]storage_files.StoredFileReference, 0, len(names))
	for _, name := range names {
		references = append(references,
			storage_files.StoredFileReference{StorageID: prereqs.storage.ID, FileName: name})
	}

	require.NoError(t, storages.DrainStorageFileDeletions(t.Context(), references...))

	requireObjectAbsent(t, prereqs.storage, *full.FileName)
	requireObjectAbsent(t, prereqs.storage, manifestName)
	requireObjectAbsent(t, prereqs.storage, *full.FileName+".metadata")
	for _, incremental := range incrementals {
		requireObjectAbsent(t, prereqs.storage, *incremental.FileName)
		requireObjectAbsent(t, prereqs.storage, *incremental.ManifestFileName)
		requireObjectAbsent(t, prereqs.storage, *incremental.FileName+".metadata")
	}
	for _, walSegment := range walSegments {
		requireObjectAbsent(t, prereqs.storage, *walSegment.FileName)
		requireObjectAbsent(t, prereqs.storage, *walSegment.FileName+".metadata")
	}
	requireObjectAbsent(t, prereqs.storage, history.FileName)
	requireObjectAbsent(t, prereqs.storage, history.FileName+".metadata")

	survivingSuccessor, _ := physical_repositories.GetFullBackupRepository().FindByID(successor.ID)
	assert.NotNil(t, survivingSuccessor, "successor chain on TL2 must be untouched")
}

// Pruning a chain deletes the WAL it anchored, up to where the next chain begins.
// The next FULL starts at firstRecordOffset inside a segment, so that segment
// begins below the boundary and holds the bytes the surviving chain replays from.
// It belongs to the successor, not to the chain being pruned.
func Test_DeleteFull_WhenSuccessorStartsMidSegment_KeepsSuccessorBoundarySegment(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	prunedFull := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1)))
	physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1,
			lsn(4)+firstRecordOffset, lsn(4)+firstRecordOffset+fullLSNSpan))

	anchoredSegment := physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2)))
	successorBoundarySegment := physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000004", lsn(4), lsn(5)))

	saveObject(t, prereqs.storage, *anchoredSegment.FileName)
	saveObject(t, prereqs.storage, *successorBoundarySegment.FileName)

	_, err := service.DeleteFull(t.Context(), prunedFull.ID, 1_000_000)
	require.NoError(t, err)

	reloadedAnchoredSegment, _ := physical_repositories.GetWalSegmentRepository().FindByID(anchoredSegment.ID)
	assert.Nil(t, reloadedAnchoredSegment, "WAL the pruned chain anchored goes with it")

	reloadedBoundarySegment, _ := physical_repositories.GetWalSegmentRepository().FindByID(successorBoundarySegment.ID)
	assert.NotNil(t, reloadedBoundarySegment,
		"the segment carrying the surviving FULL's start_lsn must outlive the pruned chain")
	assert.True(t, objectExists(t, prereqs.storage, *successorBoundarySegment.FileName),
		"its stored object must survive too")
}

func Test_DeleteFull_WhenPredecessorSurvives_KeepsSharedWalRestorable(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()
	base := time.Now().UTC().Add(-time.Hour)

	predecessorModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	predecessorModel.CreatedAt = base
	predecessorModel.CompletedAt = new(base.Add(time.Minute))
	predecessor := physical_testing.CreateTestFullBackup(t, predecessorModel)

	successorModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(2), lsn(3))
	successorModel.CreatedAt = base.Add(5 * time.Minute)
	successorModel.CompletedAt = new(base.Add(10 * time.Minute))
	successor := physical_testing.CreateTestFullBackup(t, successorModel)
	saveObject(t, prereqs.storage, *successor.FileName)

	walSegments := []*physical_models.PhysicalWalSegment{
		physical_testing.NewTestWalSegment(
			databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2)),
		physical_testing.NewTestWalSegment(
			databaseID, storageID, 1, "000000010000000000000002", lsn(2), lsn(3)),
		physical_testing.NewTestWalSegment(
			databaseID, storageID, 1, "000000010000000000000003", lsn(3), lsn(4)),
	}
	for walIndex, walSegment := range walSegments {
		walSegment.ReceivedAt = base.Add(time.Duration(4+walIndex*3) * time.Minute)
		physical_testing.CreateTestWalSegment(t, walSegment)
		saveObject(t, prereqs.storage, *walSegment.FileName)
	}

	preview, err := service.GetDependentsSummary(successor.ID)
	require.NoError(t, err)
	assert.Zero(t, preview.WalSegments)

	deleted, err := service.DeleteFull(t.Context(), successor.ID, 1_000_000)
	require.NoError(t, err)
	assert.True(t, deleted.ChainFullyDeleted)
	assert.Zero(t, deleted.WalSegments)

	for _, walSegment := range walSegments {
		reloadedSegment, reloadErr := physical_repositories.GetWalSegmentRepository().FindByID(walSegment.ID)
		require.NoError(t, reloadErr)
		assert.NotNil(t, reloadedSegment)
		assert.True(t, objectExists(t, prereqs.storage, *walSegment.FileName))
	}

	targetTime := base.Add(7 * time.Minute)
	restoreSet, err := chain_view.GetChainViewService().ResolveRestoreSet(databaseID, &targetTime)
	require.NoError(t, err)
	assert.Equal(t, predecessor.ID, restoreSet.RootFull.ID)
	require.Len(t, restoreSet.WalSegments, len(walSegments))
}

func Test_CascadeDelete_WhenFullClaimCommitsConcurrently_KeepsWal(t *testing.T) {
	testCases := []struct {
		name     string
		keepFull bool
	}{
		{name: "delete full", keepFull: false},
		{name: "keep full", keepFull: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			prereqs := createServiceTestPrereqs(t)
			physicalBackupService := physical_service.GetPhysicalBackupService()
			rootFull := physical_testing.CreateTestFullBackup(t,
				physical_testing.NewTestCompletedFullBackup(
					prereqs.database.ID,
					prereqs.storage.ID,
					1,
					lsn(0),
					lsn(1),
				))
			walSegment := physical_testing.CreateTestWalSegment(t,
				physical_testing.NewTestWalSegment(
					prereqs.database.ID,
					prereqs.storage.ID,
					1,
					"000000010000000000000001",
					lsn(1),
					lsn(2),
				))
			saveObject(t, prereqs.storage, *walSegment.FileName)

			claimTransaction := storage.GetDb().Begin()
			require.NoError(t, claimTransaction.Error)
			t.Cleanup(func() { _ = claimTransaction.Rollback().Error })
			require.NoError(t, physical_repositories.AcquireBackupAndOrphanCleanupLock(
				claimTransaction,
				prereqs.database.ID,
			))

			type cascadeDeletionOutcome struct {
				summary physical_service.DeletedSummary
				err     error
			}
			deletionOutcomes := make(chan cascadeDeletionOutcome, 1)
			go func() {
				var summary physical_service.DeletedSummary
				var err error
				if testCase.keepFull {
					summary, err = physicalBackupService.DeleteChainDependentsKeepFull(
						t.Context(),
						rootFull.ID,
						1_000_000,
					)
				} else {
					summary, err = physicalBackupService.DeleteFull(
						t.Context(),
						rootFull.ID,
						1_000_000,
					)
				}
				deletionOutcomes <- cascadeDeletionOutcome{summary: summary, err: err}
			}()

			select {
			case <-deletionOutcomes:
				t.Fatal("cascade deletion must wait while FULL claim creation holds the database lock")
			case <-time.After(100 * time.Millisecond):
			}

			activeFull := physical_testing.NewTestCompletedFullBackup(
				prereqs.database.ID,
				prereqs.storage.ID,
				1,
				lsn(4),
				lsn(5),
			)
			activeFull.Status = physical_enums.PhysicalBackupStatusInProgress
			activeFull.StartLSN = nil
			activeFull.StopLSN = nil
			activeFull.FileName = nil
			activeFull.CompletedAt = nil
			require.NoError(t, claimTransaction.Create(activeFull).Error)

			isClaimed, err := physical_repositories.GetInFlightBackupRepository().Claim(
				claimTransaction,
				physical_repositories.ClaimSpec{
					DatabaseID: prereqs.database.ID,
					BackupType: physical_enums.PhysicalBackupTypeFull,
					BackupID:   activeFull.ID,
				},
			)
			require.NoError(t, err)
			require.True(t, isClaimed)
			require.NoError(t, claimTransaction.Commit().Error)

			deletionOutcome := <-deletionOutcomes
			require.NoError(t, deletionOutcome.err)
			assert.Zero(t, deletionOutcome.summary.WalSegments)

			reloadedWalSegment, err := physical_repositories.GetWalSegmentRepository().FindByID(walSegment.ID)
			require.NoError(t, err)
			assert.NotNil(t, reloadedWalSegment)
			assert.True(t, objectExists(t, prereqs.storage, *walSegment.FileName))

			reloadedFull, err := physical_repositories.GetFullBackupRepository().FindByID(rootFull.ID)
			require.NoError(t, err)
			if testCase.keepFull {
				assert.NotNil(t, reloadedFull)
			} else {
				assert.Nil(t, reloadedFull)
			}
		})
	}
}

func Test_GetDependentsSummary_WhenFullClaimIsActive_MatchesDeleteSet(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	physicalBackupService := physical_service.GetPhysicalBackupService()
	rootFull := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			prereqs.database.ID,
			prereqs.storage.ID,
			1,
			lsn(0),
			lsn(1),
		))
	walSegment := physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			prereqs.database.ID,
			prereqs.storage.ID,
			1,
			"000000010000000000000001",
			lsn(1),
			lsn(2),
		))
	historyFile := physical_testing.CreateTestWalHistoryFile(t,
		physical_testing.NewTestWalHistoryFile(
			prereqs.database.ID,
			prereqs.storage.ID,
			1,
		))
	physical_testing.CreateTestInFlightClaim(
		t,
		prereqs.database.ID,
		uuid.New(),
		physical_enums.PhysicalBackupTypeFull,
	)

	deletionPreview, err := physicalBackupService.GetDependentsSummary(rootFull.ID)
	require.NoError(t, err)
	assert.Zero(t, deletionPreview.WalSegments)
	assert.Zero(t, deletionPreview.HistoryFiles)

	deletionSummary, err := physicalBackupService.DeleteFull(t.Context(), rootFull.ID, 1_000_000)
	require.NoError(t, err)
	assert.Zero(t, deletionSummary.WalSegments)
	assert.Zero(t, deletionSummary.HistoryFiles)
	assert.Equal(t, deletionPreview.WalSegments, deletionSummary.WalSegments)
	assert.Equal(t, deletionPreview.HistoryFiles, deletionSummary.HistoryFiles)

	reloadedWalSegment, err := physical_repositories.GetWalSegmentRepository().FindByID(walSegment.ID)
	require.NoError(t, err)
	assert.NotNil(t, reloadedWalSegment)
	reloadedHistoryFile, err := physical_repositories.GetWalHistoryRepository().FindByDatabaseTimeline(
		prereqs.database.ID,
		historyFile.TimelineID,
	)
	require.NoError(t, err)
	assert.NotNil(t, reloadedHistoryFile)
}

func Test_GetDependentsSummary_WhenUpperSegmentCrossesOwnershipBoundary_MatchesDeleteSet(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	fullModel.BackupSizeMb = new(50.0)
	full := physical_testing.CreateTestFullBackup(t, fullModel)
	physical_testing.CreateTestFullBackup(t, physical_testing.NewTestCompletedFullBackup(
		databaseID,
		storageID,
		1,
		lsn(4)+firstRecordOffset,
		lsn(4)+firstRecordOffset+fullLSNSpan,
	))

	ownedSegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2)))
	boundarySegment := physical_testing.CreateTestWalSegment(t, physical_testing.NewTestWalSegment(
		databaseID, storageID, 1, "000000010000000000000004", lsn(4), lsn(5)))
	saveObject(t, prereqs.storage, *full.FileName)
	saveObject(t, prereqs.storage, *ownedSegment.FileName)
	saveObject(t, prereqs.storage, *boundarySegment.FileName)

	preview, err := service.GetDependentsSummary(full.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, preview.WalSegments)
	assert.Equal(t, 66.0, preview.TotalSizeMB)

	deleted, err := service.DeleteFull(t.Context(), full.ID, 1_000_000)
	require.NoError(t, err)
	assert.Equal(t, preview.WalSegments, deleted.WalSegments)

	remainingBoundary, err := physical_repositories.GetWalSegmentRepository().FindByID(boundarySegment.ID)
	require.NoError(t, err)
	assert.NotNil(t, remainingBoundary)
}

func Test_GetDependentsSummary_ReturnsCountsWithoutDeleting(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	fullModel.BackupSizeMb = new(50.0)
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	physical_testing.CreateTestIncrementalBackup(t,
		physical_testing.NewTestCompletedIncrementalBackup(databaseID, storageID, full.ID, nil, 1, lsn(1), lsn(2)))
	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2)))
	physical_testing.CreateTestWalHistoryFile(t,
		physical_testing.NewTestWalHistoryFile(databaseID, storageID, 1))

	for range 2 {
		summary, err := service.GetDependentsSummary(full.ID)
		require.NoError(t, err)

		assert.Equal(t, 1, summary.WalSegments)
		assert.Equal(t, 1, summary.Incrementals)
		assert.Equal(t, 1, summary.HistoryFiles)
		assert.Positive(t, summary.TotalSizeMB)
	}

	stillThere, _ := physical_repositories.GetFullBackupRepository().FindByID(full.ID)
	assert.NotNil(t, stillThere, "GetDependentsSummary must not delete")
}

func Test_DeleteFull_NullFileNameIncrRow_RemovesRowWithoutError(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1)))

	brokenReason := physical_enums.PhysicalBackupErrorSummarizerOff
	brokenIncr := &physical_models.PhysicalIncrementalBackup{
		DatabaseID:       databaseID,
		StorageID:        storageID,
		RootFullBackupID: full.ID,
		TimelineID:       1,
		Status:           physical_enums.PhysicalBackupStatusChainBroken,
		ErrorReason:      &brokenReason,
		FileName:         nil,
	}
	physical_testing.CreateTestIncrementalBackup(t, brokenIncr)

	summary, err := service.DeleteFull(t.Context(), full.ID, 1_000_000)
	require.NoError(t, err)

	assert.True(t, summary.ChainFullyDeleted)
	assert.Equal(t, 1, summary.Incrementals)

	gotIncr, _ := physical_repositories.GetIncrementalBackupRepository().FindByID(brokenIncr.ID)
	assert.Nil(t, gotIncr, "null-file-name incr row must be gone")
}

func Test_DeleteFull_WhenBudgetExhausted_DeletesWalPartiallyAndKeepsFull(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1)))

	// 5 WAL rows at 16 MB each; budget of 20 MB allows ~2 before the cap.
	for i := 1; i <= 5; i++ {
		physical_testing.CreateTestWalSegment(t,
			physical_testing.NewTestWalSegment(databaseID, storageID, 1,
				"00000001000000000000000"+string(rune('0'+i)), lsn(i), lsn(i+1)))
	}

	summary, err := service.DeleteFull(t.Context(), full.ID, 20)
	require.NoError(t, err)

	assert.False(t, summary.ChainFullyDeleted, "budget cap must leave the FULL in place")
	assert.Positive(t, summary.WalSegments)
	assert.Less(t, summary.WalSegments, 5, "budget must stop before draining all WAL")

	stillThere, _ := physical_repositories.GetFullBackupRepository().FindByID(full.ID)
	assert.NotNil(t, stillThere, "FULL must survive a budget-capped call")
}

func Test_DeleteChainDependentsKeepFull_DropsIncrAndWalKeepsFull(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1)))
	incr := physical_testing.CreateTestIncrementalBackup(t,
		physical_testing.NewTestCompletedIncrementalBackup(databaseID, storageID, full.ID, nil, 1, lsn(1), lsn(2)))
	walSegment := physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2)))

	summary, err := service.DeleteChainDependentsKeepFull(t.Context(), full.ID, 1_000_000)
	require.NoError(t, err)

	assert.False(t, summary.ChainFullyDeleted)
	assert.Equal(t, 1, summary.Incrementals)
	assert.Equal(t, 1, summary.WalSegments)

	survivingFull, _ := physical_repositories.GetFullBackupRepository().FindByID(full.ID)
	assert.NotNil(t, survivingFull, "FULL must be kept")

	goneIncr, _ := physical_repositories.GetIncrementalBackupRepository().FindByID(incr.ID)
	assert.Nil(t, goneIncr, "incr must be dropped")

	goneWal, _ := physical_repositories.GetWalSegmentRepository().FindByID(walSegment.ID)
	assert.Nil(t, goneWal, "wal must be dropped")
}

// --- Last backup time ---

func Test_GetLastBackupTimesByDatabaseIDs_WhenWalIsNewest_ReturnsWalReceivedAt(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	base := time.Now().UTC().Add(-time.Hour)

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	fullModel.CreatedAt = base
	fullModel.CompletedAt = new(base)
	physical_testing.CreateTestFullBackup(t, fullModel)

	incrModel := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, storageID, fullModel.ID, nil, 1, lsn(1), lsn(2))
	incrModel.CreatedAt = base.Add(10 * time.Minute)
	physical_testing.CreateTestIncrementalBackup(t, incrModel)

	walModel := physical_testing.NewTestWalSegment(
		databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2))
	walModel.ReceivedAt = base.Add(20 * time.Minute)
	physical_testing.CreateTestWalSegment(t, walModel)

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs([]uuid.UUID{databaseID})
	require.NoError(t, err)

	assert.WithinDuration(t, walModel.ReceivedAt, lastBackupTimes[databaseID], time.Second)
}

func Test_GetLastBackupTimesByDatabaseIDs_WhenFullIsNewest_ReturnsFullCreatedAt(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	base := time.Now().UTC().Add(-time.Hour)

	walModel := physical_testing.NewTestWalSegment(
		databaseID, storageID, 1, "000000010000000000000001", lsn(1), lsn(2))
	walModel.ReceivedAt = base
	physical_testing.CreateTestWalSegment(t, walModel)

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(2), lsn(3))
	fullModel.CreatedAt = base.Add(20 * time.Minute)
	fullModel.CompletedAt = new(base.Add(20 * time.Minute))
	physical_testing.CreateTestFullBackup(t, fullModel)

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs([]uuid.UUID{databaseID})
	require.NoError(t, err)

	assert.WithinDuration(t, fullModel.CreatedAt, lastBackupTimes[databaseID], time.Second)
}

func Test_GetLastBackupTimesByDatabaseIDs_WhenIncrementalIsNewest_ReturnsIncrementalCreatedAt(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	base := time.Now().UTC().Add(-time.Hour)

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	fullModel.CreatedAt = base
	fullModel.CompletedAt = new(base)
	physical_testing.CreateTestFullBackup(t, fullModel)

	incrModel := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, storageID, fullModel.ID, nil, 1, lsn(1), lsn(2))
	incrModel.CreatedAt = base.Add(30 * time.Minute)
	physical_testing.CreateTestIncrementalBackup(t, incrModel)

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs([]uuid.UUID{databaseID})
	require.NoError(t, err)

	assert.WithinDuration(t, incrModel.CreatedAt, lastBackupTimes[databaseID], time.Second)
}

func Test_GetLastBackupTimesByDatabaseIDs_IgnoresNonCompletedFullIncr(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID
	service := physical_service.GetPhysicalBackupService()

	base := time.Now().UTC().Add(-time.Hour)

	completedFull := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	completedFull.CreatedAt = base
	completedFull.CompletedAt = new(base)
	physical_testing.CreateTestFullBackup(t, completedFull)

	// A newer IN_PROGRESS full must not count as the last successful backup.
	inProgressFull := physical_testing.NewTestInProgressFullBackup(databaseID, storageID, 1)
	inProgressFull.CreatedAt = base.Add(40 * time.Minute)
	physical_testing.CreateTestFullBackup(t, inProgressFull)

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs([]uuid.UUID{databaseID})
	require.NoError(t, err)

	assert.WithinDuration(t, completedFull.CreatedAt, lastBackupTimes[databaseID], time.Second)
}

func Test_GetLastBackupTimesByDatabaseIDs_NoBackups_OmitsDatabase(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	service := physical_service.GetPhysicalBackupService()

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs([]uuid.UUID{prereqs.database.ID})
	require.NoError(t, err)

	_, hasBackup := lastBackupTimes[prereqs.database.ID]
	assert.False(t, hasBackup, "a database with no backups must be absent from the map")
}

func Test_GetLastBackupTimesByDatabaseIDs_MultipleDatabases_KeyedPerDatabase(t *testing.T) {
	first := createServiceTestPrereqs(t)
	second := createServiceTestPrereqs(t)
	service := physical_service.GetPhysicalBackupService()

	base := time.Now().UTC().Add(-time.Hour)

	firstFull := physical_testing.NewTestCompletedFullBackup(first.database.ID, first.storage.ID, 1, lsn(0), lsn(1))
	firstFull.CreatedAt = base
	firstFull.CompletedAt = new(base)
	physical_testing.CreateTestFullBackup(t, firstFull)

	secondFull := physical_testing.NewTestCompletedFullBackup(second.database.ID, second.storage.ID, 1, lsn(0), lsn(1))
	secondFull.CreatedAt = base.Add(25 * time.Minute)
	secondFull.CompletedAt = new(base.Add(25 * time.Minute))
	physical_testing.CreateTestFullBackup(t, secondFull)

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs(
		[]uuid.UUID{first.database.ID, second.database.ID})
	require.NoError(t, err)

	assert.WithinDuration(t, firstFull.CreatedAt, lastBackupTimes[first.database.ID], time.Second)
	assert.WithinDuration(t, secondFull.CreatedAt, lastBackupTimes[second.database.ID], time.Second)
}

func Test_GetLastBackupTimesByDatabaseIDs_EmptyInput_ReturnsEmptyMap(t *testing.T) {
	service := physical_service.GetPhysicalBackupService()

	lastBackupTimes, err := service.GetLastBackupTimesByDatabaseIDs(nil)
	require.NoError(t, err)

	assert.NotNil(t, lastBackupTimes)
	assert.Empty(t, lastBackupTimes)
}

func Test_OnBeforeDatabaseRemove_HandsEveryPhysicalArtifactBackToCleanup(t *testing.T) {
	prereqs := createServiceTestPrereqs(t)
	databaseID := prereqs.database.ID
	storageID := prereqs.storage.ID

	fullModel := physical_testing.NewTestCompletedFullBackup(databaseID, storageID, 1, lsn(0), lsn(1))
	manifestName := *fullModel.FileName + ".manifest"
	fullModel.ManifestFileName = &manifestName
	full := physical_testing.CreateTestFullBackup(t, fullModel)

	history := physical_testing.CreateTestWalHistoryFile(t,
		physical_testing.NewTestWalHistoryFile(databaseID, storageID, 1))

	names := []string{*full.FileName, manifestName, *full.FileName + ".metadata", history.FileName}
	for _, name := range names {
		saveObject(t, prereqs.storage, name)
	}

	// Every physical table cascades on databases.id, so this listener is the last
	// moment the names exist anywhere.
	require.NoError(t,
		physical_service.GetPhysicalBackupService().OnBeforeDatabaseRemove(t.Context(), databaseID))

	references := make([]storage_files.StoredFileReference, 0, len(names))
	for _, name := range names {
		references = append(references,
			storage_files.StoredFileReference{StorageID: storageID, FileName: name})
	}

	require.NoError(t, storages.DrainStorageFileDeletions(t.Context(), references...))

	for _, name := range names {
		requireObjectAbsent(t, prereqs.storage, name)
	}
}
