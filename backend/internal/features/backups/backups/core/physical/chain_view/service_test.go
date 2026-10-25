package chain_view_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_enums "databasus-backend/internal/features/backups/backups/core/physical/enums"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_testing "databasus-backend/internal/features/backups/backups/core/physical/testing"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_dto "databasus-backend/internal/features/users/dto"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_models "databasus-backend/internal/features/workspaces/models"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/util/walmath"
)

const segmentBytes = 16 * 1024 * 1024

const (
	firstRecordOffset = physical_testing.FirstRecordOffset
	fullLSNSpan       = physical_testing.FullLSNSpan
)

type chainViewTestPrereqs struct {
	user      *users_dto.SignInResponseDTO
	workspace *workspaces_models.Workspace
	storage   *storages.Storage
	notifier  *notifiers.Notifier
	database  *databases.Database
}

func createChainViewTestPrereqs(t *testing.T) *chainViewTestPrereqs {
	t.Helper()

	router := newChainViewTestRouter()
	user := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Chain View Test "+uuid.NewString(), user, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestPhysicalPostgresDatabase(workspace.ID, notifier, "17")

	t.Cleanup(func() {
		physical_testing.DeleteAllPhysicalCatalogForDatabase(t, database.ID)
		databases.RemoveTestDatabase(t.Context(), database)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
	})

	return &chainViewTestPrereqs{
		user:      user,
		workspace: workspace,
		storage:   storage,
		notifier:  notifier,
		database:  database,
	}
}

func newChainViewTestRouter() *gin.Engine {
	return workspaces_testing.CreateTestRouter(
		workspaces_controllers.GetWorkspaceController(),
		workspaces_controllers.GetMembershipController(),
		databases.GetDatabaseController(),
	)
}

func Test_FindLastExtendableChainByDatabase_WhenChainHasOnlyHealthyFull_ReturnsChain(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(0), walmath.LSN(segmentBytes),
		))

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000001",
			walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	view, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	require.NotNil(t, view)
	assert.Equal(t, full.ID, view.RootFull.ID)
}

func Test_FindLastExtendableChainByDatabase_WhenChainHasBrokenIncr_ReturnsNil(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(0), walmath.LSN(segmentBytes),
		))

	brokenIncr := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes),
	)
	brokenIncr.Status = physical_enums.PhysicalBackupStatusChainBroken
	brokenIncr.ErrorReason = new(physical_enums.PhysicalBackupErrorSummariesExpired)

	physical_testing.CreateTestIncrementalBackup(t, brokenIncr)

	service := chain_view.GetChainViewService()

	view, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	assert.Nil(t, view, "broken chain must not be returned as active")

	nonExtendable, err := service.FindNonExtendableChainsByDatabase(databaseID)
	require.NoError(t, err)
	require.Len(t, nonExtendable, 1)
	assert.Equal(t, full.ID, nonExtendable[0].RootFull.ID)
}

func Test_FindWalGapsInChain_WhenMidChainGapExists_ReturnsGapAndChainStaysExtendable(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(0), walmath.LSN(segmentBytes),
		))

	seg1End := walmath.LSN(2 * segmentBytes)
	seg2Start := walmath.LSN(4 * segmentBytes)

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000001",
			walmath.LSN(segmentBytes), seg1End,
		))
	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000003",
			seg2Start, walmath.LSN(5*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	view, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	require.NotNil(t, view,
		"WAL gaps must not break the chain — chain stays extendable, gap is only an unreachable PITR window")
	assert.Equal(t, full.ID, view.RootFull.ID)

	gaps, err := service.FindWalGapsInChain(full.ID)
	require.NoError(t, err)
	require.Len(t, gaps, 1)
	assert.Equal(t, seg1End, gaps[0].Start)
	assert.Equal(t, seg2Start, gaps[0].End)
}

