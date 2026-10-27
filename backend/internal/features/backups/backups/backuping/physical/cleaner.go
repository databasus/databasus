package backuping_physical

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"slices"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	physical_models "databasus-backend/internal/features/backups/backups/core/physical/models"
	physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	"databasus-backend/internal/features/backups/backups/core/physical/wal_health"
	usecases_physical_postgresql "databasus-backend/internal/features/backups/backups/usecases/physical/postgresql"
	"databasus-backend/internal/lib/gfs"
	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups"
)

// PhysicalBackupCleaner trims physical backups every tick through two passes:
// retention policy and orphan sweep. It never issues raw DELETEs — every removal
// goes through PhysicalBackupService — and it never touches the active
// (extendable) chain: it operates only on the non-extendable set that chain_view
// derives.
type PhysicalBackupCleaner struct {
	physicalBackupService *physical_service.PhysicalBackupService
	chainViewService      *chain_view.ChainViewService
	getBackups            func() backups.Module
	fullRepo              *physical_repositories.PhysicalFullBackupRepository
	walSegmentRepo        *physical_repositories.PhysicalWalSegmentRepository
	walStreamSampleRepo   *physical_repositories.PhysicalWalStreamSampleRepository
	logger                *slog.Logger

	hasRun atomic.Bool
}

func (c *PhysicalBackupCleaner) Run(ctx context.Context) {
	if c.hasRun.Swap(true) {
		panic(fmt.Sprintf("%T.Run() called multiple times", c))
	}

	if ctx.Err() != nil {
		return
	}

	c.runStartupSlotCleanup(ctx)

	ticker := time.NewTicker(cleanerTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			logger := c.logger.With("job_id", uuid.New(), "job_name", cleanerJobName)

			if err := c.cleanByRetentionPolicy(ctx, logger); err != nil {
				logger.ErrorContext(ctx, "failed to clean by retention policy", "error", err)
			}

			if err := c.cleanOrphans(ctx, logger); err != nil {
				logger.ErrorContext(ctx, "failed to clean orphans", "error", err)
			}

			c.deleteExpiredWalStreamSamples(ctx, logger, time.Now().UTC())
		}
	}
}

// One statement for every database, so samples of a database that no longer
// streams expire as well; a removed database's samples go with it by cascade.
func (c *PhysicalBackupCleaner) deleteExpiredWalStreamSamples(ctx context.Context, logger *slog.Logger, now time.Time) {
	deletedCount, err := c.walStreamSampleRepo.DeleteOlderThan(now.Add(-wal_health.SampleRetention))
	if err != nil {
		logger.ErrorContext(ctx, "failed to delete expired wal stream samples", "error", err)

		return
	}

	if deletedCount > 0 {
		logger.DebugContext(ctx, fmt.Sprintf("deleted %d expired wal stream samples", deletedCount))
	}
}

// runStartupSlotCleanup drops orphan per-backup replication slots once at boot.
// In single-instance mode any backup that was running died with the previous
// process, so no slot is protected — the scheduler's restart recovery fails and
// releases every in-flight claim, and the matching slots are reclaimable orphans.
func (c *PhysicalBackupCleaner) runStartupSlotCleanup(ctx context.Context) {
	isSlotProtected := func(uuid.UUID) (bool, error) { return false, nil }

	logger := c.logger.With("job_id", uuid.New(), "job_name", cleanerJobName)

	// Best effort at boot: a slot that cannot be dropped is retried by the orphan sweep.
	if err := usecases_physical_postgresql.RunStartupCleanup(ctx, logger, isSlotProtected); err != nil {
		logger.WarnContext(ctx, "physical startup slot cleanup reported failures", "error", err)
	}
}

