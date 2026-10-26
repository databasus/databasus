// Package mariadb connects to MariaDB servers, inspects them, provisions read-only users and finds
// the MariaDB client tools.
package mariadb

import (
	"fmt"

	"databasus-backend/internal/engines/clienttools"
)

type Version string

const (
	Version55   Version = "5.5"
	Version101  Version = "10.1"
	Version102  Version = "10.2"
	Version103  Version = "10.3"
	Version104  Version = "10.4"
	Version105  Version = "10.5"
	Version106  Version = "10.6"
	Version1011 Version = "10.11"
	Version114  Version = "11.4"
	Version118  Version = "11.8"
	Version120  Version = "12.0"
	Version130  Version = "13.0"
)

// ClientVersion is a client tool version shipped under the bundle root.
type ClientVersion string

const (
	// ClientLegacy serves the 5.5 and 10.1 servers, which lack the generation_expression column of
	// information_schema.columns that newer clients query.
	ClientLegacy ClientVersion = "10.6"
	// ClientModern serves the servers from 10.2 up to the 12 line.
	ClientModern ClientVersion = "12.3"
	// Client13 serves the 13 line, which the 12.3 client predates.
	Client13 ClientVersion = "13.0"
)

func getShippedClientVersions() []ClientVersion {
	return []ClientVersion{ClientLegacy, ClientModern, Client13}
}

func GetClientVersionForServer(serverVersion Version) ClientVersion {
	switch serverVersion {
	case Version55, Version101:
		return ClientLegacy
	case Version130:
		return Client13
	case Version102, Version103, Version104, Version105, Version106, Version1011, Version114, Version118,
		Version120:
		return ClientModern
	default:
		return ClientModern
	}
}

// IsBackupVersionHigherThanRestoreVersion reports whether restoring a backup made on backupVersion
// onto restoreVersion would be a downgrade.
func IsBackupVersionHigherThanRestoreVersion(backupVersion, restoreVersion Version) (bool, error) {
	order, err := clienttools.CompareReleaseLines(string(backupVersion), string(restoreVersion))
	if err != nil {
		return false, fmt.Errorf("cannot order MariaDB versions: %w", err)
	}

	return order > 0, nil
}
