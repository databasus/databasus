package main

import (
	"context"
	"net/url"
)

// listAll reads every page of a contract list. The path already carries its query.
func listAll(ctx context.Context, client *apiClient, path string) ([]namedItem, error) {
	var listedItems []namedItem

	pagePath := path

	for {
		var page namedItemPage
		if err := client.get(ctx, pagePath, &page); err != nil {
			return nil, err
		}

		listedItems = append(listedItems, page.Items...)

		if page.NextCursor == nil {
			return listedItems, nil
		}

		pagePath = path + "&after=" + url.QueryEscape(*page.NextCursor)
	}
}

func findNamedItem(listedItems []namedItem, name string) (namedItem, bool) {
	for _, listedItem := range listedItems {
		if listedItem.Name == name {
			return listedItem, true
		}
	}

	return namedItem{}, false
}
