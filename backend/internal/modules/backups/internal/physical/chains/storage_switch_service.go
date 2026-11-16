package chains

import (
	"context"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/config"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
	"databasus-backend/internal/modules/storages"
)

type (
	storageSwitchFilesKey struct{}
	storageSwitchFiles    struct {
		references []storages.StoredFileReference
	}
)

func (s *Service) RemoveBackupsForStorageChange(
	ctx context.Context,
	removal config.HistoryRemoval,
) (config.RemovedBackupHistory, error) {
	hasHistory, err := s.repository.HasDatabaseHistory(ctx, removal.DatabaseID)
	if err != nil {
		return config.RemovedBackupHistory{}, err
	}

	if removal.ShouldResetAll {
		references, err := s.repository.GetDatabaseFileReferences(ctx, removal.DatabaseID)
		if err != nil {
			return config.RemovedBackupHistory{}, err
		}
		if err := s.repository.DeleteDatabaseRows(ctx, removal.DatabaseID); err != nil {
			return config.RemovedBackupHistory{}, err
		}
		return config.RemovedBackupHistory{HasHistory: hasHistory, References: references}, nil
	}

	captured := &storageSwitchFiles{}
	ctx = context.WithValue(ctx, storageSwitchFilesKey{}, captured)

	fulls, err := s.backupRepository.FindAllFullByDatabaseOutsideStorage(
		ctx,
		removal.DatabaseID,
		removal.TargetStorageID,
	)
	if err != nil {
		return config.RemovedBackupHistory{}, err
	}

	for _, full := range fulls {
		if _, err := s.DeleteFullCompletely(ctx, full.ID); err != nil {
			return config.RemovedBackupHistory{}, err
		}
	}

	remainingFulls, err := s.backupRepository.FindCompletedFullNewestFirstByDatabase(ctx, removal.DatabaseID)
	if err != nil {
		return config.RemovedBackupHistory{}, err
	}

	if len(remainingFulls) == 0 {
		references, err := s.repository.GetDatabaseFileReferences(ctx, removal.DatabaseID)
		if err != nil {
			return config.RemovedBackupHistory{}, err
		}
		captured.references = append(captured.references, references...)
		if err := s.repository.DeleteDatabaseRows(ctx, removal.DatabaseID); err != nil {
			return config.RemovedBackupHistory{}, err
		}
	} else if err := s.removeUnneededRecoveryArtifacts(ctx, removal.DatabaseID); err != nil {
		return config.RemovedBackupHistory{}, err
	}

	return config.RemovedBackupHistory{
		HasHistory: hasHistory,
		References: captured.references,
	}, nil
}

// removeUnneededRecoveryArtifacts drops the WAL and timeline histories that no line of a surviving
// completed FULL holds. WAL of a promoted timeline whose history is still pending goes too: the new
// storage receives its history, and a line never crosses into another storage.
func (s *Service) removeUnneededRecoveryArtifacts(ctx context.Context, databaseID uuid.UUID) error {
	chainCatalog, err := s.loadChainCatalog(ctx, databaseID)
	if err != nil {
		return err
	}

	ownership := newWalOwnership(chainCatalog, uuid.Nil)

	candidates, err := s.walSegmentRepository.FindOrphanCandidates(ctx, databaseID)
	if err != nil {
		return err
	}

	for _, segment := range candidates {
		segmentSpan := catalog.LSNRange{Start: segment.StartLSN, End: segment.EndLSN}
		if ownership.isTimelineOnAnyLine(segment.TimelineID, segmentSpan) {
			continue
		}

		if err := s.deleteWalSegment(ctx, segment); err != nil {
			return err
		}
	}

	for _, historyFile := range chainCatalog.Histories {
		if ownership.hasTimelineOnAnyLine(historyFile.TimelineID) {
			continue
		}

		if err := s.requestFileDeletions(
			ctx,
			historyFile.StorageID,
			historyFile.FileName,
			historyFile.FileName+metadataSuffix,
		); err != nil {
			return err
		}

		if err := s.walHistoryRepository.DeleteByID(ctx, historyFile.ID); err != nil {
			return err
		}
	}

	return nil
}
