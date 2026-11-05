package upgrade

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the database API of the current code, like current_release_checks_test.go.

type currentDatabasePage struct {
	Items      []currentDatabase `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type currentDatabase struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	NotifierIDs []string       `json:"notifierIds"`
	Settings    map[string]any `json:"settings"`
}

type currentBackupConfig struct {
	Cleanup *struct {
		PendingFileCount int64 `json:"pendingFileCount"`
		FailedFileCount  int64 `json:"failedFileCount"`
	} `json:"cleanup"`
	IsBackupsEnabled bool           `json:"isBackupsEnabled"`
	StorageID        *string        `json:"storageId"`
	Encryption       string         `json:"encryption"`
	Settings         map[string]any `json:"settings"`
}

type currentBackupPage struct {
	Items []currentBackup `json:"items"`
}

type currentBackup struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Encryption string `json:"encryption"`
}

// listDatabaseItems follows the pages of the workspace's databases to the last and lists each with
// the settings the current code returns, the notifiers it uses, its backup config and its backups.
// The notifier items name the notifiers the databases refer to by ID.
func listDatabaseItems(
	ctx context.Context,
	api *apiClient,
	workspaceID string,
	notifierItems []upgradecheck.Item,
) ([]upgradecheck.Item, error) {
	notifierNamesByID := make(map[string]string, len(notifierItems))
	for _, notifierItem := range notifierItems {
		notifierNamesByID[notifierItem.Values["id"]] = notifierItem.Name
	}

	databaseItems := []upgradecheck.Item{}
	query := url.Values{"workspaceId": {workspaceID}, "limit": {"100"}}

	for {
		var databasePage currentDatabasePage

		err := api.get(ctx, "/databases?"+query.Encode(), &databasePage)
		if err != nil {
			return nil, fmt.Errorf("list databases: %w", err)
		}

		for _, database := range databasePage.Items {
			usedNotifiers := make(map[string]string, len(database.NotifierIDs))
			for _, notifierID := range database.NotifierIDs {
				usedNotifiers[notifierNamesByID[notifierID]] = notifierID
			}

			backupItems, readErr := readBackupItems(ctx, api, database)
			if readErr != nil {
				return nil, readErr
			}

			databaseItems = append(databaseItems,
				newDatabaseItem(database),
				upgradecheck.Item{Kind: databaseNotifiersKind, Name: database.Name, Values: usedNotifiers},
			)
			databaseItems = append(databaseItems, backupItems...)
		}

		if databasePage.NextCursor == nil {
			return databaseItems, nil
		}

		query.Set("after", *databasePage.NextCursor)
	}
}

// newDatabaseItem writes every setting as text. A nested setting, such as the SSH tunnel, becomes
// one value per field under its name.
func newDatabaseItem(database currentDatabase) upgradecheck.Item {
	values := map[string]string{"id": database.ID}

	for settingName, setting := range database.Settings {
		nestedSettings, isNested := setting.(map[string]any)
		if !isNested {
			values[settingName] = formatSetting(setting)

			continue
		}

		for nestedName, nestedSetting := range nestedSettings {
			values[settingName+"."+nestedName] = formatSetting(nestedSetting)
		}
	}

	return upgradecheck.Item{Kind: "database", Name: database.Name, Values: values}
}

// readBackupItems lists the database's backup config and backups. Only the PostgreSQL database was
// backed up before the upgrade, so a config without a storage is listed with an empty one.
func readBackupItems(ctx context.Context, api *apiClient, database currentDatabase) ([]upgradecheck.Item, error) {
	var backupConfig currentBackupConfig

	err := api.get(ctx, "/backup-configs/"+database.ID, &backupConfig)
	if err != nil {
		return nil, fmt.Errorf("read the backup config of database %q: %w", database.Name, err)
	}

	if backupConfig.Cleanup == nil || backupConfig.Cleanup.PendingFileCount != 0 ||
		backupConfig.Cleanup.FailedFileCount != 0 {
		return nil, fmt.Errorf("database %q must have zero computed cleanup counts after upgrade", database.Name)
	}

	if database.Settings["type"] == "POSTGRES_PHYSICAL" {
		return nil, nil
	}

	var backupPage currentBackupPage

	err = api.get(ctx, "/logical-backups?limit=100&databaseId="+database.ID, &backupPage)
	if err != nil {
		return nil, fmt.Errorf("list backups of database %q: %w", database.Name, err)
	}

	backupConfigValues := map[string]string{
		"isBackupsEnabled": strconv.FormatBool(backupConfig.IsBackupsEnabled),
		"storageId":        getOptionalValue(backupConfig.StorageID),
		"encryption":       backupConfig.Encryption,
	}
	addFlattenedSettings(backupConfigValues, "settings", backupConfig.Settings)

	backupItems := []upgradecheck.Item{
		{Kind: "backup config", Name: database.Name, Values: backupConfigValues},
	}

	for _, backup := range backupPage.Items {
		backupItems = append(backupItems, upgradecheck.Item{
			Kind:   "backup",
			Name:   backup.ID,
			Values: map[string]string{"status": backup.Status, "encryption": backup.Encryption},
		})
	}

	return backupItems, nil
}

// addFlattenedSettings names a nested setting by its path, such as settings.schedule.type.
func addFlattenedSettings(values map[string]string, path string, settings map[string]any) {
	for settingName, setting := range settings {
		settingPath := path + "." + settingName

		if nestedSettings, isNested := setting.(map[string]any); isNested {
			addFlattenedSettings(values, settingPath, nestedSettings)

			continue
		}

		values[settingPath] = formatSetting(setting)
	}
}
