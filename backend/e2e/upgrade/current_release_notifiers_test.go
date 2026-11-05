package upgrade

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"databasus-backend/e2e/upgrade/upgradecheck"
)

// This file speaks the notifier API of the current code, like current_release_checks_test.go.

type currentNotifierPage struct {
	Items      []currentNotifier `json:"items"`
	NextCursor *string           `json:"nextCursor"`
}

type currentNotifier struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Settings map[string]any `json:"settings"`
}

// listNotifierItems follows the pages of the workspace's notifiers to the last and lists each with
// the settings the current code returns.
func listNotifierItems(ctx context.Context, api *apiClient, workspaceID string) ([]upgradecheck.Item, error) {
	notifierItems := []upgradecheck.Item{}
	query := url.Values{"workspaceId": {workspaceID}, "limit": {"100"}}

	for {
		var notifierPage currentNotifierPage

		err := api.get(ctx, "/notifiers?"+query.Encode(), &notifierPage)
		if err != nil {
			return nil, fmt.Errorf("list notifiers: %w", err)
		}

		for _, notifier := range notifierPage.Items {
			notifierItems = append(notifierItems, newNotifierItem(notifier))
		}

		if notifierPage.NextCursor == nil {
			return notifierItems, nil
		}

		query.Set("after", *notifierPage.NextCursor)
	}
}

// newNotifierItem writes every setting as text: a list of webhook headers becomes their names and
// any other list its values, both joined with commas.
func newNotifierItem(notifier currentNotifier) upgradecheck.Item {
	values := map[string]string{"id": notifier.ID}

	for settingName, setting := range notifier.Settings {
		values[settingName] = formatSetting(setting)
	}

	return upgradecheck.Item{Kind: "notifier", Name: notifier.Name, Values: values}
}

func formatSetting(setting any) string {
	switch typedSetting := setting.(type) {
	case string:
		return typedSetting
	case bool:
		return strconv.FormatBool(typedSetting)
	case float64:
		return strconv.FormatFloat(typedSetting, 'f', -1, 64)
	case []any:
		elements := make([]string, 0, len(typedSetting))

		for _, element := range typedSetting {
			if header, isHeader := element.(map[string]any); isHeader {
				elements = append(elements, formatSetting(header["name"]))

				continue
			}

			elements = append(elements, formatSetting(element))
		}

		return strings.Join(elements, ",")
	case nil:
		return ""
	}

	return fmt.Sprint(setting)
}
