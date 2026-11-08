package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	apiclient "databasus-backend/internal/lib/apiclient"
)

func checkBackendHealth(ctx context.Context, client *apiClient, logger *slog.Logger) error {
	healthClient, err := apiclient.NewClientWithResponses(client.baseURL+"/api/v1",
		apiclient.WithHTTPClient(client.httpClient))
	if err != nil {
		return fmt.Errorf("create backend health client: %w", err)
	}

	response, err := healthClient.GetInstanceHealthWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("backend at %s does not answer, start the app with `mise run run`: %w", client.baseURL, err)
	}
	if response.JSON200 != nil {
		return nil
	}

	if report := response.ApplicationproblemJSON503; report != nil {
		if hasOnlyStaleAgents(report) {
			for _, check := range report.Checks {
				if check.Name == apiclient.HealthCheckNameVERIFICATIONAGENTS {
					logger.WarnContext(ctx, "seeding with stale verification agents",
						slog.Int("stale_agent_count", *check.StaleAgentCount))
				}
			}
			return nil
		}
		return fmt.Errorf("backend at %s reports itself unhealthy: %s", client.baseURL, describeHealthFailure(report))
	}

	return fmt.Errorf("backend at %s reports itself unhealthy: %s", client.baseURL,
		readRefusalReason(response.StatusCode(), response.Body))
}

func hasOnlyStaleAgents(report *apiclient.HealthProblem) bool {
	if report.Code != apiclient.ProblemCodeInstanceUnhealthy || report.Jobs == nil {
		return false
	}

	checksByName := make(map[apiclient.HealthCheckName]apiclient.HealthCheck, len(report.Checks))
	for _, check := range report.Checks {
		checksByName[check.Name] = check
	}
	if len(report.Checks) != 4 || len(checksByName) != 4 {
		return false
	}
	for _, name := range []apiclient.HealthCheckName{
		apiclient.HealthCheckNameDISK, apiclient.HealthCheckNameINTERNALDATABASE, apiclient.HealthCheckNameCLIENTTOOLS,
	} {
		if checksByName[name].Outcome != apiclient.HealthCheckOutcomePASSED {
			return false
		}
	}

	agents := checksByName[apiclient.HealthCheckNameVERIFICATIONAGENTS]
	if agents.Outcome != apiclient.HealthCheckOutcomeFAILED || agents.FailureCode == nil ||
		*agents.FailureCode != apiclient.HealthCheckFailureCodeAGENTSSTALE ||
		agents.StaleAgentCount == nil || *agents.StaleAgentCount <= 0 {
		return false
	}
	for _, job := range report.Jobs {
		if job.IsStale {
			return false
		}
	}
	return true
}

func describeHealthFailure(report *apiclient.HealthProblem) string {
	reasons := []string{report.Detail}
	for _, check := range report.Checks {
		if check.Outcome != apiclient.HealthCheckOutcomeFAILED {
			continue
		}
		reason := string(check.Name)
		if check.FailureCode != nil {
			reason += ": " + string(*check.FailureCode)
		}
		reasons = append(reasons, reason)
	}
	for _, job := range report.Jobs {
		if job.IsStale {
			reasons = append(reasons, "stale background job: "+job.Name)
		}
	}
	return strings.Join(reasons, "; ")
}
