package mysql

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"databasus-backend/internal/engines/clienttools"
)

type Executable string

const (
	ExecutableMysqldump Executable = "mysqldump"
	ExecutableMysql     Executable = "mysql"
)

// MySQL 8.0 and later answer --version with "mysqldump  Ver 8.0.46 for ...", while 5.7 answers
// with its own tool version and puts the server line behind "Distrib":
// "mysqldump  Ver 10.13 Distrib 5.7.44, for ...".
var (
	distribVersionPattern = regexp.MustCompile(`Distrib\s+(\d[\d.]*)`)
	verVersionPattern     = regexp.MustCompile(`Ver\s+(\d[\d.]*)`)
)

// ClientTools finds the MySQL clients under the bundle root.
type ClientTools struct {
	Root string
}

// GetExecutable fails when this architecture ships no bundle for the version (upstream never built
// a 5.7 client for arm64), so the caller reports a supported-version problem instead of a missing
// file.
func (c ClientTools) GetExecutable(version Version, executable Executable) (string, error) {
	if err := c.RequireBundle(version); err != nil {
		return "", err
	}

	return filepath.Join(c.getBinDir(version), string(executable)), nil
}

// RequireBundle reports whether this architecture ships a client for the version, naming the
// architecture when it does not.
func (c ClientTools) RequireBundle(version Version) error {
	if fileErrors := clienttools.CheckBinDir(c.getBinDir(version), getRequiredCommands()); len(fileErrors) > 0 {
		return fmt.Errorf(
			"MySQL %s is not supported on %s: %w", version, filepath.Base(c.Root), ErrNoClientForArchitecture,
		)
	}

	return nil
}

func (c ClientTools) getBinDir(version Version) string {
	return filepath.Join(c.Root, "mysql", fmt.Sprintf("mysql-%s", version), "bin")
}

// GetClientToolBundles declares one bundle per supported version. A missing bundle disables that
// version only, so none of them is fatal.
func GetClientToolBundles(root string) []clienttools.Bundle {
	clientTools := ClientTools{Root: root}
	supportedVersions := getSupportedVersions()
	bundles := make([]clienttools.Bundle, 0, len(supportedVersions))

	for _, version := range supportedVersions {
		if version == Version57 && filepath.Base(root) == "arm" {
			continue
		}

		bundles = append(bundles, clienttools.Bundle{
			Engine:           "mysql",
			Version:          string(version),
			BinDir:           clientTools.getBinDir(version),
			RequiredCommands: getRequiredCommands(),
			ReleaseLine:      string(version),
			ParseVersion:     parseClientVersion,
			IsFatal:          false,
		})
	}

	return bundles
}

func getRequiredCommands() []string {
	return []string{string(ExecutableMysqldump), string(ExecutableMysql)}
}

func parseClientVersion(output string) (string, error) {
	if match := distribVersionPattern.FindStringSubmatch(output); match != nil {
		return strings.TrimSuffix(match[1], "."), nil
	}

	if match := verVersionPattern.FindStringSubmatch(output); match != nil {
		return strings.TrimSuffix(match[1], "."), nil
	}

	return "", fmt.Errorf("%w: %q", ErrUnreadableClientVersion, strings.TrimSpace(output))
}
