package verifications

import (
	"context"

	"github.com/google/uuid"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/verification/internal/agents"
)

func (s *Service) ClaimVerification(
	ctx context.Context,
	agent *agents.Agent,
	capacity AgentCapacity,
) (*JobAssignment, error) {
	runningBackupIDs, err := s.verificationRepository.ListRunningBackupIDsByAgentID(ctx, agent.ID)
	if err != nil {
		return nil, err
	}
	runningBackups, err := s.backups.GetLogicalBackups(ctx, runningBackupIDs)
	if err != nil {
		return nil, err
	}
	var assignment *JobAssignment
	err = s.transactor.Transaction(ctx, func(ctx context.Context) error {
		pending, err := s.verificationRepository.LockOldestPendingTop100(ctx)
		if err != nil {
			return err
		}
		candidates, err := s.pairWithBackups(ctx, pending)
		if err != nil {
			return err
		}
		databaseIDs := make([]uuid.UUID, len(candidates))
		for index, candidate := range candidates {
			databaseIDs[index] = candidate.Verification.DatabaseID
		}
		foundDatabases, err := s.databases.GetDatabases(ctx, databaseIDs)
		if err != nil {
			return err
		}
		versions, err := s.databases.GetPostgresqlLogicalVersions(ctx, databaseIDs)
		if err != nil {
			return err
		}
		databasesByID := make(map[uuid.UUID]databases.Database, len(foundDatabases))
		for _, database := range foundDatabases {
			databasesByID[database.ID] = database
		}
		for _, candidate := range candidates {
			if !EvaluateDiskAdmission(capacity, runningBackups, candidate.Backup).IsFit {
				continue
			}
			database, isFound := databasesByID[candidate.Verification.DatabaseID]
			version := versions[database.ID]
			if !isFound || version == "" {
				continue
			}
			if err := s.verificationRepository.UpdateClaim(
				ctx,
				ClaimUpdate{VerificationID: candidate.Verification.ID, AgentID: agent.ID, StartedAt: s.clock.Now()},
			); err != nil {
				return err
			}
			assignment = &JobAssignment{
				VerificationID:     candidate.Verification.ID,
				BackupID:           candidate.Backup.ID,
				BackupSizeMb:       candidate.Backup.BackupSizeMb,
				MaxContainerDiskMb: float64(EstimateRequiredForRestoreDiskMb(candidate.Backup)),
				Database: AssignedDatabase{
					Type:              database.Type,
					PostgresqlLogical: &AssignedPostgresql{Version: pgshared.Version(version)},
				},
				TimescaledbVersion: candidate.Backup.TimescaledbVersion,
			}
			break
		}
		return nil
	})
	return assignment, err
}
