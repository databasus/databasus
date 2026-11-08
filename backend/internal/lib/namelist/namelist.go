// Package namelist normalizes and formats user-supplied lists of database object names.
//
// User-supplied lists may contain pasted separators and surrounding whitespace. Database
// dump tools match names literally, so unnormalized exclusions can silently leave the
// named objects in a backup.
package namelist

import (
	"fmt"
	"strings"
)

func NormalizeUniqueNames(names []string) []string {
	normalizedNames := make([]string, 0, len(names))
	seenNames := make(map[string]struct{}, len(names))

	for _, name := range names {
		for _, splitName := range splitOnSeparators(name) {
			if _, isSeen := seenNames[splitName]; isSeen {
				continue
			}

			seenNames[splitName] = struct{}{}
			normalizedNames = append(normalizedNames, splitName)
		}
	}

	return normalizedNames
}

// FormatTruncatedNames keeps an error message readable when the caller found hundreds of offending
// objects: the reader needs a few examples and the scale, not the whole list.
func FormatTruncatedNames(names []string) string {
	const maxListedNames = 5

	if len(names) <= maxListedNames {
		return strings.Join(names, ", ")
	}

	return fmt.Sprintf(
		"%s and %d more",
		strings.Join(names[:maxListedNames], ", "),
		len(names)-maxListedNames,
	)
}

func splitOnSeparators(rawNames string) []string {
	splitNames := make([]string, 0, 1)

	for _, name := range strings.FieldsFunc(rawNames, isSeparator) {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		splitNames = append(splitNames, name)
	}

	return splitNames
}

func isSeparator(char rune) bool {
	return char == ',' || char == '\n' || char == '\r' || char == '\t'
}
