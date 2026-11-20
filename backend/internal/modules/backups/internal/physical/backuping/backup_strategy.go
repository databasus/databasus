package backuping

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type backupPreparation struct {
	// TimelineID is the live timeline at preflight.
	TimelineID int
	// IsAcrossPromotion marks an INCREMENTAL taken on a newer timeline than the chain's newest backup.
	IsAcrossPromotion bool
	ManifestPath      string
	Cleanup           func()
}

type backupStrategy struct {
	labelKind       string
	streamOperation string
	prepare         func(context.Context, backupExecutionSpec) (backupPreparation, Result, bool)
	classify        func(backupPreparation) streamErrorClassifier
	recheckFailure  func(context.Context, backupPreparation, Result) Result
	validate        func(context.Context, Result) Result
}

func newBackupStrategy(spec backupExecutionSpec) backupStrategy {
	switch spec.Backup.Kind {
	case catalog.BackupKindFull:
		return backupStrategy{
			labelKind: "FULL", streamOperation: "pg_basebackup stream",
			prepare: prepareFullBackup,
			classify: func(backupPreparation) streamErrorClassifier {
				return classifyFullStreamError
			},
			recheckFailure: func(ctx context.Context, preparation backupPreparation, result Result) Result {
				return recheckStreamFailure(ctx, spec.CommonBackupSpec, preparation.TimelineID, result)
			},
			validate: func(ctx context.Context, result Result) Result {
				return validateFullBackup(ctx, spec.CommonBackupSpec, result)
			},
		}

	case catalog.BackupKindIncremental:
		return backupStrategy{
			labelKind: "INCR", streamOperation: "pg_basebackup --incremental stream",
			prepare: prepareIncrementalBackup,
			classify: func(preparation backupPreparation) streamErrorClassifier {
				return func(streamErr error, stderr []byte) streamOutcome {
					return classifyIncrStreamError(streamErr, stderr, preparation.IsAcrossPromotion)
				}
			},
			recheckFailure: func(ctx context.Context, preparation backupPreparation, result Result) Result {
				return recheckStreamFailure(ctx, spec.CommonBackupSpec, preparation.TimelineID, result)
			},
		}
	}

	panic("unknown physical backup kind")
}
