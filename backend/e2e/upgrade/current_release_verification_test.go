package upgrade

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

type currentVerificationAgent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type currentVerificationConfig struct {
	IsScheduledVerificationEnabled bool   `json:"isScheduledVerificationEnabled"`
	ScheduleType                   string `json:"scheduleType"`
	VerificationInterval           struct {
		Type      string  `json:"type"`
		Weekday   *int    `json:"weekday"`
		TimeOfDay *string `json:"timeOfDay"`
	} `json:"verificationInterval"`
	SendNotificationsOn []string `json:"sendNotificationsOn"`
}

type currentVerification struct {
	ID         string `json:"id"`
	DatabaseID string `json:"databaseId"`
	BackupID   string `json:"backupId"`
	Trigger    string `json:"trigger"`
	Status     string `json:"status"`
}

type verificationReferences struct {
	databaseID     string
	verificationID string
}

func readVerificationItems(
	ctx context.Context,
	api *apiClient,
	references verificationReferences,
) ([]upgradecheck.Item, error) {
	var agents []currentVerificationAgent
	if err := api.get(ctx, "/verification/agents", &agents); err != nil {
		return nil, fmt.Errorf("list verification agents after upgrade: %w", err)
	}
	items := make([]upgradecheck.Item, 0, len(agents)+2)
	for _, agent := range agents {
		items = append(
			items,
			upgradecheck.Item{
				Kind:   "verification agent",
				Name:   agent.ID,
				Values: map[string]string{"name": agent.Name},
			},
		)
	}
	var config currentVerificationConfig
	if err := api.get(ctx, "/verification-config/"+references.databaseID, &config); err != nil {
		return nil, fmt.Errorf("read verification config after upgrade: %w", err)
	}
	weekday := ""
	if config.VerificationInterval.Weekday != nil {
		weekday = strconv.Itoa(*config.VerificationInterval.Weekday)
	}
	items = append(
		items,
		upgradecheck.Item{Kind: "verification config", Name: references.databaseID, Values: map[string]string{
			"isScheduledVerificationEnabled": strconv.FormatBool(config.IsScheduledVerificationEnabled),
			"scheduleType":                   config.ScheduleType, "interval.type": config.VerificationInterval.Type,
			"interval.weekday": weekday, "interval.timeOfDay": getOptionalValue(config.VerificationInterval.TimeOfDay),
			"sendNotificationsOn": strings.Join(slices.Sorted(slices.Values(config.SendNotificationsOn)), ","),
		}},
	)
	var verification currentVerification
	if err := api.get(ctx, "/verifications/"+references.verificationID, &verification); err != nil {
		return nil, fmt.Errorf("read manual verification after upgrade: %w", err)
	}
	items = append(items, upgradecheck.Item{Kind: "verification", Name: verification.ID, Values: map[string]string{
		"databaseId": verification.DatabaseID,
		"backupId":   verification.BackupID,
		"trigger":    verification.Trigger,
		"status":     verification.Status,
	}})
	return items, nil
}