func (c *PhysicalBackupCleaner) cleanByRetentionPolicy(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := c.getBackups().ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledConfigs {
		dbLog := logger.With("database_id", backupConfig.DatabaseID, "retention", backupConfig.Retention.Policy)

		var cleanErr error

		switch backupConfig.Retention.Policy {
		case backups.PhysicalRetentionPolicyChains:
			cleanErr = c.cleanByChains(ctx, dbLog, backupConfig)
		case backups.PhysicalRetentionPolicyFullBackups:
			cleanErr = c.cleanByFulls(ctx, dbLog, backupConfig)
		case backups.PhysicalRetentionPolicyChainsAndFullBackups:
			cleanErr = c.cleanByCombined(ctx, dbLog, backupConfig)
		}

		if cleanErr != nil {
			dbLog.Error("failed to clean database by retention policy", "error", cleanErr)
		}
	}

	return nil
}

// nonExtendableChainsNewestEndFirst returns every non-extendable chain for a DB
// sorted by chain-end timestamp, newest first. The active (extendable) head is
// already excluded by FindNonExtendableChainsByDatabase, so nothing here can
// ever delete it.
func (c *PhysicalBackupCleaner) nonExtendableChainsNewestEndFirst(
	databaseID uuid.UUID,
) ([]chainCandidate, error) {
	views, err := c.chainViewService.FindNonExtendableChainsByDatabase(databaseID)
	if err != nil {
		return nil, err
	}

	candidates := make([]chainCandidate, 0, len(views))

	for _, view := range views {
		endTs, err := c.chainViewService.GetChainEndTimestamp(view)
		if err != nil {
			c.logger.Error("failed to get chain end timestamp",
				"root_full_backup_id", view.RootFull.ID, "error", err)

			continue
		}

		candidates = append(candidates, chainCandidate{view: view, endTs: endTs})
	}

	slices.SortFunc(candidates, func(a, b chainCandidate) int {
		return b.endTs.Compare(a.endTs)
	})

	return candidates, nil
}

// Reports whether the delete succeeded, so callers count outcomes rather than attempts.
func (c *PhysicalBackupCleaner) deleteChainThroughService(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig backups.PhysicalBackupConfig,
	rootFullBackupID uuid.UUID,
) bool {
	summary, err := c.physicalBackupService.DeleteFull(
		ctx, rootFullBackupID, c.walDeleteBudgetMB(ctx, backupConfig.DatabaseID),
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to delete chain", "root_full_backup_id", rootFullBackupID, "error", err)

		return false
	}

	logger.InfoContext(ctx, fmt.Sprintf(
		"chain cleanup progress: %d wal, %d incr, %d history, %.1f MB, complete=%v",
		summary.WalSegments, summary.Incrementals, summary.HistoryFiles,
		summary.BytesDeletedMB, summary.ChainFullyDeleted,
	), "root_full_backup_id", rootFullBackupID)

	return true
}

// A kept FULL stays a standalone restore point, so its INCRs and WAL go while it remains.
// Reports whether the delete succeeded, so callers count outcomes rather than attempts.
func (c *PhysicalBackupCleaner) deleteChainDependentsThroughService(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig backups.PhysicalBackupConfig,
	rootFullBackupID uuid.UUID,
) bool {
	summary, err := c.physicalBackupService.DeleteChainDependentsKeepFull(
		ctx, rootFullBackupID, c.walDeleteBudgetMB(ctx, backupConfig.DatabaseID),
	)
	if err != nil {
		logger.ErrorContext(ctx, "failed to delete chain dependents",
			"root_full_backup_id", rootFullBackupID,
			"error", err,
		)

		return false
	}

	logger.InfoContext(ctx, fmt.Sprintf(
		"chain dependents cleanup: %d wal, %d incr, %.1f MB",
		summary.WalSegments, summary.Incrementals, summary.BytesDeletedMB,
	), "root_full_backup_id", rootFullBackupID)

	return true
}

// walDeleteBudgetMB anchors the per-tick WAL byte budget to the latest COMPLETED
// FULL's size, floored at minWalDeleteBudgetMB.
func (c *PhysicalBackupCleaner) walDeleteBudgetMB(ctx context.Context, databaseID uuid.UUID) float64 {
	fulls, err := c.fullRepo.FindCompletedNewestFirstByDatabase(databaseID)
	if err != nil {
		// Falling back to the floor silently would look like retention crawling for no reason.
		c.logger.WarnContext(ctx, fmt.Sprintf(
			"failed to size the wal delete budget, falling back to the %.0f MB floor",
			minWalDeleteBudgetMB), "database_id", databaseID, "error", err)

		return minWalDeleteBudgetMB
	}

	if len(fulls) > 0 && fulls[0].BackupSizeMb != nil {
		return math.Max(*fulls[0].BackupSizeMb, minWalDeleteBudgetMB)
	}

	return minWalDeleteBudgetMB
}

