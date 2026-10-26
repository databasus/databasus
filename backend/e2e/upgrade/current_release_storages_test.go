package upgrade

import (
	"context"
	"fmt"
	"net/url"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the storage API of the current code, like current_release_checks_test.go.

type currentStoragePage struct {
	Items      []currentStorage `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

type currentStorage struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Settings map[string]any `json:"settings"`
}

// listStorageItems follows the pages of the workspace's storages to the last and lists each with the
// settings the current code returns.
func listStorageItems(ctx context.Context, api *apiClient, workspaceID string) ([]upgradecheck.Item, error) {
	storageItems := []upgradecheck.Item{}
	query := url.Values{"workspaceId": {workspaceID}, "limit": {"100"}}

	for {
		var storagePage currentStoragePage

		err := api.get(ctx, "/storages?"+query.Encode(), &storagePage)
		if err != nil {
			return nil, fmt.Errorf("list storages: %w", err)
		}

		for _, storage := range storagePage.Items {
			storageValues := map[string]string{"id": storage.ID}
			for settingName, setting := range storage.Settings {
				storageValues[settingName] = formatSetting(setting)
			}

			storageItems = append(storageItems, upgradecheck.Item{
				Kind: "storage", Name: storage.Name, Values: storageValues,
			})
		}

		if storagePage.NextCursor == nil {
			return storageItems, nil
		}

		query.Set("after", *storagePage.NextCursor)
	}
}
