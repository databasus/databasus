package upgrade

import (
	"context"
	"fmt"
	"net/http"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

func writeHealthcheckData(ctx context.Context, api *apiClient, databaseID string) ([]upgradecheck.Item, error) {
	config := map[string]any{
		"isHealthcheckEnabled":            false,
		"isNotificationEnabled":           false,
		"intervalMinutes":                 7,
		"failedAttemptsBeforeUnavailable": 4,
		"attemptRetentionDays":            11,
	}
	if err := api.send(ctx, http.MethodPut, "/healthcheck-configs/"+databaseID, config, nil); err != nil {
		return nil, fmt.Errorf("save the healthcheck config: %w", err)
	}
	return []upgradecheck.Item{
		{
			Kind: "healthcheck config",
			Name: databaseID,
			Values: map[string]string{
				"isHealthcheckEnabled":            "false",
				"isNotificationEnabled":           "false",
				"intervalMinutes":                 "7",
				"failedAttemptsBeforeUnavailable": "4",
				"attemptRetentionDays":            "11",
			},
		},
	}, nil
}
