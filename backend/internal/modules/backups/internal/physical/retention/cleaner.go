package retention

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/lib/clock"
	"databasus-backend/internal/lib/gfs"
	"databasus-backend/internal/lib/schedule"
	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/backups/internal/physical/chains"
	"databasus-backend/internal/modules/backups/internal/physical/streamhealth"
)

// Cleaner trims physical backups every tick through two passes:
// retention policy and orphan sweep. It never issues raw DELETEs — every removal
// goes through chains.Service — and it never touches the active
// (extendable) chain: it operates only on the non-extendable set that chains
// derives.
type Cleaner struct {
	chains              *chains.Service
	configs             *config.Service
	backupRepo          *catalog.BackupRepository
	walSegmentRepo      *catalog.WalSegmentRepository
	walStreamSampleRepo *catalog.WalStreamSampleRepository
	logger              *slog.Logger

	clock clock.Source
}

func (c *Cleaner) GetJob() background.Job {
	return background.Job{
		Name:     cleanerJobName,
		Interval: time.Minute,
		Run:      func(ctx context.Context, _ time.Time) error { return c.RunPass(ctx) },
	}
}

func (c *Cleaner) RunPass(ctx context.Context) error {
	logger := c.logger
	retentionErr := c.cleanByRetentionPolicy(ctx, logger)
	orphanErr := c.cleanOrphans(ctx, logger)
	c.deleteExpiredWalStreamSamples(ctx, logger, c.clock.Now())

	return errors.Join(retentionErr, orphanErr)
}

// Samples also expire for databases that have stopped streaming.
func (c *Cleaner) deleteExpiredWalStreamSamples(ctx context.Context, logger *slog.Logger, now time.Time) {
	deletedCount, err := c.walStreamSampleRepo.DeleteOlderThan(ctx, now.Add(-streamhealth.SampleRetention))
	if err != nil {
		logger.ErrorContext(ctx, "failed to delete expired wal stream samples", slog.Any("error", err))

		return
	}

	if deletedCount > 0 {
		logger.DebugContext(ctx, "deleted expired wal stream samples", slog.Any("deleted_count", deletedCount))
	}
}

func (c *Cleaner) cleanByRetentionPolicy(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := c.configs.ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledConfigs {
		dbLog := logger.With(
			slog.Any("database_id", backupConfig.DatabaseID),
			slog.Any("retention", backupConfig.Retention.Policy),
		)

		var cleanErr error

		switch backupConfig.Retention.Policy {
		case config.PhysicalRetentionPolicyChains:
			cleanErr = c.cleanByChains(ctx, dbLog, backupConfig)
		case config.PhysicalRetentionPolicyFullBackups:
			cleanErr = c.cleanByFulls(ctx, dbLog, backupConfig)
		case config.PhysicalRetentionPolicyChainsAndFullBackups:
			cleanErr = c.cleanByCombined(ctx, dbLog, backupConfig)
		}

		if cleanErr != nil {
			dbLog.ErrorContext(ctx, "failed to clean database by retention policy", slog.Any("error", cleanErr))
		}
	}

	return nil
}

// nonExtendableChainsNewestEndFirst returns every non-extendable chain for a DB
// sorted by chain-end timestamp, newest first. The active (extendable) head is
// already excluded by FindNonExtendableChainsByDatabase, so nothing here can
// ever delete it.
func (c *Cleaner) nonExtendableChainsNewestEndFirst(
	ctx context.Context,
	databaseID uuid.UUID,
) ([]chainCandidate, error) {
	views, err := c.chains.FindNonExtendableChainsByDatabase(ctx, databaseID)
	if err != nil {
		return nil, err
	}

	candidates := make([]chainCandidate, 0, len(views))

	for _, view := range views {
		endAt, err := c.chains.GetChainEndTimestamp(ctx, view)
		if err != nil {
			c.logger.ErrorContext(
				ctx,
				"failed to get chain end timestamp",
				slog.Any("root_full_backup_id", view.RootFull.ID),
				slog.Any("error", err),
			)

			continue
		}

		candidates = append(candidates, chainCandidate{view: view, endAt: endAt})
	}

	slices.SortFunc(candidates, func(a, b chainCandidate) int {
		return b.endAt.Compare(a.endAt)
	})

	return candidates, nil
}