func Test_FindLastExtendableChainByDatabase_WhenNewerFullExists_ReturnsNewestAndDemotesOlder(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	olderFull := physical_testing.NewTestCompletedFullBackup(
		databaseID, prereqs.storage.ID, 1,
		walmath.LSN(0), walmath.LSN(segmentBytes),
	)
	olderFull.CreatedAt = time.Now().UTC().Add(-time.Hour)
	olderFull.CompletedAt = new(olderFull.CreatedAt.Add(time.Minute))
	physical_testing.CreateTestFullBackup(t, olderFull)

	newerFull := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(4*segmentBytes), walmath.LSN(5*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	activeView, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	require.NotNil(t, activeView)
	assert.Equal(t, newerFull.ID, activeView.RootFull.ID,
		"newer COMPLETED FULL is the extendable head; older one is implicitly closed")

	closedViews, err := service.FindNonExtendableChainsByDatabase(databaseID)
	require.NoError(t, err)
	require.Len(t, closedViews, 1)
	assert.Equal(t, olderFull.ID, closedViews[0].RootFull.ID)
}

func Test_GetChainSpan_WhenSuccessorFullExists_EndIsSuccessorStartLSN(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	olderStart := walmath.LSN(0)
	olderStop := walmath.LSN(segmentBytes)
	newerStart := walmath.LSN(4 * segmentBytes)
	newerStop := walmath.LSN(5 * segmentBytes)

	olderFull := physical_testing.NewTestCompletedFullBackup(
		databaseID, prereqs.storage.ID, 1, olderStart, olderStop,
	)
	olderFull.CreatedAt = time.Now().UTC().Add(-time.Hour)
	olderFull.CompletedAt = new(olderFull.CreatedAt.Add(time.Minute))
	physical_testing.CreateTestFullBackup(t, olderFull)

	physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1, newerStart, newerStop,
		))

	service := chain_view.GetChainViewService()

	span, err := service.GetChainSpan(olderFull.ID)
	require.NoError(t, err)
	assert.Equal(t, olderStart, span.Start)
	assert.Equal(t, newerStart, span.End, "older FULL's span ends at next FULL's start_lsn")

	walInSpan, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, span.Start, span.End,
	)
	require.NoError(t, err)
	assert.Empty(t, walInSpan)
}

func Test_GetChainSpan_WhenNoSuccessorFull_EndIsLSNMax(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	fullStart := walmath.LSN(4 * segmentBytes)
	fullStop := walmath.LSN(5 * segmentBytes)

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1, fullStart, fullStop,
		))

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000005",
			fullStop, walmath.LSN(6*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	span, err := service.GetChainSpan(full.ID)
	require.NoError(t, err)
	assert.Equal(t, fullStart, span.Start)
	assert.Equal(t, chain_view.LSNMax, span.End, "latest FULL has unbounded span")

	walInSpan, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, span.Start, span.End,
	)
	require.NoError(t, err)
	assert.Len(t, walInSpan, 1, "WAL after the FULL's stop_lsn belongs to its open-ended span")
}

func Test_CheckHistoryFilePresence_WhenNoHistoryRows_ReturnsOk(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)

	service := chain_view.GetChainViewService()

	result, err := service.CheckHistoryFilePresence(prereqs.database.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, chain_view.ValidationStatusOK, result.Status)
	assert.Empty(t, result.Message)
}

func Test_CheckHistoryFilePresence_WhenTimelineHasNoHistoryButOthersExist_ReturnsWarning(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	physical_testing.CreateTestWalHistoryFile(t,
		physical_testing.NewTestWalHistoryFile(databaseID, prereqs.storage.ID, 1))
	physical_testing.CreateTestWalHistoryFile(t,
		physical_testing.NewTestWalHistoryFile(databaseID, prereqs.storage.ID, 3))

	service := chain_view.GetChainViewService()

	result, err := service.CheckHistoryFilePresence(databaseID, 2)
	require.NoError(t, err)
	assert.Equal(t, chain_view.ValidationStatusOKWithWarning, result.Status)
	assert.Contains(t, result.Message, "timeline 2")
}

