package upgrade

import (
	"context"
	"fmt"
	"strconv"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

type currentHealthcheckConfig struct {
	IsHealthcheckEnabled            bool `json:"isHealthcheckEnabled"`
	IsNotificationEnabled           bool `json:"isNotificationEnabled"`
	IntervalMinutes                 int  `json:"intervalMinutes"`
	FailedAttemptsBeforeUnavailable int  `json:"failedAttemptsBeforeUnavailable"`
	AttemptRetentionDays            int  `json:"attemptRetentionDays"`
}

func readHealthcheckItems(ctx context.Context, api *apiClient, databaseID string) ([]upgradecheck.Item, error) {
	var config currentHealthcheckConfig
	if err := api.get(ctx, "/healthcheck-configs/"+databaseID, &config); err != nil {
		return nil, fmt.Errorf("read healthcheck config after upgrade: %w", err)
	}
	return []upgradecheck.Item{
		{
			Kind: "healthcheck config",
			Name: databaseID,
			Values: map[string]string{
				"isHealthcheckEnabled":            strconv.FormatBool(config.IsHealthcheckEnabled),
				"isNotificationEnabled":           strconv.FormatBool(config.IsNotificationEnabled),
				"intervalMinutes":                 strconv.Itoa(config.IntervalMinutes),
				"failedAttemptsBeforeUnavailable": strconv.Itoa(config.FailedAttemptsBeforeUnavailable),
				"attemptRetentionDays":            strconv.Itoa(config.AttemptRetentionDays),
			},
		},
	}, nil
}
