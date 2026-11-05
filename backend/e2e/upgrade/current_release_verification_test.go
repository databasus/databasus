package upgrade

import (
	"context"
	"fmt"
	"net/url"
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
	IsAutomaticVerificationEnabled bool   `json:"isAutomaticVerificationEnabled"`
	ScheduleType                   string `json:"scheduleType"`
	Schedule                       struct {
		Type      string  `json:"type"`
		Weekday   *int    `json:"weekday"`
		TimeOfDay *string `json:"timeOfDay"`
	} `json:"schedule"`
	NotificationTriggers []string `json:"notificationTriggers"`
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
	path := "/verification-agents?limit=100"
	for {
		var page struct {
			Items      []currentVerificationAgent `json:"items"`
			NextCursor *string                    `json:"nextCursor"`
		}
		if err := api.get(ctx, path, &page); err != nil {
			return nil, fmt.Errorf("list verification agents after upgrade: %w", err)
		}
		agents = append(agents, page.Items...)
		if page.NextCursor == nil {
			break
		}
		path = "/verification-agents?limit=100&after=" + url.QueryEscape(*page.NextCursor)
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
	if err := api.get(ctx, "/verification-configs/"+references.databaseID, &config); err != nil {
		return nil, fmt.Errorf("read verification config after upgrade: %w", err)
	}

	weekday := ""
	if config.Schedule.Weekday != nil {
		weekday = strconv.Itoa(*config.Schedule.Weekday)
	}

	items = append(
		items,
		upgradecheck.Item{Kind: "verification config", Name: references.databaseID, Values: map[string]string{
			"isScheduledVerificationEnabled": strconv.FormatBool(config.IsAutomaticVerificationEnabled),
			"scheduleType":                   config.ScheduleType, "interval.type": config.Schedule.Type,
			"interval.weekday": weekday, "interval.timeOfDay": getOptionalValue(config.Schedule.TimeOfDay),
			"sendNotificationsOn": strings.Join(slices.Sorted(slices.Values(config.NotificationTriggers)), ","),
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