// isChainWithinGrace reports whether a chain is too young to evict: its end
// timestamp is within max(full, incr cadence) × 2 (floored at the 60-minute
// per-backup grace). A chain whose end timestamp failed to load never becomes a
// candidate, so a transient error never causes a premature delete.
func isChainWithinGrace(
	backupConfig backups.PhysicalBackupConfig,
	chainEndTs time.Time,
) bool {
	grace := max(
		approxScheduleDuration(&backupConfig.FullBackupSchedule),
		approxScheduleDuration(backupConfig.IncrementalBackupSchedule),
	) * chainGraceIntervalMultiplier

	grace = max(grace, recentBackupGracePeriod)

	return time.Since(chainEndTs) < grace
}

// cleanByChains keeps the N newest non-extendable chains (by chain-end
// timestamp) and deletes the rest, honoring the per-chain grace period. The
// active chain is never a candidate.
func (c *PhysicalBackupCleaner) cleanByChains(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig backups.PhysicalBackupConfig,
) error {
	keepCount := backupConfig.Retention.ChainsCount
	if keepCount <= 0 {
		logger.DebugContext(ctx, "chain retention has no count configured, keeping every chain")

		return nil
	}

	candidates, err := c.nonExtendableChainsNewestEndFirst(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	if len(candidates) <= keepCount {
		logger.DebugContext(ctx, fmt.Sprintf("chain retention: keeping all %d chains, limit is %d",
			len(candidates), keepCount))

		return nil
	}

	deletedCount, withinGraceCount := 0, 0

	for _, candidate := range candidates[keepCount:] {
		rootFullBackupID := candidate.view.RootFull.ID

		if isChainWithinGrace(backupConfig, candidate.endTs) {
			withinGraceCount++

			logger.DebugContext(ctx, "keeping chain, it is still within the retention grace period",
				"root_full_backup_id", rootFullBackupID)

			continue
		}

		if c.deleteChainThroughService(ctx, logger, backupConfig, rootFullBackupID) {
			deletedCount++
		}
	}

	summary := fmt.Sprintf(
		"chain retention: %d non-extendable chains, %d kept by limit, %d deleted, %d within grace",
		len(candidates), keepCount, deletedCount, withinGraceCount)

	// The cleaner ticks every few seconds, so a run that changed nothing belongs at debug.
	if deletedCount > 0 {
		logger.InfoContext(ctx, summary)
	} else {
		logger.DebugContext(ctx, summary)
	}

	return nil
}

// cleanByFulls dispatches the FULL_BACKUPS retention policy to its LAST_N or GFS
// variant. Both keep a set of FULLs; non-extendable chains rooted at a kept FULL
// shed their INCR + WAL (the FULL stays a standalone restore point), and chains
// rooted at a non-kept FULL are deleted whole.
func (c *PhysicalBackupCleaner) cleanByFulls(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig backups.PhysicalBackupConfig,
) error {
	completedFulls, err := c.fullRepo.FindCompletedNewestFirstByDatabase(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	retainedTiersByFullBackupID := c.getRetainedTiersByFullBackupID(backupConfig, completedFulls)
	if retainedTiersByFullBackupID == nil {
		return nil
	}

	candidates, err := c.nonExtendableChainsNewestEndFirst(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	deletedCount, dependentsOnlyCount, withinGraceCount := 0, 0, 0

	for _, candidate := range candidates {
		rootFullBackupID := candidate.view.RootFull.ID

		if isChainWithinGrace(backupConfig, candidate.endTs) {
			withinGraceCount++

			logger.DebugContext(ctx, "keeping chain, it is still within the retention grace period",
				"root_full_backup_id", rootFullBackupID)

			continue
		}

		if tiers, isKept := retainedTiersByFullBackupID[rootFullBackupID]; isKept {
			logger.DebugContext(ctx, fmt.Sprintf("keeping the full backup%s, shedding its dependents",
				formatRetentionTiersSuffix(tiers)), "root_full_backup_id", rootFullBackupID)

			if c.deleteChainDependentsThroughService(ctx, logger, backupConfig, rootFullBackupID) {
				dependentsOnlyCount++
			}

			continue
		}

		if c.deleteChainThroughService(ctx, logger, backupConfig, rootFullBackupID) {
			deletedCount++
		}
	}

	summary := fmt.Sprintf(
		"full-backup retention: %d chains, %d shed to their full, %d deleted whole, %d within grace",
		len(candidates), dependentsOnlyCount, deletedCount, withinGraceCount)

	if deletedCount+dependentsOnlyCount > 0 {
		logger.InfoContext(ctx, summary)
	} else {
		logger.DebugContext(ctx, summary)
	}

	return nil
}

// Returns nil when the policy has no effective configuration, so the caller no-ops rather than
// treating an empty result as "delete everything".
func (c *PhysicalBackupCleaner) getRetainedTiersByFullBackupID(
	backupConfig backups.PhysicalBackupConfig,
	completedFullsNewestFirst []*physical_models.PhysicalFullBackup,
) map[uuid.UUID][]gfs.Tier {
	retention := backupConfig.Retention.FullBackups

	switch retention.Policy {
	case backups.FullBackupsRetentionPolicyLastN:
		if retention.Count <= 0 {
			return nil
		}

		keep := make(map[uuid.UUID][]gfs.Tier)
		for i, full := range completedFullsNewestFirst {
			if i < retention.Count {
				keep[full.ID] = nil
			}
		}

		return keep

	case backups.FullBackupsRetentionPolicyGfs:
		if retention.Gfs.Hours <= 0 && retention.Gfs.Days <= 0 && retention.Gfs.Weeks <= 0 &&
			retention.Gfs.Months <= 0 && retention.Gfs.Years <= 0 {
			return nil
		}

		items := make([]gfs.Item, len(completedFullsNewestFirst))
		for i, full := range completedFullsNewestFirst {
			items[i] = gfs.Item{ID: full.ID, CreatedAt: full.CreatedAt}
		}

		return gfs.GetItemsToRetain(
			items, retention.Gfs.Hours, retention.Gfs.Days, retention.Gfs.Weeks,
			retention.Gfs.Months, retention.Gfs.Years,
		)
	}

	return nil
}

// cleanByCombined keeps the UNION of the CHAINS keep-set (N newest
// non-extendable chains) and the FULL_BACKUPS keep-set (LAST_N or GFS over
// FULLs), deleting every other non-extendable chain whole. A chain in either
// keep-set is preserved entirely (INCR + WAL included).
func (c *PhysicalBackupCleaner) cleanByCombined(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig backups.PhysicalBackupConfig,
) error {
	candidates, err := c.nonExtendableChainsNewestEndFirst(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	keepRoots := make(map[uuid.UUID]bool)

	for i, candidate := range candidates {
		if i < backupConfig.Retention.ChainsCount {
			keepRoots[candidate.view.RootFull.ID] = true
		}
	}

	completedFulls, err := c.fullRepo.FindCompletedNewestFirstByDatabase(backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	for fullID := range c.getRetainedTiersByFullBackupID(backupConfig, completedFulls) {
		keepRoots[fullID] = true
	}

	deletedCount, keptCount, withinGraceCount := 0, 0, 0

	for _, candidate := range candidates {
		rootFullBackupID := candidate.view.RootFull.ID

		if keepRoots[rootFullBackupID] {
			keptCount++

			logger.DebugContext(ctx, "keeping chain, either policy retains it",
				"root_full_backup_id", rootFullBackupID)

			continue
		}

		if isChainWithinGrace(backupConfig, candidate.endTs) {
			withinGraceCount++

			logger.DebugContext(ctx, "keeping chain, it is still within the retention grace period",
				"root_full_backup_id", rootFullBackupID)

			continue
		}

		if c.deleteChainThroughService(ctx, logger, backupConfig, rootFullBackupID) {
			deletedCount++
		}
	}

	summary := fmt.Sprintf(
		"combined retention: %d chains, %d kept by either policy, %d deleted, %d within grace",
		len(candidates), keptCount, deletedCount, withinGraceCount)

	if deletedCount > 0 {
		logger.InfoContext(ctx, summary)
	} else {
		logger.DebugContext(ctx, summary)
	}

	return nil
}

// cleanOrphans removes WAL that no longer belongs to any chain and reaps
// abandoned insert-first claims, per enabled database.
func (c *PhysicalBackupCleaner) cleanOrphans(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := c.getBackups().ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledConfigs {
		dbLog := logger.With("database_id", backupConfig.DatabaseID)

		c.cleanOrphanWalForDatabase(ctx, dbLog, backupConfig.DatabaseID)
		c.reapAbandonedWalClaims(ctx, dbLog, backupConfig.DatabaseID)
	}

	return nil
}

func (c *PhysicalBackupCleaner) cleanOrphanWalForDatabase(
	ctx context.Context,
	logger *slog.Logger,
	databaseID uuid.UUID,
) {
	orphans, err := c.chainViewService.FindWalOrphansByDatabase(databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find orphan wal", "error", err)

		return
	}

	budget := c.walDeleteBudgetMB(ctx, databaseID)

	for _, orphan := range orphans {
		segment := orphan.WalSegment

		// A one-segment span [start, end) uniquely matches this orphan — segments
		// never overlap, so no chain-covered WAL is ever caught.
		span := chain_view.LSNRange{Start: segment.StartLSN, End: segment.EndLSN}

		rows, mb, err := c.physicalBackupService.DeleteOrphanWalSegmentsInSpan(
			ctx,
			physical_service.DeleteOrphanWalSegmentsSpec{
				DatabaseID:      databaseID,
				TimelineID:      segment.TimelineID,
				Span:            span,
				WalByteBudgetMB: budget,
			},
		)
		if err != nil {
			logger.ErrorContext(ctx, "failed to delete orphan wal segment",
				"wal_filename", segment.WalFilename,
				"error", err,
			)

			continue
		}

		if rows > 0 {
			logger.InfoContext(ctx, fmt.Sprintf("deleted orphan wal segment (%.2f MB)", mb),
				"wal_filename", segment.WalFilename,
			)
		}
	}
}

func (c *PhysicalBackupCleaner) reapAbandonedWalClaims(
	ctx context.Context,
	logger *slog.Logger,
	databaseID uuid.UUID,
) {
	cutoff := time.Now().UTC().Add(-walClaimGracePeriod)

	deletedClaimCount, err := c.walSegmentRepo.DeleteAbandonedClaims(databaseID, cutoff)
	if err != nil {
		logger.ErrorContext(ctx, "failed to reap abandoned wal claims", "error", err)

		return
	}

	if deletedClaimCount > 0 {
		logger.InfoContext(ctx, fmt.Sprintf("reaped %d abandoned wal claims", deletedClaimCount))
	}
}

// approxIntervalDuration maps a schedule to an approximate period for grace
// math. Cron and unset schedules are treated as daily.
// approxScheduleDuration falls back to a day for a missing or cron schedule, whose cadence it cannot
// tell.
func approxScheduleDuration(cadence *schedule.Schedule) time.Duration {
	if cadence == nil {
		return 24 * time.Hour
	}

	if approxPeriod := cadence.GetApproxPeriod(); approxPeriod > 0 {
		return approxPeriod
	}

	return 24 * time.Hour
}

// The LAST_N variant keeps by position rather than by tier, so it records no tiers at all.
func formatRetentionTiersSuffix(tiers []gfs.Tier) string {
	if len(tiers) == 0 {
		return ""
	}

	return " (retained as " + gfs.FormatTiers(tiers) + ")"
}