func Test_GetChainEndTimestamp_WhenWalReceivedAfterIncr_ReturnsWalReceivedAt(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	walReceived := time.Now().UTC().Add(-1 * time.Minute)

	full := physical_testing.NewTestCompletedFullBackup(
		databaseID, prereqs.storage.ID, 1,
		walmath.LSN(0), walmath.LSN(segmentBytes),
	)
	full.CompletedAt = new(time.Now().UTC().Add(-2 * time.Hour))
	physical_testing.CreateTestFullBackup(t, full)

	incr := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes),
	)
	incr.CompletedAt = new(time.Now().UTC().Add(-30 * time.Minute))
	physical_testing.CreateTestIncrementalBackup(t, incr)

	seg := physical_testing.NewTestWalSegment(
		databaseID, prereqs.storage.ID, 1, "000000010000000000000001",
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes),
	)
	seg.ReceivedAt = walReceived
	physical_testing.CreateTestWalSegment(t, seg)

	service := chain_view.GetChainViewService()

	view, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	require.NotNil(t, view)

	end, err := service.GetChainEndTimestamp(view)
	require.NoError(t, err)
	assert.WithinDuration(t, walReceived, end, time.Second,
		"WAL.ReceivedAt is the latest of the three; should win the max")
}

func Test_FindWalOrphansByDatabase_WhenWalOutsideAllChainSpans_ReturnsOrphan(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(4*segmentBytes), walmath.LSN(5*segmentBytes),
		))

	orphanSeg := physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000001",
			walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes),
		))

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000005",
			walmath.LSN(5*segmentBytes), walmath.LSN(6*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	orphans, err := service.FindWalOrphansByDatabase(databaseID)
	require.NoError(t, err)
	require.Len(t, orphans, 1,
		"WAL before the only FULL's start_lsn has no chain anchor and is an orphan")
	assert.Equal(t, orphanSeg.ID, orphans[0].WalSegment.ID)
}

// A FULL whose start_lsn lands exactly on a segment boundary is the one alignment
// where comparing it against a segment's start_lsn happens to be right. Real FULLs
// start at firstRecordOffset into the segment, so the segment that physically
// holds the FULL's start and stop positions begins below it and is not an orphan.
func Test_FindWalOrphansByDatabase_WhenFullStartsMidSegment_BoundarySegmentIsNotOrphan(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(4*segmentBytes+firstRecordOffset),
			walmath.LSN(4*segmentBytes+firstRecordOffset+fullLSNSpan),
		))

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000004",
			walmath.LSN(4*segmentBytes), walmath.LSN(5*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	orphans, err := service.FindWalOrphansByDatabase(databaseID)
	require.NoError(t, err)
	assert.Empty(t, orphans,
		"the segment carrying the FULL's own start_lsn is referenced by that FULL")
}

func Test_FindWalGapsInChain_WhenMultipleGapsExist_ReturnsAllGaps(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	full := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(0), walmath.LSN(segmentBytes),
		))

	seg1End := walmath.LSN(2 * segmentBytes)
	seg2Start := walmath.LSN(4 * segmentBytes)
	seg2End := walmath.LSN(5 * segmentBytes)
	seg3Start := walmath.LSN(7 * segmentBytes)

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000001",
			walmath.LSN(segmentBytes), seg1End,
		))
	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000003",
			seg2Start, seg2End,
		))
	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 1, "000000010000000000000006",
			seg3Start, walmath.LSN(8*segmentBytes),
		))

	service := chain_view.GetChainViewService()

	gaps, err := service.FindWalGapsInChain(full.ID)
	require.NoError(t, err)
	require.Len(t, gaps, 2)
	assert.Equal(t, seg1End, gaps[0].Start)
	assert.Equal(t, seg2Start, gaps[0].End)
	assert.Equal(t, seg2End, gaps[1].Start)
	assert.Equal(t, seg3Start, gaps[1].End)
}

