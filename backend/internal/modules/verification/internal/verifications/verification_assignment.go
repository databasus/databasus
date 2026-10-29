package verifications

import "github.com/google/uuid"

type VerificationAssignment struct {
	VerificationID     uuid.UUID
	BackupID           uuid.UUID
	BackupSizeMb       float64
	MaxContainerDiskMb float64
	Database           AssignedDatabase
	TimescaledbVersion string
}
