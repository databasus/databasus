package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionNumberRules(t *testing.T) {
	t.Parallel()

	for _, versionCase := range []struct {
		name                     string
		version                  Version
		expectedNumber           int
		isPhysicalBackupPossible bool
		isZstdDumpPossible       bool
	}{
		{"compresses with gzip and backs up only logically on 15", Version15, 15, false, false},
		{"compresses with zstd and backs up only logically on 16", Version16, 16, false, true},
		{"compresses with zstd and backs up physically on 17", Version17, 17, true, true},
		{"compresses with zstd and backs up physically on 18", Version18, 18, true, true},
		{"compresses with zstd and backs up physically on 19", Version19, 19, true, true},
		{"allows nothing when the version is not a number", Version("unknown"), 0, false, false},
	} {
		t.Run(versionCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, versionCase.expectedNumber, versionCase.version.GetNumber())
			assert.Equal(t, versionCase.isPhysicalBackupPossible, versionCase.version.CanBackUpPhysically())
			assert.Equal(t, versionCase.isZstdDumpPossible, versionCase.version.CanDumpWithZstd())
		})
	}
}

func TestFindSupportedVersion(t *testing.T) {
	t.Parallel()

	t.Run("finds every bundled major", func(t *testing.T) {
		t.Parallel()

		for _, supportedVersion := range getSupportedVersions() {
			foundVersion, isFound := FindSupportedVersion(string(supportedVersion))

			assert.True(t, isFound)
			assert.Equal(t, supportedVersion, foundVersion)
		}
	})

	t.Run("finds nothing for a major without a bundle", func(t *testing.T) {
		t.Parallel()

		for _, major := range []string{"11", "20"} {
			_, isFound := FindSupportedVersion(major)

			assert.False(t, isFound, major)
		}
	})
}

func TestVersionRanges(t *testing.T) {
	t.Parallel()

	t.Run("names the logical range from the oldest to the newest bundled major", func(t *testing.T) {
		t.Parallel()

		newestVersion := getSupportedVersions()[len(getSupportedVersions())-1]

		assert.Equal(t, "PostgreSQL 12 to "+string(newestVersion), GetSupportedVersionRange())
	})

	t.Run("names the physical range from 17 to the newest bundled major", func(t *testing.T) {
		t.Parallel()

		newestVersion := getSupportedVersions()[len(getSupportedVersions())-1]

		assert.Equal(t, "PostgreSQL 17 to "+string(newestVersion), GetPhysicalBackupVersionRange())
	})
}

func TestIsBackupVersionHigherThanRestoreVersion(t *testing.T) {
	t.Parallel()

	for _, versionCase := range []struct {
		name           string
		backupVersion  Version
		restoreVersion Version
		isDowngrade    bool
	}{
		{"refuses a backup of 18 onto 17", Version18, Version17, true},
		{"refuses a backup of 19 onto 18", Version19, Version18, true},
		{"allows a backup of 17 onto 18", Version17, Version18, false},
		{"allows a backup onto the same major", Version18, Version18, false},
		{"allows a restore when a version is not a number", Version("unknown"), Version17, false},
	} {
		t.Run(versionCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(
				t,
				versionCase.isDowngrade,
				IsBackupVersionHigherThanRestoreVersion(versionCase.backupVersion, versionCase.restoreVersion),
			)
		})
	}
}