func Test_GetChainEndTimestamp_WhenChainHasNoWal_ReturnsLatestBackupCompletion(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	full := physical_testing.NewTestCompletedFullBackup(
		databaseID, prereqs.storage.ID, 1,
		walmath.LSN(0), walmath.LSN(segmentBytes),
	)
	full.CompletedAt = new(time.Now().UTC().Add(-2 * time.Hour))
	physical_testing.CreateTestFullBackup(t, full)

	incrCompletedAt := time.Now().UTC().Add(-30 * time.Minute)
	incr := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, prereqs.storage.ID, full.ID, nil, 1,
		walmath.LSN(segmentBytes), walmath.LSN(2*segmentBytes),
	)
	incr.CompletedAt = new(incrCompletedAt)
	physical_testing.CreateTestIncrementalBackup(t, incr)

	service := chain_view.GetChainViewService()

	view, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	require.NotNil(t, view)

	end, err := service.GetChainEndTimestamp(view)
	require.NoError(t, err)
	assert.WithinDuration(t, incrCompletedAt, end, time.Second)
}

func Test_FindNonExtendableChainsByDatabase_WhenNewestChainBroken_ListsItWithOlderChains(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	olderFull := physical_testing.NewTestCompletedFullBackup(
		databaseID, prereqs.storage.ID, 1,
		walmath.LSN(0), walmath.LSN(segmentBytes),
	)
	olderFull.CreatedAt = time.Now().UTC().Add(-time.Hour)
	olderFull.CompletedAt = new(olderFull.CreatedAt.Add(time.Minute))
	physical_testing.CreateTestFullBackup(t, olderFull)

	newerFull := physical_testing.CreateTestFullBackup(t,
		physical_testing.NewTestCompletedFullBackup(
			databaseID, prereqs.storage.ID, 1,
			walmath.LSN(4*segmentBytes), walmath.LSN(5*segmentBytes),
		))

	brokenIncr := physical_testing.NewTestCompletedIncrementalBackup(
		databaseID, prereqs.storage.ID, newerFull.ID, nil, 1,
		walmath.LSN(5*segmentBytes), walmath.LSN(6*segmentBytes),
	)
	brokenIncr.Status = physical_enums.PhysicalBackupStatusChainBroken
	brokenIncr.ErrorReason = new(physical_enums.PhysicalBackupErrorSummariesExpired)
	physical_testing.CreateTestIncrementalBackup(t, brokenIncr)

	service := chain_view.GetChainViewService()

	extendableView, err := service.FindLastExtendableChainByDatabase(databaseID)
	require.NoError(t, err)
	assert.Nil(t, extendableView)

	closedViews, err := service.FindNonExtendableChainsByDatabase(databaseID)
	require.NoError(t, err)
	require.Len(t, closedViews, 2)
	assert.Equal(t, newerFull.ID, closedViews[0].RootFull.ID)
	assert.Equal(t, olderFull.ID, closedViews[1].RootFull.ID)
}

func Test_FindByChainSpan_ForMiddleChain_ReturnsOnlyItsSegmentsIncludingBoundary(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	fullIDsBySegmentIndex := make(map[int]uuid.UUID)

	for _, segmentIndex := range []int{0, 4, 8} {
		fullStart := walmath.LSN(segmentIndex*segmentBytes + firstRecordOffset)
		full := physical_testing.CreateTestFullBackup(t,
			physical_testing.NewTestCompletedFullBackup(
				databaseID, prereqs.storage.ID, 1, fullStart, fullStart+fullLSNSpan,
			))

		fullIDsBySegmentIndex[segmentIndex] = full.ID
	}

	segmentIDsByIndex := createContiguousWalSegments(t, prereqs, databaseID, 1, 0, 10)

	middleSpan, err := chain_view.GetChainViewService().GetChainSpan(fullIDsBySegmentIndex[4])
	require.NoError(t, err)

	middleSegments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, middleSpan.Start, middleSpan.End,
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]uuid.UUID{
			segmentIDsByIndex[4], segmentIDsByIndex[5], segmentIDsByIndex[6],
			segmentIDsByIndex[7], segmentIDsByIndex[8],
		},
		getWalSegmentIDs(middleSegments),
		"the segment holding the middle FULL's start begins below the span and is included; "+
			"segments of the first chain are not",
	)
}

