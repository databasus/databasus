// Package mysql connects to MySQL servers, inspects them, provisions read-only users and finds the
// MySQL client tools.
package mysql

import (
	"fmt"

	"databasus-backend/internal/engines/clienttools"
)

type Version string

const (
	Version57 Version = "5.7"
	Version80 Version = "8.0"
	Version84 Version = "8.4"
	Version9  Version = "9"
	Version26 Version = "26"
)

func getSupportedVersions() []Version {
	return []Version{Version57, Version80, Version84, Version9, Version26}
}

// IsBackupVersionHigherThanRestoreVersion reports whether restoring a backup made on backupVersion
// onto restoreVersion would be a downgrade.
func IsBackupVersionHigherThanRestoreVersion(backupVersion, restoreVersion Version) (bool, error) {
	order, err := clienttools.CompareReleaseLines(string(backupVersion), string(restoreVersion))
	if err != nil {
		return false, fmt.Errorf("cannot order MySQL versions: %w", err)
	}

	return order > 0, nil
}
