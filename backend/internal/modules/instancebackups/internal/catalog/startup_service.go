package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
)

// Start ends what a stopped process left unfinished before the jobs run: a running dump is
// interrupted, an upload returns to the queue while its spool file exists and is interrupted when
// the file was lost, and spool files no backup needs are removed.
func (s *Service) Start(ctx context.Context) error {
	now := s.clock.Now()

	if err := s.backups.interruptRunningDumps(ctx, now); err != nil {
		return err
	}

	spooledBackups, err := s.backups.listSpooled(ctx)
	if err != nil {
		return err
	}

	keptPaths := make([]string, 0, len(spooledBackups))

	for _, backup := range spooledBackups {
		isKept, err := s.resumeDelivery(ctx, backup)
		if err != nil {
			return err
		}

		if isKept {
			keptPaths = append(keptPaths, *backup.SpoolPath)
		}
	}

	return s.removeUnneededSpoolFiles(ctx, keptPaths)
}

func (s *Service) resumeDelivery(ctx context.Context, backup Backup) (bool, error) {
	_, err := os.Stat(*backup.SpoolPath)
	if err == nil {
		return true, s.copies.retryUploading(ctx, backup.ID, s.clock.Now(),
			"Databasus stopped while the copy was uploaded")
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("check the Databasus backup spool file: %w", err)
	}

	s.logger.WarnContext(ctx, "databasus backup spool file was lost while Databasus was stopped",
		slog.String("instance_backup_id", backup.ID.String()))

	err = s.copies.endUnfinished(ctx, backup.ID, CopyStatusInterrupted,
		"the local file of the backup was lost when Databasus stopped")
	if err != nil {
		return false, err
	}

	if err := s.configs.RequestRun(ctx); err != nil {
		return false, err
	}

	return false, s.finishBackupIfEnded(ctx, backup.ID)
}

func (s *Service) removeUnneededSpoolFiles(ctx context.Context, keptPaths []string) error {
	spoolDirectory := s.producer.GetSpoolDirectory()

	entries, err := os.ReadDir(spoolDirectory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("list the Databasus backup spool: %w", err)
	}

	for _, entry := range entries {
		path := filepath.Join(spoolDirectory, entry.Name())
		if !slices.Contains(keptPaths, path) {
			removeSpoolFile(ctx, s.logger, path)
		}
	}

	return nil
}
