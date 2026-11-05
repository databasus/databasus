// Package mongodb connects to MongoDB servers, inspects them, provisions read-only users and finds
// the MongoDB Database Tools.
package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"databasus-backend/internal/engines/clienttools"
)

type Version string

const (
	Version4 Version = "4"
	Version5 Version = "5"
	Version6 Version = "6"
	Version7 Version = "7"
	Version8 Version = "8"
	Version9 Version = "9"
)

// newestSupportedMajor is the newest major line the bundled MongoDB Database Tools list in their
// compatibility statement; raise it only with a tools release that lists the next line.
const newestSupportedMajor = 9

var majorMinorPattern = regexp.MustCompile(`^(\d+)\.(\d+)`)

// IsBackupVersionHigherThanRestoreVersion reports whether restoring a backup made on backupVersion
// onto restoreVersion would be a downgrade.
func IsBackupVersionHigherThanRestoreVersion(backupVersion, restoreVersion Version) (bool, error) {
	order, err := clienttools.CompareReleaseLines(string(backupVersion), string(restoreVersion))
	if err != nil {
		return false, fmt.Errorf("cannot order MongoDB versions: %w", err)
	}

	return order > 0, nil
}

func detectVersion(ctx context.Context, client *mongo.Client) (Version, error) {
	var buildInfo bson.M

	err := client.Database("admin").RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&buildInfo)
	if err != nil {
		return "", fmt.Errorf("failed to get MongoDB version: %w", err)
	}

	reportedVersion, isText := buildInfo["version"].(string)
	if !isText {
		return "", fmt.Errorf("%w from buildInfo", ErrUnreadableVersion)
	}

	return mapVersion(reportedVersion)
}

// mapVersion validates a reported server version against the supported range. The bundled MongoDB
// Database Tools drop wire-protocol v7, so the minimum supported server version is 4.2; see
// assets/tools/README.md for the bundled release.
func mapVersion(reportedVersion string) (Version, error) {
	matches := majorMinorPattern.FindStringSubmatch(reportedVersion)
	if len(matches) < 3 {
		return "", fmt.Errorf("%w: %q", ErrUnreadableVersion, reportedVersion)
	}

	major, majorErr := strconv.Atoi(matches[1])
	minor, minorErr := strconv.Atoi(matches[2])

	if majorErr != nil || minorErr != nil {
		return "", fmt.Errorf("%w: %q", ErrUnreadableVersion, reportedVersion)
	}

	if major < 4 || (major == 4 && minor < 2) {
		return "", fmt.Errorf("%w: %d.%d (minimum supported: 4.2)", ErrUnsupportedVersion, major, minor)
	}

	if major > newestSupportedMajor {
		return "", fmt.Errorf(
			"%w: %d.%d (supported: 4.2 through %d.x)",
			ErrUnsupportedVersion, major, minor, newestSupportedMajor,
		)
	}

	return Version(strconv.Itoa(major)), nil
}
