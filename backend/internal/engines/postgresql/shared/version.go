// Package shared holds what logical and physical PostgreSQL sources share: the server versions,
// the client tools, TLS, credential files and connection errors. It is imported as pgshared.
package shared

import (
	"fmt"
	"strconv"
)

type Version string

const (
	Version12 Version = "12"
	Version13 Version = "13"
	Version14 Version = "14"
	Version15 Version = "15"
	Version16 Version = "16"
	Version17 Version = "17"
	Version18 Version = "18"
	Version19 Version = "19"
)

// Incremental backups, and pg_combinebackup that joins them on restore, exist from PostgreSQL 17.
const physicalBackupMinimumMajor = 17

// pg_dump learned zstd compression in PostgreSQL 16.
const zstdDumpMinimumMajor = 16

// getSupportedVersions lists the majors with a client bundle. It is the only list of versions: every
// other decision derives from the major number.
func getSupportedVersions() []Version {
	return []Version{Version12, Version13, Version14, Version15, Version16, Version17, Version18, Version19}
}

// GetNumber returns the major as a number, or 0 when the version is not a number.
func (v Version) GetNumber() int {
	number, err := strconv.Atoi(string(v))
	if err != nil {
		return 0
	}

	return number
}

func (v Version) CanBackUpPhysically() bool {
	return v.GetNumber() >= physicalBackupMinimumMajor
}

func (v Version) CanDumpWithZstd() bool {
	return v.GetNumber() >= zstdDumpMinimumMajor
}

// FindSupportedVersion returns the version of a reported major when a client bundle serves it.
func FindSupportedVersion(major string) (Version, bool) {
	for _, version := range getSupportedVersions() {
		if string(version) == major {
			return version, true
		}
	}

	return "", false
}

// GetSupportedVersionRange names the majors logical backups support, such as "PostgreSQL 12 to 19".
func GetSupportedVersionRange() string {
	supportedVersions := getSupportedVersions()

	return formatVersionRange(supportedVersions[0], supportedVersions[len(supportedVersions)-1])
}

// GetPhysicalBackupVersionRange names the majors physical backups support, such as
// "PostgreSQL 17 to 19".
func GetPhysicalBackupVersionRange() string {
	supportedVersions := getSupportedVersions()

	return formatVersionRange(
		Version(strconv.Itoa(physicalBackupMinimumMajor)),
		supportedVersions[len(supportedVersions)-1],
	)
}

func formatVersionRange(oldestVersion, newestVersion Version) string {
	if oldestVersion == newestVersion {
		return fmt.Sprintf("PostgreSQL %s", oldestVersion)
	}

	return fmt.Sprintf("PostgreSQL %s to %s", oldestVersion, newestVersion)
}

func IsBackupVersionHigherThanRestoreVersion(backupVersion, restoreVersion Version) bool {
	backupVersionNumber := backupVersion.GetNumber()
	restoreVersionNumber := restoreVersion.GetNumber()
	if backupVersionNumber == 0 || restoreVersionNumber == 0 {
		return false
	}

	return backupVersionNumber > restoreVersionNumber
}
