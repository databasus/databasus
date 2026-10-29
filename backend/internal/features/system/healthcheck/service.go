package system_healthcheck

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"databasus-backend/internal/features/disk"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/verification"
	"databasus-backend/internal/storage"
)

type HealthcheckService struct {
	diskService *disk.DiskService
	// getBackups is read per call: each test application installs its own module.
	getBackups      func() backups.Module
	getVerification func() verification.Module
	logger          *slog.Logger
}

func (s *HealthcheckService) IsHealthy(ctx context.Context) error {
	return s.performHealthCheck(ctx)
}

// Each branch is a degradation the probe exists to surface. Returning the reason to the caller only
// answers "is it healthy now"; logging it is what leaves a history to read afterwards.
func (s *HealthcheckService) performHealthCheck(ctx context.Context) error {
	diskUsage, err := s.diskService.GetDiskUsage(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "healthcheck degraded: cannot get disk usage", "error", err)

		return errors.New("cannot get disk usage")
	}

	if float64(diskUsage.UsedSpaceBytes) >= float64(diskUsage.TotalSpaceBytes)*0.95 {
		s.logger.WarnContext(ctx, fmt.Sprintf("healthcheck degraded: %.1f%% of the disk is used",
			float64(diskUsage.UsedSpaceBytes)/float64(diskUsage.TotalSpaceBytes)*100))

		return errors.New("more than 95% of the disk is used")
	}

	if err := getClientToolsChecker().GetHealthError(ctx); err != nil {
		return err
	}

	db := storage.GetDb()
	err = db.Raw("SELECT 1").Error
	if err != nil {
		s.logger.WarnContext(ctx, "healthcheck degraded: cannot connect to the database", "error", err)

		return errors.New("cannot connect to the database")
	}

	if !s.getBackups().IsLogicalBackupSchedulerRunning(ctx) {
		s.logger.WarnContext(ctx, "healthcheck degraded: the backup scheduler has not ticked in 5 minutes")

		return errors.New("backups are not running for more than 5 minutes")
	}

	staleAgents, err := s.getVerification().ListStaleAgents(ctx)
	if err != nil {
		s.logger.WarnContext(ctx, "healthcheck degraded: cannot query verification agents", "error", err)

		return errors.New("cannot query verification agents")
	}

	if len(staleAgents) > 0 {
		names := make([]string, len(staleAgents))
		for i, agent := range staleAgents {
			names[i] = agent.Name
		}

		return fmt.Errorf(
			"verification agents not seen for more than 5 minutes: %s",
			strings.Join(names, ", "),
		)
	}

	return nil
}
