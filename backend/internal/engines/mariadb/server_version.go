package mariadb

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// MariaDB answers VERSION() with "10.11.6-MariaDB" or "11.4.2-MariaDB-1:11.4.2+maria~ubu2204", so
// only the leading numbers are read.
var serverVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)`)

func (s *Source) detectVersion(ctx context.Context, db *sql.DB) (Version, error) {
	var versionText string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&versionText); err != nil {
		return "", fmt.Errorf("failed to query MariaDB version: %w", err)
	}

	return s.parseServerVersion(versionText)
}

// parseServerVersion turns what a server answers VERSION() with into the version identity that
// names its release line, so minors collapse onto it (12.1 → 12.0, every 13.x → 13.0). A line no
// shipped client serves is refused here rather than at backup time.
func (s *Source) parseServerVersion(versionText string) (Version, error) {
	if !strings.Contains(strings.ToLower(versionText), "mariadb") {
		return "", fmt.Errorf(
			"not a MariaDB server (version: %s). Use MySQL database type instead",
			versionText,
		)
	}

	matches := serverVersionPattern.FindStringSubmatch(versionText)
	if len(matches) < 3 {
		return "", fmt.Errorf("could not parse MariaDB version: %s", versionText)
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
		return Version55, nil
	case "10":
		return mapServer10xVersion(minor), nil
	case "11":
		return mapServer11xVersion(minor), nil
	case "12":
		return Version120, nil
	// Every 13.x release belongs to one line, whatever its minor, and the 13.0 client serves all of
	// them.
	case "13":
		return Version130, nil
	default:
		return "", fmt.Errorf(
			"unsupported MariaDB major version: %s (supported: 5.x, 10.x, 11.x, 12.x, 13.x)",
			major,
		)
	}
}

func mapServer10xVersion(minor string) Version {
	switch minor {
	case "1":
		return Version101
	case "2":
		return Version102
	case "3":
		return Version103
	case "4":
		return Version104
	case "5":
		return Version105
	case "6", "7", "8", "9", "10":
		return Version106
	default:
		return Version1011
	}
}

func mapServer11xVersion(minor string) Version {
	switch minor {
	case "0", "1", "2", "3", "4":
		return Version114
	default:
		return Version118
	}
}
