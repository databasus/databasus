package upgrade

import (
	"context"
	"fmt"
)

type currentInstanceBackupSettings struct {
	RetentionDays int `json:"retentionDays"`
	Schedule      struct {
		Type string `json:"type"`
	} `json:"schedule"`
	StorageIDs []string `json:"storageIds"`
}

type currentInstanceBackupStatus struct {
	Recovery *struct {
		ID string `json:"id"`
	} `json:"recovery"`
}

// checkInstanceBackupDefaults checks that the upgrade created the Databasus backup tables with
// their defaults: no destination, an hourly schedule, 30-day retention and no recovery.
func checkInstanceBackupDefaults(ctx context.Context, api *apiClient) error {
	var settings currentInstanceBackupSettings
	if err := api.get(ctx, "/instance-backup-settings", &settings); err != nil {
		return fmt.Errorf("read the Databasus backup settings after the upgrade: %w", err)
	}

	if settings.RetentionDays != 30 || settings.Schedule.Type != "HOURLY" || len(settings.StorageIDs) != 0 {
		return fmt.Errorf("the upgraded Databasus backup settings are %+v, not the defaults", settings)
	}

	var status currentInstanceBackupStatus
	if err := api.get(ctx, "/instance-backup-statuses", &status); err != nil {
		return fmt.Errorf("read the Databasus backup status after the upgrade: %w", err)
	}

	if status.Recovery != nil {
		return fmt.Errorf("an upgraded installation reports recovery %s as if it were recovered", status.Recovery.ID)
	}

	return nil
}
