package pgrestore

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const ignoredErrorsMarker = "errors ignored on restore:"

var extensionNamePattern = regexp.MustCompile(`extension "([^"]+)"`)

type IgnorableErrorRule int

const (
	PreexistingPublicSchema IgnorableErrorRule = iota
	UnavailableExtension
)

// Every skipped error must be accounted for; a missing line could hide lost data.
func HasOnlyIgnorableErrors(stderr string, rules ...IgnorableErrorRule) bool {
	ignoredCount, hasMarker := parseIgnoredErrorCount(stderr)
	if !hasMarker || ignoredCount < 1 {
		return false
	}

	itemErrors := queryErrorLines(stderr)
	if len(itemErrors) != ignoredCount {
		return false
	}

	for _, line := range itemErrors {
		if !slices.ContainsFunc(rules, func(rule IgnorableErrorRule) bool { return rule.matches(line) }) {
			return false
		}
	}

	return true
}

func GetUnavailableExtensions(stderr string) []string {
	seenExtensions := map[string]struct{}{}

	var extensionNames []string

	for _, line := range queryErrorLines(stderr) {
		if !isMissingExtensionLine(line) {
			continue
		}

		for _, extensionMatch := range extensionNamePattern.FindAllStringSubmatch(line, -1) {
			extensionName := extensionMatch[1]
			if _, isDup := seenExtensions[extensionName]; isDup {
				continue
			}

			seenExtensions[extensionName] = struct{}{}
			extensionNames = append(extensionNames, extensionName)
		}
	}

	slices.Sort(extensionNames)

	return extensionNames
}

func parseIgnoredErrorCount(stderr string) (int, bool) {
	markerIndex := strings.LastIndex(strings.ToLower(stderr), ignoredErrorsMarker)
	if markerIndex < 0 {
		return 0, false
	}

	fields := strings.Fields(stderr[markerIndex+len(ignoredErrorsMarker):])
	if len(fields) == 0 {
		return 0, false
	}

	count, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, false
	}

	return count, true
}

func queryErrorLines(stderr string) []string {
	var errorLines []string

	for line := range strings.SplitSeq(stderr, "\n") {
		if strings.Contains(strings.ToLower(line), "could not execute query: error:") {
			errorLines = append(errorLines, line)
		}
	}

	return errorLines
}

func (rule IgnorableErrorRule) matches(line string) bool {
	switch rule {
	case PreexistingPublicSchema:
		return isPreexistingPublicSchemaLine(line)
	case UnavailableExtension:
		return isMissingExtensionLine(line)
	default:
		return false
	}
}

// Selecting public with include-schemas makes pg_dump emit CREATE SCHEMA public,
// which collides with the restore target's existing schema. Without --exit-on-error,
// pg_restore still loads the data and exits nonzero. A blanket "already exists" rule
// would hide a partially restored archive; every error must match this schema.
func isPreexistingPublicSchemaLine(line string) bool {
	return strings.Contains(strings.ToLower(line), `schema "public" already exists`)
}

// The "extension" guard keeps unrelated cascades (`type "..." does not exist`) out, leaving the
// failed CREATE EXTENSION and its cascading COMMENT ON EXTENSION.
func isMissingExtensionLine(line string) bool {
	lowered := strings.ToLower(line)
	if !strings.Contains(lowered, "extension") {
		return false
	}

	return strings.Contains(lowered, "is not available") ||
		strings.Contains(lowered, "could not open extension control file") ||
		strings.Contains(lowered, "does not exist")
}
