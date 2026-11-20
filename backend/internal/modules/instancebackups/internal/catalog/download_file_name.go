package catalog

import "databasus-backend/internal/modules/instancebackups/internal/archive"

// downloadFileNameLayout names a download after its snapshot in UTC. The stored file keeps its
// unique attempt name, which only tells copies apart inside a storage.
const downloadFileNameLayout = "databasus-backup-2006-01-02-15-04-05"

func getDownloadFileName(backup Backup) string {
	takenAt := backup.StartedAt
	if backup.SnapshotAt != nil {
		takenAt = *backup.SnapshotAt
	}

	return takenAt.UTC().Format(downloadFileNameLayout) + archive.FileExtension
}
