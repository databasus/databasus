package logical

import (
	"strconv"
	"strings"
)

const ignoredErrorsMarker = "errors ignored on restore:"

// A backup taken with include-schemas selects public explicitly, so pg_dump emits CREATE SCHEMA
// public and every restore target already owns that schema. pg_restore runs without
// --exit-on-error, so it reports "errors ignored on restore: N" and exits non-zero even though the
// data restored. Tolerating that is only safe when every ignored error is accounted for and is this
// one — a --clean restore never reaches here, since the DROP removes the collision first.
func isPreexistingPublicSchemaOnly(stderr string) bool {
	ignoredCount, hasMarker := parseIgnoredErrorCount(stderr)
	if !hasMarker || ignoredCount < 1 {
		return false
	}

	itemErrors := queryErrorLines(stderr)
	if len(itemErrors) != ignoredCount {
		return false
	}

	for _, line := range itemErrors {
		if !strings.Contains(strings.ToLower(line), `schema "public" already exists`) {
			return false
		}
	}

	return true
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