// Reports whether the delete succeeded, so callers count outcomes rather than attempts.
func (c *Cleaner) deleteChainThroughService(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
	rootFullBackupID uuid.UUID,
) bool {
	summary, err := c.chains.DeleteFull(
		ctx, rootFullBackupID, c.walDeleteBudgetMB(ctx, backupConfig.DatabaseID),
	)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to delete chain",
			slog.Any("root_full_backup_id", rootFullBackupID),
			slog.Any("error", err),
		)

		return false
	}

	logger.InfoContext(
		ctx,
		"chain cleanup progressed",
		slog.Any("root_full_backup_id", rootFullBackupID),
		slog.Any("wal_segments", summary.WalSegments),
		slog.Any("incrementals", summary.Incrementals),
		slog.Any("history_files", summary.HistoryFiles),
		slog.Any("size_mb", summary.BytesDeletedMB),
		slog.Any("is_complete", summary.ChainFullyDeleted),
	)

	return true
}

// A kept FULL stays a standalone restore point, so its INCRs and WAL go while it remains.
// Reports whether the delete succeeded, so callers count outcomes rather than attempts.
func (c *Cleaner) deleteChainDependentsThroughService(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
	rootFullBackupID uuid.UUID,
) bool {
	summary, err := c.chains.DeleteChainDependentsKeepFull(
		ctx, rootFullBackupID, c.walDeleteBudgetMB(ctx, backupConfig.DatabaseID),
	)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"failed to delete chain dependents",
			slog.Any("root_full_backup_id", rootFullBackupID),
			slog.Any("error", err),
		)

		return false
	}

	logger.InfoContext(
		ctx,
		"chain dependents cleaned",
		slog.Any("root_full_backup_id", rootFullBackupID),
		slog.Any("wal_segments", summary.WalSegments),
		slog.Any("incrementals", summary.Incrementals),
		slog.Any("size_mb", summary.BytesDeletedMB),
	)

	return true
}

