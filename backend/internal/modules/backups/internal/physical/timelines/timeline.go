package timelines

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/databases"
)

type TimelineDecisionKind int

const (
	TimelineContinue TimelineDecisionKind = iota
	TimelineFailoverDetected
	TimelineRegression
	TimelineDifferentCluster
)

type TimelineDecision struct {
	Kind                     TimelineDecisionKind
	ExpectedTimelineID       int
	LiveTimelineID           int
	ExpectedSystemIdentifier string
	LiveSystemIdentifier     string
}

type timelineComparison struct {
	ExpectedSystemIdentifier *string
	LiveTimelineID           int
	LiveSystemIdentifier     string
	ExpectedTimelineID       int
}

func CheckFullTimelineCompatibility(
	ctx context.Context,
	conn *pgphysical.Inspection,
	db databases.PostgresqlPhysicalConnectionSettings,
	backupRepo *catalog.BackupRepository,
	historyRepo *catalog.WalHistoryRepository,
) (*TimelineDecision, error) {
	liveTimelineID, liveSystemIdentifier, err := conn.ReadLiveTimeline(ctx)
	if err != nil {
		return nil, err
	}

	decision := decideTimelineCompatibility(timelineComparison{
		ExpectedSystemIdentifier: db.SystemIdentifier,
		LiveTimelineID:           liveTimelineID,
		LiveSystemIdentifier:     liveSystemIdentifier,
	})
	if decision.Kind == TimelineDifferentCluster {
		return decision, nil
	}

	knownTimelineID, err := newestKnownTimeline(ctx, db.DatabaseID, backupRepo, historyRepo)
	if err != nil {
		return nil, err
	}

	return decideTimelineCompatibility(timelineComparison{
		ExpectedSystemIdentifier: db.SystemIdentifier,
		LiveTimelineID:           liveTimelineID,
		LiveSystemIdentifier:     liveSystemIdentifier,
		ExpectedTimelineID:       knownTimelineID,
	}), nil
}

func CheckIncrementalTimelineCompatibility(
	ctx context.Context,
	conn *pgphysical.Inspection,
	db databases.PostgresqlPhysicalConnectionSettings,
	rootFullTimelineID int,
) (*TimelineDecision, error) {
	liveTimelineID, liveSystemIdentifier, err := conn.ReadLiveTimeline(ctx)
	if err != nil {
		return nil, err
	}

	return decideTimelineCompatibility(timelineComparison{
		ExpectedSystemIdentifier: db.SystemIdentifier,
		LiveTimelineID:           liveTimelineID,
		LiveSystemIdentifier:     liveSystemIdentifier,
		ExpectedTimelineID:       rootFullTimelineID,
	}), nil
}

func decideTimelineCompatibility(comparison timelineComparison) *TimelineDecision {
	decision := &TimelineDecision{
		Kind:                     TimelineContinue,
		ExpectedTimelineID:       comparison.ExpectedTimelineID,
		LiveTimelineID:           comparison.LiveTimelineID,
		LiveSystemIdentifier:     comparison.LiveSystemIdentifier,
		ExpectedSystemIdentifier: "",
	}

	if comparison.ExpectedSystemIdentifier != nil {
		decision.ExpectedSystemIdentifier = *comparison.ExpectedSystemIdentifier
		if *comparison.ExpectedSystemIdentifier != comparison.LiveSystemIdentifier {
			decision.Kind = TimelineDifferentCluster

			return decision
		}
	}

	if comparison.ExpectedTimelineID == 0 {
		return decision
	}

	if comparison.LiveTimelineID < comparison.ExpectedTimelineID {
		decision.Kind = TimelineRegression

		return decision
	}

	if comparison.LiveTimelineID > comparison.ExpectedTimelineID {
		decision.Kind = TimelineFailoverDetected
	}

	return decision
}

func newestKnownTimeline(
	ctx context.Context,
	dbID uuid.UUID,
	backupRepo *catalog.BackupRepository,
	historyRepo *catalog.WalHistoryRepository,
) (int, error) {
	fulls, err := backupRepo.FindCompletedFullNewestFirstByDatabase(ctx, dbID)
	if err != nil {
		return 0, fmt.Errorf("load full backups: %w", err)
	}

	historyRows, err := historyRepo.FindAllByDatabase(ctx, dbID)
	if err != nil {
		return 0, fmt.Errorf("load history rows: %w", err)
	}

	maxTL := 0
	for _, full := range fulls {
		if full.TimelineID > maxTL {
			maxTL = full.TimelineID
		}
	}

	for _, row := range historyRows {
		if row.TimelineID > maxTL {
			maxTL = row.TimelineID
		}
	}

	return maxTL, nil
}
