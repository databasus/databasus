package backuping

import (
	"context"
	"fmt"
	"log/slog"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/timelines"
)

type historyPublisher func(ctx context.Context, timelineID int) (*catalog.WalHistoryFile, error)

type promotionContinuationSpec struct {
	Extension      chains.ChainExtension
	Decision       *timelines.TimelineDecision
	PublishHistory historyPublisher
	LoadHistories  func(context.Context) ([]*catalog.WalHistoryFile, error)
	Logger         *slog.Logger
}

// continueAcrossPromotion decides an INCREMENTAL on a timeline newer than the chain's newest
// backup. It publishes the history of the live timeline and of every parent timeline up to the
// newest backup's before deciding, so the decision and later restores read the same links. The
// chain continues only when its line reaches the live timeline; otherwise the decision stays
// TimelineFailoverDetected and the INCREMENTAL breaks the chain.
func continueAcrossPromotion(
	ctx context.Context,
	spec promotionContinuationSpec,
) (*timelines.TimelineDecision, error) {
	extension, decision := spec.Extension, spec.Decision

	if err := publishLineHistories(
		ctx,
		spec.PublishHistory,
		extension.NewestBackup.TimelineID,
		decision.LiveTimelineID,
	); err != nil {
		spec.Logger.WarnContext(
			ctx,
			"timeline history publication failed; the incremental cannot continue the chain across the promotion",
			slog.Int("live_timeline_id", decision.LiveTimelineID),
			slog.Any("error", err),
		)

		return decision, nil
	}

	histories, err := spec.LoadHistories(ctx)
	if err != nil {
		return nil, fmt.Errorf("load timeline histories: %w", err)
	}

	continued := decideContinuationAcrossPromotion(histories, extension, decision)
	if continued.Kind != timelines.TimelineContinueAcrossPromotion {
		spec.Logger.WarnContext(
			ctx,
			"the chain's timeline line does not reach the live timeline",
			slog.Int("newest_backup_timeline_id", extension.NewestBackup.TimelineID),
			slog.Int("live_timeline_id", decision.LiveTimelineID),
		)
	}

	return continued, nil
}

func decideContinuationAcrossPromotion(
	histories []*catalog.WalHistoryFile,
	extension chains.ChainExtension,
	decision *timelines.TimelineDecision,
) *timelines.TimelineDecision {
	if !chains.CanExtendChain(histories, extension) {
		return decision
	}

	continued := *decision
	continued.Kind = timelines.TimelineContinueAcrossPromotion

	return &continued
}

// publishLineHistories publishes the live timeline's history, then the history of each parent
// timeline it names, down to the newest backup's timeline. A link without a usable fork point ends
// the walk: the line cannot cross it, which the line check then reports.
func publishLineHistories(
	ctx context.Context,
	publish historyPublisher,
	newestBackupTimelineID, liveTimelineID int,
) error {
	timelineID := liveTimelineID
	for timelineID > newestBackupTimelineID {
		history, err := publish(ctx, timelineID)
		if err != nil {
			return fmt.Errorf("publish timeline %d history: %w", timelineID, err)
		}

		if !history.HasUsableFork() {
			return nil
		}

		timelineID = *history.ParentTimelineID
	}

	return nil
}
