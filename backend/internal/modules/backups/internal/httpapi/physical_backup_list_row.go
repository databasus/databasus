package httpapi

import (
	"databasus-backend/internal/modules/backups/internal/physical"
	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

func toPhysicalBackupListRow(backup physical.PhysicalBackupListItem) (PhysicalBackupListRow, error) {
	body := PhysicalBackupListRow{}
	body.CompletedAt = backup.CompletedAt
	body.CreatedAt = backup.CreatedAt
	body.FailMessage = backup.FailMessage
	body.FromWalFilename = backup.FromWalFilename
	body.HasGapBefore = backup.HasGapBefore
	body.Id = backup.ID
	body.ParentIncrementalBackupId = backup.ParentIncrementalBackupID
	body.RootFullBackupId = backup.RootFullBackupID
	body.SizeMb = backup.SizeMb
	body.StartLsn = backup.StartLSN

	switch backup.Status {
	case "IN_PROGRESS", "COMPLETED", "ERROR", "CHAIN_BROKEN", "CANCELED":
	default:
		return PhysicalBackupListRow{}, errUnknownPhysicalValue
	}

	body.Status = PhysicalBackupStatus(backup.Status)
	body.StopLsn = backup.StopLSN
	body.TimelineId = backup.TimelineID
	body.ToWalFilename = backup.ToWalFilename

	switch backup.Type {
	case catalog.BackupTypeFull, catalog.BackupTypeIncremental, catalog.BackupTypeWal:
	default:
		return PhysicalBackupListRow{}, errUnknownPhysicalValue
	}

	body.Type = PhysicalBackupListRowType(backup.Type)
	body.WalSegmentsCount = backup.WalSegmentsCount

	return body, nil
}
