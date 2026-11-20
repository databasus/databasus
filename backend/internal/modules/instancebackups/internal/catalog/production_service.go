package catalog

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/logging"
	"databasus-backend/internal/modules/instancebackups/internal/configs"
	"databasus-backend/internal/modules/instancebackups/internal/snapshot"
)

const (
	ProductionJobName      = "instancebackups_backup_production"
	ProductionJobInterval  = time.Minute
	maxPersistedErrorRunes = 1000
	// spoolLifetime bounds how long undelivered copies retry from one file; afterwards the next dump
	// gives fresher data.
	spoolLifetime = 24 * time.Hour
)

// RunProductionPass dumps a backup root admin started, or starts and dumps one when a destination
// was added or the schedule is due. Copies wait for the delivery job.
func (s *Service) RunProductionPass(ctx context.Context, now time.Time) error {
	backup, isRunning, err := s.backups.getRunning(ctx)
	if err != nil {
		return err
	}

	if !isRunning {
		trigger, isDue, err := s.takeDueTrigger(ctx, now)
		if err != nil || !isDue {
			return err
		}

		backup = s.newRunningBackup(trigger)

		isStarted, err := s.backups.insertRunning(ctx, backup)
		if err != nil || !isStarted {
			return err
		}
	}

	return s.produce(ctx, backup)
}

// takeDueTrigger reports why a backup is due now. A due scheduled occurrence is recorded as owed by
// the destinations, so health measures lateness from it even when the dump fails.
func (s *Service) takeDueTrigger(ctx context.Context, now time.Time) (Trigger, bool, error) {
	destinations, err := s.configs.GetDestinations(ctx)
	if err != nil {
		return "", false, err
	}

	isRequested, err := s.configs.TakeRunRequest(ctx)
	if err != nil || len(destinations) == 0 {
		return "", false, err
	}

	if isRequested {
		return getRequestedTrigger(destinations), true, nil
	}

	config, err := s.configs.GetConfig(ctx)
	if err != nil {
		return "", false, err
	}

	latestStartedAt, err := s.backups.getLatestStartedAt(ctx)
	if err != nil {
		return "", false, err
	}

	if !config.GetSchedule().IsDue(now, latestStartedAt) {
		return "", false, nil
	}

	if err := s.configs.RecordOutstanding(ctx, now); err != nil {
		return "", false, err
	}

	return TriggerSchedule, true, nil
}

func (s *Service) produce(ctx context.Context, backup Backup) error {
	ctx = logging.With(ctx, slog.String("instance_backup_id", backup.ID.String()))

	spooled, err := s.producer.Produce(ctx, backup.ID)
	if err != nil {
		if ctx.Err() != nil {
			return err
		}

		return s.failDump(ctx, backup, err)
	}

	isPlanned, err := s.planCopies(ctx, backup, spooled)
	if err != nil || !isPlanned {
		// Without planned copies nothing records or delivers the file, and the next pass dumps the
		// still running backup again into the same spool path.
		removeSpoolFile(ctx, s.logger, spooled.Path)

		return err
	}

	s.logger.InfoContext(ctx, "databasus backup dumped", slog.Int64("size_bytes", spooled.SizeBytes))
	wake(s.deliverySignal)

	return nil
}

// planCopies records the dump and one waiting copy for every current destination. It reports false
// when no destination remained, which ends the backup as failed.
func (s *Service) planCopies(ctx context.Context, backup Backup, spooled snapshot.SpooledArchive) (bool, error) {
	isPlanned := false

	err := s.transactor.Transaction(ctx, func(txCtx context.Context) error {
		destinations, err := s.configs.GetDestinations(txCtx)
		if err != nil {
			return err
		}

		if len(destinations) == 0 {
			return s.backups.endDump(txCtx, backup.ID, dumpEnd{
				DumpStatus:    DumpStatusSucceeded,
				Status:        StatusFailed,
				FailureReason: "no destination was selected when the dump finished",
				CompletedAt:   s.clock.Now(),
			})
		}

		summaries, err := s.configs.GetDestinationSummaries(txCtx)
		if err != nil {
			return err
		}

		snapshotAt := spooled.Manifest.SnapshotAt
		spoolExpiresAt := s.clock.Now().Add(spoolLifetime)
		backup.SnapshotAt = &snapshotAt
		backup.SizeBytes = &spooled.SizeBytes
		backup.SpoolPath = &spooled.Path
		backup.SpoolExpiresAt = &spoolExpiresAt

		if err := s.backups.recordDumped(txCtx, backup); err != nil {
			return err
		}

		for _, destination := range destinations {
			if err := s.copies.insert(txCtx, s.newCopy(backup, destination, summaries)); err != nil {
				return err
			}
		}

		isPlanned = true

		return nil
	})

	return isPlanned, err
}

func (s *Service) failDump(ctx context.Context, backup Backup, dumpErr error) error {
	s.logger.ErrorContext(ctx, "databasus backup dump failed", slog.Any("error", dumpErr))

	return s.backups.endDump(ctx, backup.ID, dumpEnd{
		DumpStatus:    DumpStatusFailed,
		Status:        StatusFailed,
		FailureReason: logging.RedactError(dumpErr, maxPersistedErrorRunes),
		CompletedAt:   s.clock.Now(),
	})
}

func (s *Service) newCopy(backup Backup, destination configs.Destination, summaries []configs.DestinationChoice) Copy {
	storageID := destination.StorageID
	now := s.clock.Now()
	storageName, workspaceName := "", ""

	for _, summary := range summaries {
		if summary.StorageID == storageID {
			storageName, workspaceName = summary.StorageName, summary.WorkspaceName
		}
	}

	return Copy{
		ID:                  uuid.New(),
		BackupID:            backup.ID,
		StorageID:           &storageID,
		SelectionGeneration: destination.SelectionGeneration,
		StorageName:         storageName,
		WorkspaceName:       workspaceName,
		Status:              CopyStatusPending,
		AttemptCount:        0,
		NextAttemptAt:       &now,
		FileName:            nil,
		StoredAt:            nil,
		LastError:           nil,
	}
}

// getRequestedTrigger tells a run requested for a new destination from one requested to replace a
// lost spool file.
func getRequestedTrigger(destinations []configs.Destination) Trigger {
	for _, destination := range destinations {
		if destination.LastSuccessfulSnapshotAt == nil {
			return TriggerDestinationAdded
		}
	}

	return TriggerSchedule
}