func Test_FindByChainSpan_WhenSpanStartsOnSegmentBoundary_ExcludesSegmentEndingThere(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	segmentIDsByIndex := createContiguousWalSegments(t, prereqs, databaseID, 1, 3, 6)

	spanSegments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, walmath.LSN(4*segmentBytes), chain_view.LSNMax,
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]uuid.UUID{segmentIDsByIndex[4], segmentIDsByIndex[5]},
		getWalSegmentIDs(spanSegments),
	)
}

func Test_FindByChainSpan_WhenOtherTimelineAndDatabaseHaveSegmentsInRange_IgnoresThem(t *testing.T) {
	t.Parallel()

	prereqs := createChainViewTestPrereqs(t)
	otherPrereqs := createChainViewTestPrereqs(t)
	databaseID := prereqs.database.ID

	segmentIDsByIndex := createContiguousWalSegments(t, prereqs, databaseID, 1, 3, 6)

	spanStart := walmath.LSN(4*segmentBytes + firstRecordOffset)
	closerToSpanStart := walmath.LSN(4*segmentBytes + firstRecordOffset/2)

	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			databaseID, prereqs.storage.ID, 2, "000000020000000000000004",
			closerToSpanStart, closerToSpanStart+segmentBytes,
		))
	physical_testing.CreateTestWalSegment(t,
		physical_testing.NewTestWalSegment(
			otherPrereqs.database.ID, otherPrereqs.storage.ID, 1, "000000010000000000000004",
			closerToSpanStart, closerToSpanStart+segmentBytes,
		))

	spanSegments, err := physical_repositories.GetWalSegmentRepository().FindByChainSpan(
		databaseID, 1, spanStart, chain_view.LSNMax,
	)
	require.NoError(t, err)

	assert.Equal(t,
		[]uuid.UUID{segmentIDsByIndex[4], segmentIDsByIndex[5]},
		getWalSegmentIDs(spanSegments),
		"segments of another timeline or database must not move the boundary segment",
	)
}

func createContiguousWalSegments(
	t *testing.T,
	prereqs *chainViewTestPrereqs,
	databaseID uuid.UUID,
	timelineID int,
	firstSegmentIndex, endSegmentIndex int,
) map[int]uuid.UUID {
	t.Helper()

	segmentIDsByIndex := make(map[int]uuid.UUID)

	for segmentIndex := firstSegmentIndex; segmentIndex < endSegmentIndex; segmentIndex++ {
		segment := physical_testing.CreateTestWalSegment(t,
			physical_testing.NewTestWalSegment(
				databaseID, prereqs.storage.ID, timelineID,
				fmt.Sprintf("%08X%016X", timelineID, segmentIndex),
				walmath.LSN(segmentIndex*segmentBytes), walmath.LSN((segmentIndex+1)*segmentBytes),
			))

		segmentIDsByIndex[segmentIndex] = segment.ID
	}

	return segmentIDsByIndex
}

func getWalSegmentIDs(segments []*physical_models.PhysicalWalSegment) []uuid.UUID {
	segmentIDs := make([]uuid.UUID, 0, len(segments))
	for _, segment := range segments {
		segmentIDs = append(segmentIDs, segment.ID)
	}

	return segmentIDs
}