// walDeleteBudgetMB anchors the per-tick WAL byte budget to the latest COMPLETED
// FULL's size, floored at minWalDeleteBudgetMB.
func (c *Cleaner) walDeleteBudgetMB(ctx context.Context, databaseID uuid.UUID) float64 {
	fulls, err := c.backupRepo.FindCompletedFullNewestFirstByDatabase(ctx, databaseID)
	if err != nil {
		// Falling back to the floor silently would look like retention crawling for no reason.
		c.logger.WarnContext(
			ctx,
			"failed to size wal delete budget",
			slog.Any("floor_mb", minWalDeleteBudgetMB),
			slog.Any("database_id", databaseID),
			slog.Any("error", err),
		)

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
	now time.Time,
	backupConfig config.PhysicalBackupConfig,
	chainEndTs time.Time,
) bool {
	grace := max(
		approxScheduleDuration(&backupConfig.FullBackupSchedule),
		approxScheduleDuration(backupConfig.IncrementalBackupSchedule),
	) * chainGraceIntervalMultiplier

	grace = max(grace, recentBackupGracePeriod)

	return now.Sub(chainEndTs) < grace
}

// cleanByChains keeps the N newest non-extendable chains (by chain-end
// timestamp) and deletes the rest, honoring the per-chain grace period. The
// active chain is never a candidate.
func (c *Cleaner) cleanByChains(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
) error {
	keepCount := backupConfig.Retention.ChainsCount
	if keepCount <= 0 {
		logger.DebugContext(ctx, "chain retention has no count configured, keeping every chain")

		return nil
	}

	candidates, err := c.nonExtendableChainsNewestEndFirst(ctx, backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	if len(candidates) <= keepCount {
		logger.DebugContext(
			ctx,
			"retaining all chains",
			slog.Any("chain_count", len(candidates)),
			slog.Any("keep_count", keepCount),
		)

		return nil
	}

	deletedCount, withinGraceCount := 0, 0

	for _, candidate := range candidates[keepCount:] {
		rootFullBackupID := candidate.view.RootFull.ID

		if isChainWithinGrace(c.clock.Now(), backupConfig, candidate.endAt) {
			withinGraceCount++

			logger.DebugContext(
				ctx,
				"keeping chain, it is still within the retention grace period",
				slog.Any("root_full_backup_id", rootFullBackupID),
			)

			continue
		}

		if c.deleteChainThroughService(ctx, logger, backupConfig, rootFullBackupID) {
			deletedCount++
		}
	}

	attributes := []any{
		slog.Any("chain_count", len(candidates)),
		slog.Any("kept_count", keepCount),
		slog.Any("deleted_count", deletedCount),
		slog.Any("within_grace_count", withinGraceCount),
	}
	if deletedCount > 0 {
		logger.InfoContext(ctx, "chain retention completed", attributes...)
	} else {
		logger.DebugContext(ctx, "chain retention completed", attributes...)
	}

	return nil
}

// cleanByFulls dispatches the FULL_BACKUPS retention policy to its LAST_N or GFS
// variant. Both keep a set of FULLs; non-extendable chains rooted at a kept FULL
// shed their INCR + WAL (the FULL stays a standalone restore point), and chains
// rooted at a non-kept FULL are deleted whole.
func (c *Cleaner) cleanByFulls(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
) error {
	completedFulls, err := c.backupRepo.FindCompletedFullNewestFirstByDatabase(ctx, backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	retainedTiersByFullBackupID := c.getRetainedTiersByFullBackupID(backupConfig, completedFulls)
	if retainedTiersByFullBackupID == nil {
		return nil
	}

	candidates, err := c.nonExtendableChainsNewestEndFirst(ctx, backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	deletedCount, dependentsOnlyCount, withinGraceCount := 0, 0, 0

	for _, candidate := range candidates {
		rootFullBackupID := candidate.view.RootFull.ID

		if isChainWithinGrace(c.clock.Now(), backupConfig, candidate.endAt) {
			withinGraceCount++

			logger.DebugContext(
				ctx,
				"keeping chain, it is still within the retention grace period",
				slog.Any("root_full_backup_id", rootFullBackupID),
			)

			continue
		}

		if tiers, isKept := retainedTiersByFullBackupID[rootFullBackupID]; isKept {
			logger.DebugContext(
				ctx,
				"keeping the full backup and shedding its dependents",
				slog.Any("retention_tiers", tiers),
				slog.Any("root_full_backup_id", rootFullBackupID),
			)

			if c.deleteChainDependentsThroughService(ctx, logger, backupConfig, rootFullBackupID) {
				dependentsOnlyCount++
			}

			continue
		}

		if c.deleteChainThroughService(ctx, logger, backupConfig, rootFullBackupID) {
			deletedCount++
		}
	}

	attributes := []any{
		slog.Any("chain_count", len(candidates)),
		slog.Any("kept_count", 0),
		slog.Any("deleted_count", deletedCount),
		slog.Any("within_grace_count", withinGraceCount),
		slog.Any("dependents_only_count", dependentsOnlyCount),
	}
	if deletedCount+dependentsOnlyCount > 0 {
		logger.InfoContext(ctx, "full backup retention completed", attributes...)
	} else {
		logger.DebugContext(ctx, "full backup retention completed", attributes...)
	}

	return nil
}

// Returns nil when the policy has no effective configuration, so the caller no-ops rather than
// treating an empty result as "delete everything".
func (c *Cleaner) getRetainedTiersByFullBackupID(
	backupConfig config.PhysicalBackupConfig,
	completedFullsNewestFirst []*catalog.Backup,
) map[uuid.UUID][]gfs.Tier {
	retention := backupConfig.Retention.FullBackups

	switch retention.Policy {
	case config.FullBackupsRetentionPolicyLastN:
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

	case config.FullBackupsRetentionPolicyGfs:
		if retention.Gfs.Hours <= 0 && retention.Gfs.Days <= 0 && retention.Gfs.Weeks <= 0 &&
			retention.Gfs.Months <= 0 && retention.Gfs.Years <= 0 {
			return nil
		}

		items := make([]gfs.Item, len(completedFullsNewestFirst))
		for i, full := range completedFullsNewestFirst {
			items[i] = gfs.Item{ID: full.ID, CreatedAt: full.CreatedAt}
		}

		return gfs.GetItemsToRetain(gfs.Selection{
			Items:  items,
			Hours:  retention.Gfs.Hours,
			Days:   retention.Gfs.Days,
			Weeks:  retention.Gfs.Weeks,
			Months: retention.Gfs.Months,
			Years:  retention.Gfs.Years,
		})
	}

	return nil
}

// cleanByCombined keeps the UNION of the CHAINS keep-set (N newest
// non-extendable chains) and the FULL_BACKUPS keep-set (LAST_N or GFS over
// FULLs), deleting every other non-extendable chain whole. A chain in either
// keep-set is preserved entirely (INCR + WAL included).
func (c *Cleaner) cleanByCombined(
	ctx context.Context,
	logger *slog.Logger,
	backupConfig config.PhysicalBackupConfig,
) error {
	candidates, err := c.nonExtendableChainsNewestEndFirst(ctx, backupConfig.DatabaseID)
	if err != nil {
		return err
	}

	keepRoots := make(map[uuid.UUID]bool)

	for i, candidate := range candidates {
		if i < backupConfig.Retention.ChainsCount {
			keepRoots[candidate.view.RootFull.ID] = true
		}
	}

	completedFulls, err := c.backupRepo.FindCompletedFullNewestFirstByDatabase(ctx, backupConfig.DatabaseID)
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

			logger.DebugContext(
				ctx,
				"keeping chain, either policy retains it",
				slog.Any("root_full_backup_id", rootFullBackupID),
			)

			continue
		}

		if isChainWithinGrace(c.clock.Now(), backupConfig, candidate.endAt) {
			withinGraceCount++

			logger.DebugContext(
				ctx,
				"keeping chain, it is still within the retention grace period",
				slog.Any("root_full_backup_id", rootFullBackupID),
			)

			continue
		}

		if c.deleteChainThroughService(ctx, logger, backupConfig, rootFullBackupID) {
			deletedCount++
		}
	}

	attributes := []any{
		slog.Any("chain_count", len(candidates)),
		slog.Any("kept_count", keptCount),
		slog.Any("deleted_count", deletedCount),
		slog.Any("within_grace_count", withinGraceCount),
	}
	if deletedCount > 0 {
		logger.InfoContext(ctx, "combined retention completed", attributes...)
	} else {
		logger.DebugContext(ctx, "combined retention completed", attributes...)
	}

	return nil
}

// cleanOrphans removes WAL that no longer belongs to any chain and reaps
// abandoned insert-first claims, per enabled database.
func (c *Cleaner) cleanOrphans(ctx context.Context, logger *slog.Logger) error {
	enabledConfigs, err := c.configs.ListEnabledPhysicalBackupConfigs(ctx)
	if err != nil {
		return err
	}

	for _, backupConfig := range enabledConfigs {
		dbLog := logger.With(slog.Any("database_id", backupConfig.DatabaseID))

		c.cleanOrphanWalForDatabase(ctx, dbLog, backupConfig.DatabaseID)
		c.reapAbandonedWalClaims(ctx, dbLog, backupConfig.DatabaseID)
	}

	return nil
}

func (c *Cleaner) cleanOrphanWalForDatabase(
	ctx context.Context,
	logger *slog.Logger,
	databaseID uuid.UUID,
) {
	orphans, err := c.chains.FindWalOrphansByDatabase(ctx, databaseID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to find orphan wal", slog.Any("error", err))

		return
	}

	budget := c.walDeleteBudgetMB(ctx, databaseID)

	for _, orphan := range orphans {
		segment := orphan.WalSegment

		// A one-segment span [start, end) uniquely matches this orphan — segments
		// never overlap, so no chain-covered WAL is ever caught.
		span := catalog.LSNRange{Start: segment.StartLSN, End: segment.EndLSN}

		rows, mb, err := c.chains.DeleteOrphanWalSegmentsInSpan(
			ctx,
			chains.DeleteOrphanWalSegmentsSpec{
				DatabaseID:      databaseID,
				TimelineID:      segment.TimelineID,
				Span:            span,
				WalByteBudgetMB: budget,
			},
		)
		if err != nil {
			logger.ErrorContext(
				ctx,
				"failed to delete orphan wal segment",
				slog.Any("wal_filename", segment.WalFilename),
				slog.Any("error", err),
			)

			continue
		}

		if rows > 0 {
			logger.InfoContext(
				ctx,
				"deleted orphan wal segment",
				slog.Any("size_mb", mb),
				slog.Any("wal_filename", segment.WalFilename),
			)
		}
	}
}

func (c *Cleaner) reapAbandonedWalClaims(
	ctx context.Context,
	logger *slog.Logger,
	databaseID uuid.UUID,
) {
	cutoff := c.clock.Now().Add(-walClaimGracePeriod)

	deletedClaimCount, err := c.walSegmentRepo.DeleteAbandonedClaims(ctx, databaseID, cutoff)
	if err != nil {
		logger.ErrorContext(ctx, "failed to reap abandoned wal claims", slog.Any("error", err))

		return
	}

	if deletedClaimCount > 0 {
		logger.InfoContext(ctx, "reaped abandoned wal claims", slog.Any("deleted_count", deletedClaimCount))
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

type Deps struct {
	Chains           *chains.Service
	Configs          *config.Service
	Backups          *catalog.BackupRepository
	WalSegments      *catalog.WalSegmentRepository
	WalStreamSamples *catalog.WalStreamSampleRepository
	Logger           *slog.Logger
	Clock            clock.Source
}

func New(deps Deps) *Cleaner {
	return &Cleaner{
		chains:              deps.Chains,
		configs:             deps.Configs,
		backupRepo:          deps.Backups,
		walSegmentRepo:      deps.WalSegments,
		walStreamSampleRepo: deps.WalStreamSamples,
		logger:              deps.Logger,
		clock:               deps.Clock,
	}
}

const (
	chainGraceIntervalMultiplier = 2
	walClaimGracePeriod          = time.Hour
	minWalDeleteBudgetMB         = 256
	recentBackupGracePeriod      = time.Hour
	cleanerJobName               = "backups_physical_cleaner"
)

type chainCandidate struct {
	view  *chains.ChainView
	endAt time.Time
}
