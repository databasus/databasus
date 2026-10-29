package restore

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const ignoredErrorsMarker = "errors ignored on restore:"

var extensionNamePattern = regexp.MustCompile(`extension "([^"]+)"`)

// The count guard matters because StderrTail is capped at 8192 bytes: a truncated tail could hide an
// error we would never tolerate, so when N can't be fully accounted for we refuse to tolerate.
func IsToleratedErrorsOnly(stderrTail string) bool {
	ignoredCount, hasMarker := parseIgnoredErrorCount(stderrTail)
	if !hasMarker || ignoredCount < 1 {
		return false
	}

	itemErrors := queryErrorLines(stderrTail)
	if len(itemErrors) != ignoredCount {
		return false
	}

	for _, line := range itemErrors {
		if !isToleratedErrorLine(line) {
			return false
		}
	}

	return true
}

// ExtractUnavailableExtensions returns the de-duplicated, sorted names of the
// extensions pg_restore could not create, for an operator-facing log line.
func ExtractUnavailableExtensions(stderrTail string) []string {
	seen := map[string]struct{}{}

	var names []string

	for _, line := range queryErrorLines(stderrTail) {
		if !isMissingExtensionLine(line) {
			continue
		}

		for _, match := range extensionNamePattern.FindAllStringSubmatch(line, -1) {
			name := match[1]
			if _, isDup := seen[name]; isDup {
				continue
			}

			seen[name] = struct{}{}
			names = append(names, name)
		}
	}

	slices.Sort(names)

	return names
}

func parseIgnoredErrorCount(stderrTail string) (int, bool) {
	idx := strings.LastIndex(strings.ToLower(stderrTail), ignoredErrorsMarker)
	if idx < 0 {
		return 0, false
	}

	fields := strings.Fields(stderrTail[idx+len(ignoredErrorsMarker):])
	if len(fields) == 0 {
		return 0, false
	}

	count, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, false
	}

	return count, true
}

func queryErrorLines(stderrTail string) []string {
	var lines []string

	for line := range strings.SplitSeq(stderrTail, "\n") {
		if strings.Contains(strings.ToLower(line), "could not execute query: error:") {
			lines = append(lines, line)
		}
	}

	return lines
}

func isToleratedErrorLine(line string) bool {
	return isMissingExtensionLine(line) || isPreexistingPublicSchemaLine(line)
}

// The archive was dumped with -n public, which makes pg_dump emit the schema definition too, and
// every freshly initdb'd target already owns public (issue #726). Deliberately narrow: a blanket
// "already exists" rule would hide a genuinely half-restored archive.
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
