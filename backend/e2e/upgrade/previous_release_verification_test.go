package upgrade

import (
	"context"
	"fmt"
	"net/http"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

type previousVerificationAgentCreation struct {
	Agent struct {
		ID string `json:"id"`
	} `json:"agent"`
}

type previousVerification struct {
	ID string `json:"id"`
}

type verificationSource struct {
	databaseID string
	backupID   string
}

func writeVerificationData(
	ctx context.Context,
	api *apiClient,
	source verificationSource,
) (string, []upgradecheck.Item, error) {
	var agent previousVerificationAgentCreation
	if err := api.post(
		ctx,
		"/verification-agents",
		map[string]string{"name": "upgrade verification agent"},
		&agent,
	); err != nil {
		return "", nil, fmt.Errorf("create the verification agent: %w", err)
	}

	config := map[string]any{
		"isAutomaticVerificationEnabled": true,
		"scheduleType":                   "INTERVAL",
		"schedule":                       map[string]any{"type": "WEEKLY", "weekday": 3, "timeOfDay": "03:30"},
		"notificationTriggers":           []string{"VERIFICATION_SUCCESS"},
	}

	if err := api.send(ctx, http.MethodPut, "/verification-configs/"+source.databaseID, config, nil); err != nil {
		return "", nil, fmt.Errorf("save the verification config: %w", err)
	}

	var verification previousVerification
	if err := api.post(
		ctx,
		"/verifications",
		map[string]string{"backupId": source.backupID},
		&verification,
	); err != nil {
		return "", nil, fmt.Errorf("enqueue the manual verification: %w", err)
	}
	return verification.ID, []upgradecheck.Item{
		{
			Kind:   "verification agent",
			Name:   agent.Agent.ID,
			Values: map[string]string{"name": "upgrade verification agent"},
		},
		{Kind: "verification config", Name: source.databaseID, Values: map[string]string{
			"isScheduledVerificationEnabled": "true", "scheduleType": "INTERVAL", "interval.type": "WEEKLY",
			"interval.weekday": "3", "interval.timeOfDay": "03:30", "sendNotificationsOn": "VERIFICATION_SUCCESS",
		}},
		{Kind: "verification", Name: verification.ID, Values: map[string]string{
			"databaseId": source.databaseID, "backupId": source.backupID, "trigger": "MANUAL", "status": "PENDING",
		}},
	}, nil
}
