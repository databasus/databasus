package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
)

// Managed providers append a suffix, "26.7.0-cloud", so only the leading numbers are read.
var serverVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)`)

func (s *Source) detectVersion(ctx context.Context, db *sql.DB) (Version, error) {
	var versionText string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&versionText); err != nil {
		return "", fmt.Errorf("failed to query MySQL version: %w", err)
	}

	return s.parseServerVersion(versionText)
}

// parseServerVersion turns what a server answers VERSION() with into the version identity that
// names its release line, so minors collapse onto it (8.1 → 8.0, 8.4+ → 8.4, every 26.x → 26). A
// line no shipped client serves is refused here rather than at backup time.
func (s *Source) parseServerVersion(versionText string) (Version, error) {
	matches := serverVersionPattern.FindStringSubmatch(versionText)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse MySQL version: %s", versionText)
	}

	version, err := mapServerVersion(matches[1], matches[2])
	if err != nil {
		return "", err
	}

	if err := s.clientTools.RequireBundle(version); err != nil {
		return "", err
	}

	return version, nil
}

func mapServerVersion(major, minor string) (Version, error) {
	switch major {
	case "5":
		return Version57, nil
	case "8":
		return mapServer8xVersion(minor), nil
	case "9":
		return Version9, nil
	// Every 26.x release belongs to one line, whatever its minor.
	case "26":
		return Version26, nil
	default:
		return "", fmt.Errorf(
			"unsupported MySQL major version: %s (supported: 5.x, 8.x, 9.x, 26.x)",
			major,
		)
	}
}

func mapServer8xVersion(minor string) Version {
	switch minor {
	case "0", "1", "2", "3":
		return Version80
	default:
		return Version84
	}
}
