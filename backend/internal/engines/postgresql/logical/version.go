package logical

import (
	"fmt"
	"regexp"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// Some builds omit the minor version: "PostgreSQL 16 maintained by ...".
var versionPattern = regexp.MustCompile(`PostgreSQL (\d+)`)

func parseVersion(rawServerVersion string) (pgshared.Version, error) {
	matches := versionPattern.FindStringSubmatch(rawServerVersion)
	if len(matches) < 2 {
		return "", fmt.Errorf("%w from: %s", ErrUnreadableVersion, rawServerVersion)
	}

	major := matches[1]
	version, isSupported := pgshared.FindSupportedVersion(major)
	if !isSupported {
		return "", fmt.Errorf(
			"%w: found PostgreSQL %s, supported versions are %s",
			ErrUnsupportedVersion, major, pgshared.GetSupportedVersionRange(),
		)
	}

	return version, nil
}
