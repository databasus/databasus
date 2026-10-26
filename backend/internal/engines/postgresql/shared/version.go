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
)

func getSupportedVersions() []Version {
	return []Version{Version12, Version13, Version14, Version15, Version16, Version17, Version18}
}

// GetVersionEnum panics on a version the backend does not support: every caller passes a value that
// was detected and checked against the supported versions before it was stored.
func GetVersionEnum(version string) Version {
	for _, supportedVersion := range getSupportedVersions() {
		if string(supportedVersion) == version {
			return supportedVersion
		}
	}

	panic(fmt.Sprintf("invalid postgresql version: %s", version))
}

func IsBackupVersionHigherThanRestoreVersion(backupVersion, restoreVersion Version) bool {
	backupVersionNumber, err := strconv.Atoi(string(backupVersion))
	if err != nil {
		return false
	}

	restoreVersionNumber, err := strconv.Atoi(string(restoreVersion))
	if err != nil {
		return false
	}

	return backupVersionNumber > restoreVersionNumber
}
