package verifications

import "github.com/google/uuid"

type VerificationAssignment struct {
	VerificationID     uuid.UUID
	BackupID           uuid.UUID
	BackupSizeMb       float64
	MaxContainerDiskMb float64
	ParallelJobs       int
	Database           AssignedDatabase
	TimescaledbVersion string
}
