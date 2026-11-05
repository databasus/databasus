package mariadb

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"databasus-backend/internal/engines/clienttools"
)

type Executable string

const (
	ExecutableMariadbDump Executable = "mariadb-dump"
	ExecutableMariadb     Executable = "mariadb"
)

// Both MariaDB --version shapes put the server line immediately before "-MariaDB":
// "mariadb-dump  Ver 10.19 Distrib 10.6.28-MariaDB, for ..." on 10.6, "mariadb-dump from
// 13.0.2-MariaDB, client 10.20 for ..." on newer clients. The other number in each line is the
// client protocol version.
var clientVersionPattern = regexp.MustCompile(`(\d[\d.]*)-MariaDB`)

// ClientTools finds the MariaDB clients under the bundle root.
type ClientTools struct {
	Root string
}

// GetExecutable returns the client designated for the server version. It fails when this
// architecture ships no bundle for that client, so the caller reports a supported-version problem
// instead of a missing file.
func (c ClientTools) GetExecutable(serverVersion Version, executable Executable) (string, error) {
	if err := c.RequireBundle(serverVersion); err != nil {
		return "", err
	}

	return filepath.Join(c.getBinDir(GetClientVersionForServer(serverVersion)), string(executable)), nil
}

// RequireBundle reports whether this architecture ships the client designated for the server
// version, naming the architecture when it does not.
func (c ClientTools) RequireBundle(serverVersion Version) error {
	binDir := c.getBinDir(GetClientVersionForServer(serverVersion))

	if fileErrors := clienttools.CheckBinDir(binDir, getRequiredCommands()); len(fileErrors) > 0 {
		return fmt.Errorf(
			"MariaDB %s is not supported on %s: %w", serverVersion, filepath.Base(c.Root), ErrNoClientForArchitecture,
		)
	}

	return nil
}

func (c ClientTools) getBinDir(clientVersion ClientVersion) string {
	return filepath.Join(c.Root, "mariadb", fmt.Sprintf("mariadb-%s", clientVersion), "bin")
}

// GetClientToolBundles declares one bundle per shipped client. A missing bundle disables the
// servers that client serves, so none of them is fatal.
func GetClientToolBundles(root string) []clienttools.Bundle {
	clientTools := ClientTools{Root: root}
	shippedClientVersions := getShippedClientVersions()
	bundles := make([]clienttools.Bundle, 0, len(shippedClientVersions))

	for _, clientVersion := range shippedClientVersions {
		bundles = append(bundles, clienttools.Bundle{
			Engine:           "mariadb",
			Version:          string(clientVersion),
			BinDir:           clientTools.getBinDir(clientVersion),
			RequiredCommands: getRequiredCommands(),
			ReleaseLine:      string(clientVersion),
			ParseVersion:     parseClientVersion,
			IsFatal:          false,
		})
	}

	return bundles
}

func getRequiredCommands() []string {
	return []string{string(ExecutableMariadbDump), string(ExecutableMariadb)}
}

func parseClientVersion(output string) (string, error) {
	match := clientVersionPattern.FindStringSubmatch(output)
	if match == nil {
		return "", fmt.Errorf("%w: %q", ErrUnreadableClientVersion, strings.TrimSpace(output))
	}

	return match[1], nil
}
