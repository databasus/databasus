package backuping

import (
	"context"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type backupPreparation struct {
	TimelineID   int
	ManifestPath string
	Cleanup      func()
}

type backupStrategy struct {
	labelKind       string
	streamOperation string
	prepare         func(context.Context, backupExecutionSpec) (backupPreparation, Result, bool)
	classify        streamErrorClassifier
	recheckFailure  func(context.Context, int, Result) Result
	validate        func(context.Context, Result) Result
}

func newBackupStrategy(spec backupExecutionSpec) backupStrategy {
	switch spec.Backup.Kind {
	case catalog.BackupKindFull:
		return backupStrategy{
			labelKind: "FULL", streamOperation: "pg_basebackup stream",
			prepare: prepareFullBackup, classify: classifyFullStreamError,
			recheckFailure: func(ctx context.Context, timelineID int, result Result) Result {
				return recheckFullStreamFailure(ctx, spec.CommonBackupSpec, timelineID, result)
			},
			validate: func(ctx context.Context, result Result) Result {
				return validateFullBackup(ctx, spec.CommonBackupSpec, result)
			},
		}

	case catalog.BackupKindIncremental:
		return backupStrategy{
			labelKind: "INCR", streamOperation: "pg_basebackup --incremental stream",
			prepare: prepareIncrementalBackup, classify: classifyIncrStreamError,
			recheckFailure: func(ctx context.Context, timelineID int, result Result) Result {
				return recheckIncrementalStreamFailure(ctx, spec.CommonBackupSpec, timelineID, result)
			},
		}
	}

	panic("unknown physical backup kind")
}
