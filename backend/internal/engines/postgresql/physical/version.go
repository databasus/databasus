package physical

import (
	"fmt"
	"regexp"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

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
			"%w: found PostgreSQL %s, physical backups support %s",
			ErrUnsupportedVersion, major, pgshared.GetPhysicalBackupVersionRange(),
		)
	}

	if !version.CanBackUpPhysically() {
		return "", fmt.Errorf(
			"%w: found PostgreSQL %s; use a logical backup, which covers the same data",
			ErrVersionTooOldForPhysicalBackup, major,
		)
	}

	return version, nil
}
