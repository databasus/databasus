// Package upgradecheck compares the items the upgrade test wrote through the previous release with
// the items the current code lists, so a migration that loses a value fails with its item and value.
package upgradecheck

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Item is one thing a user created, such as a storage or a notifier, with the values the test
// compares after the upgrade. Kind and Name identify it in both lists.
type Item struct {
	Kind   string
	Name   string
	Values map[string]string
}

// Compare returns ErrItemsDiffer naming every written item that is not listed and every value that
// changed. An item listed but never written is no difference: the current code may add its own.
func Compare(writtenItems, listedItems []Item) error {
	var differences []string

	for _, writtenItem := range writtenItems {
		listedIndex := slices.IndexFunc(listedItems, func(listedItem Item) bool {
			return listedItem.Kind == writtenItem.Kind && listedItem.Name == writtenItem.Name
		})
		if listedIndex < 0 {
			differences = append(differences,
				fmt.Sprintf("%s %q is not listed after the upgrade", writtenItem.Kind, writtenItem.Name))

			continue
		}

		listedValues := listedItems[listedIndex].Values

		for _, valueName := range slices.Sorted(maps.Keys(writtenItem.Values)) {
			writtenValue := writtenItem.Values[valueName]

			listedValue, isListed := listedValues[valueName]
			if !isListed {
				differences = append(differences, fmt.Sprintf("%s %q: %s is missing after the upgrade, was %q",
					writtenItem.Kind, writtenItem.Name, valueName, writtenValue))

				continue
			}

			if listedValue != writtenValue {
				differences = append(differences, fmt.Sprintf("%s %q: %s is %q after the upgrade, was %q",
					writtenItem.Kind, writtenItem.Name, valueName, listedValue, writtenValue))
			}
		}
	}

	if len(differences) > 0 {
		return fmt.Errorf("%w:\n%s", ErrItemsDiffer, strings.Join(differences, "\n"))
	}

	return nil
}
