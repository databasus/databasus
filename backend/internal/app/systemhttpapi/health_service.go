package systemhttpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func (s *Service) GetHealth(ctx context.Context) (GetInstanceHealthResponseObject, error) {
	now := s.clock.Now().UTC()
	jobs := s.getJobs(now)
	checks := []HealthCheck{s.checkDisk(ctx), s.checkDatabase(ctx), s.checkTools(ctx), s.checkAgents(ctx)}

	isHealthy := true
	for _, check := range checks {
		if check.Outcome == HealthCheckOutcomeFAILED {
			isHealthy = false
		}
	}
	for _, job := range jobs {
		if job.IsStale {
			isHealthy = false
			s.logger.WarnContext(ctx, "background job is stale",
				slog.String("job_name", job.Name),
				slog.Bool("is_running", job.IsRunning),
				slog.Float64("interval_seconds", job.IntervalSeconds),
				slog.Float64("max_pass_duration_seconds", job.MaxPassDurationSeconds),
				slog.Time("activated_at", job.ActivatedAt),
				slog.Any("last_started_at", job.LastStartedAt),
				slog.Any("last_completed_at", job.LastCompletedAt),
			)
		}
	}

	if isHealthy {
		return healthResponse{
			response: GetInstanceHealth200JSONResponse{Status: HealthReportStatusHEALTHY, Checks: checks, Jobs: jobs},
		}, nil
	}
	return healthResponse{response: GetInstanceHealth503ApplicationProblemPlusJSONResponse{
		Type:   "urn:databasus:problem:instance_unhealthy",
		Title:  "Instance unhealthy",
		Status: http.StatusServiceUnavailable,
		Detail: "one or more instance monitoring checks failed",
		Code:   ProblemCodeInstanceUnhealthy,
		Errors: nil,
		Checks: checks,
		Jobs:   jobs,
	}}, nil
}

func (s *Service) checkDisk(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:            HealthCheckNameDISK,
		Outcome:         HealthCheckOutcomePASSED,
		FailureCode:     nil,
		StaleAgentCount: nil,
	}

	usage, err := s.inspectDisk(s.dataDirectory)
	if err != nil {
		s.logCheckFailure(ctx, check.Name, err)
		return failCheck(check, HealthCheckFailureCodeINSPECTIONFAILED)
	}
	if usage.TotalBytes <= 0 || float64(usage.UsedBytes) >= float64(usage.TotalBytes)*0.95 {
		s.logger.WarnContext(ctx, "instance disk usage exceeds health limit",
			slog.Int64("total_bytes", usage.TotalBytes),
			slog.Int64("used_bytes", usage.UsedBytes),
			slog.Int64("free_bytes", usage.FreeBytes),
		)
		return failCheck(check, HealthCheckFailureCodeDISKFULL)
	}
	return check
}

func (s *Service) checkDatabase(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:            HealthCheckNameINTERNALDATABASE,
		Outcome:         HealthCheckOutcomePASSED,
		FailureCode:     nil,
		StaleAgentCount: nil,
	}

	checkContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.connectivity.check(checkContext); err != nil {
		s.logCheckFailure(ctx, check.Name, err)
		return failCheck(check, HealthCheckFailureCodeCONNECTIONFAILED)
	}
	return check
}

func (s *Service) checkTools(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:            HealthCheckNameCLIENTTOOLS,
		Outcome:         HealthCheckOutcomePASSED,
		FailureCode:     nil,
		StaleAgentCount: nil,
	}
	if err := s.clientTools.GetHealthError(ctx); err != nil {
		s.logCheckFailure(ctx, check.Name, err)
		return failCheck(check, HealthCheckFailureCodeTOOLSUNUSABLE)
	}
	return check
}

func (s *Service) checkAgents(ctx context.Context) HealthCheck {
	check := HealthCheck{
		Name:            HealthCheckNameVERIFICATIONAGENTS,
		Outcome:         HealthCheckOutcomePASSED,
		FailureCode:     nil,
		StaleAgentCount: nil,
	}

	checkContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	staleAgents, err := s.verification.ListStaleAgents(checkContext)
	if err != nil {
		s.logCheckFailure(ctx, check.Name, err)
		return failCheck(check, HealthCheckFailureCodeQUERYFAILED)
	}

	check.StaleAgentCount = new(len(staleAgents))
	if len(staleAgents) > 0 {
		for _, agent := range staleAgents {
			s.logger.WarnContext(ctx, "verification agent is stale",
				slog.String("agent_id", agent.ID.String()),
				slog.Time("created_at", agent.CreatedAt),
				slog.Any("last_seen_at", agent.LastSeenAt),
			)
		}
		return failCheck(check, HealthCheckFailureCodeAGENTSSTALE)
	}
	return check
}

func failCheck(check HealthCheck, failureCode HealthCheckFailureCode) HealthCheck {
	check.Outcome = HealthCheckOutcomeFAILED
	check.FailureCode = &failureCode
	return check
}

func (s *Service) logCheckFailure(ctx context.Context, check HealthCheckName, err error) {
	s.logger.WarnContext(
		ctx,
		"instance dependency check failed",
		slog.String("check", string(check)),
		slog.Any("error", err),
	)
}
