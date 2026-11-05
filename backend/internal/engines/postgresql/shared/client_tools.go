package shared

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"databasus-backend/internal/engines/clienttools"
)

type Executable string

const (
	ExecutablePgDump          Executable = "pg_dump"
	ExecutablePgRestore       Executable = "pg_restore"
	ExecutablePsql            Executable = "psql"
	ExecutablePgBasebackup    Executable = "pg_basebackup"
	ExecutablePgReceivewal    Executable = "pg_receivewal"
	ExecutablePgCombinebackup Executable = "pg_combinebackup"
)

// Every PostgreSQL client answers --version with "<tool> (PostgreSQL) 17.10".
var clientVersionPattern = regexp.MustCompile(`\(PostgreSQL\)\s+(\d[\d.]*)`)

// ClientTools finds the PostgreSQL clients under the bundle root.
type ClientTools struct {
	Root string
}

func (c ClientTools) GetExecutable(version Version, executable Executable) string {
	return filepath.Join(c.getBinDir(version), string(executable))
}

func (c ClientTools) getBinDir(version Version) string {
	return filepath.Join(c.Root, "postgresql", fmt.Sprintf("postgresql-%s", version), "bin")
}

// GetClientToolBundles declares one fatal bundle per supported version: the backend reads the
// version of every managed database and must be able to start the matching client.
func GetClientToolBundles(root string) []clienttools.Bundle {
	clientTools := ClientTools{Root: root}
	supportedVersions := getSupportedVersions()
	bundles := make([]clienttools.Bundle, 0, len(supportedVersions))

	for _, version := range supportedVersions {
		bundles = append(bundles, clienttools.Bundle{
			Engine:           "postgresql",
			Version:          string(version),
			BinDir:           clientTools.getBinDir(version),
			RequiredCommands: getRequiredCommands(version),
			ReleaseLine:      string(version),
			ParseVersion:     parseClientVersion,
			IsFatal:          true,
		})
	}

	return bundles
}

// pg_basebackup --incremental and pg_combinebackup exist from PostgreSQL 17, so only those bundles
// must hold the physical backup clients.
func getRequiredCommands(version Version) []string {
	requiredCommands := []string{string(ExecutablePgDump), string(ExecutablePsql)}

	if version == Version17 || version == Version18 {
		requiredCommands = append(
			requiredCommands,
			string(ExecutablePgBasebackup),
			string(ExecutablePgReceivewal),
			string(ExecutablePgCombinebackup),
		)
	}

	return requiredCommands
}

func parseClientVersion(output string) (string, error) {
	match := clientVersionPattern.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("%w: %q", ErrUnreadableClientVersion, strings.TrimSpace(output))
	}

	return match[1], nil
}
