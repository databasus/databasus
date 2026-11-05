package timelines

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/engines/postgresql/physical/walmath"
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

// ValidateStartLsnAgainstHistory checks that startLSN falls inside the LSN
// range covered by timelineID per the catalog's .history files. Skipped
// when no history rows exist (first FULL on a fresh DB). OK_WITH_WARNING
// when history is gapped (TL N+1 present but N missing); the caller logs
// timeline_history_gap. CHAIN_BROKEN means the FULL is on a stale or
// rewound timeline whose history moved on — refuse the artifact.
func ValidateStartLsnAgainstHistory(
	ctx context.Context,
	dbID uuid.UUID,
	timelineID int,
	startLSN walmath.LSN,
	historyRepo *catalog.WalHistoryRepository,
) (ValidationResult, error) {
	historyRows, err := historyRepo.FindAllByDatabase(ctx, dbID)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("load wal history rows: %w", err)
	}

	if len(historyRows) == 0 {
		return ValidationResult{Status: ValidationStatusOK}, nil
	}

	if !historyHasTimeline(historyRows, timelineID) && timelineID > 1 {
		if historyIsGapped(historyRows, timelineID) {
			return ValidationResult{
				Status: ValidationStatusOKWithWarning,
				Message: fmt.Sprintf(
					"timeline_history_gap: TL %d has no history row but higher TLs do; older history likely retention-deleted",
					timelineID,
				),
			}, nil
		}

		return ValidationResult{
			Status: ValidationStatusChainBroken,
			Message: fmt.Sprintf(
				"start_lsn %s outside known timeline range: no history record for TL %d",
				startLSN.String(),
				timelineID,
			),
		}, nil
	}

	return ValidationResult{Status: ValidationStatusOK}, nil
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

func historyHasTimeline(rows []*catalog.WalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID == timelineID {
			return true
		}
	}

	return false
}

func historyIsGapped(rows []*catalog.WalHistoryFile, timelineID int) bool {
	for _, row := range rows {
		if row.TimelineID > timelineID {
			return true
		}
	}

	return false
}

type ValidationStatus string

const (
	ValidationStatusOK            ValidationStatus = "OK"
	ValidationStatusOKWithWarning ValidationStatus = "OK_WITH_WARNING"
	ValidationStatusChainBroken   ValidationStatus = "CHAIN_BROKEN"
)

type ValidationResult struct {
	Status  ValidationStatus
	Message string
}
