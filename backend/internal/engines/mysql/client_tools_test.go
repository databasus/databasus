package mysql

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/clienttools"
)

func TestGetClientToolBundles(t *testing.T) {
	t.Parallel()

	for _, architecture := range []struct {
		name     string
		versions []string
	}{
		{"x64", []string{"5.7", "8.0", "8.4", "9", "26"}},
		{"arm", []string{"8.0", "8.4", "9", "26"}},
	} {
		t.Run("reports missing shipped bundles when architecture is "+architecture.name, func(t *testing.T) {
			t.Parallel()
			bundles := GetClientToolBundles(filepath.Join(t.TempDir(), architecture.name))
			versions := make([]string, 0, len(bundles))
			for _, bundle := range bundles {
				versions = append(versions, bundle.Version)
			}

			assert.Equal(t, architecture.versions, versions)
			for _, bundleCheck := range clienttools.NewChecker(bundles).Check(t.Context()) {
				assert.NotEmpty(
					t,
					bundleCheck.Errors,
					"missing MySQL %s must fail its check",
					bundleCheck.Bundle.Version,
				)
			}
		})
	}
}

func TestParseClientVersion(t *testing.T) {
	t.Parallel()

	t.Run("reads the release from both output shapes", func(t *testing.T) {
		t.Parallel()
		outputCases := []struct {
			output, expectedVersion string
		}{
			{"mysqldump  Ver 26.7.0 for Linux on x86_64 (MySQL Community Server - GPL)", "26.7.0"},
			{"mysql  Ver 9.7.2 for Linux on x86_64 (MySQL Community Server - GPL)", "9.7.2"},
			// 5.7 reports the tool's own version first and the server line behind "Distrib", so the
			// first number in the line is the wrong one.
			{"mysqldump  Ver 10.13 Distrib 5.7.44, for linux-glibc2.12 (x86_64)", "5.7.44"},
			{"mysql  Ver 14.14 Distrib 5.7.44, for linux-glibc2.12 (x86_64) using  EditLine wrapper", "5.7.44"},
		}

		for _, outputCase := range outputCases {
			reportedVersion, err := parseClientVersion(outputCase.output)

			require.NoError(t, err)
			assert.Equal(t, outputCase.expectedVersion, reportedVersion)
		}
	})

	t.Run("returns an error when the output carries no version", func(t *testing.T) {
		t.Parallel()

		_, err := parseClientVersion("mysqldump: command not found")

		require.ErrorIs(t, err, ErrUnreadableClientVersion)
	})
}

func TestIsBackupVersionHigherThanRestoreVersion(t *testing.T) {
	t.Parallel()

	t.Run("refuses downgrades across the calendar line", func(t *testing.T) {
		t.Parallel()
		restoreCases := []struct {
			backupVersion, restoreVersion Version
			isDowngrade                   bool
		}{
			{Version26, Version9, true},
			{Version26, Version80, true},
			{Version26, Version26, false},
			{Version9, Version26, false},
			{Version84, Version80, true},
			{Version57, Version80, false},
		}

		for _, restoreCase := range restoreCases {
			isDowngrade, err := IsBackupVersionHigherThanRestoreVersion(
				restoreCase.backupVersion,
				restoreCase.restoreVersion,
			)

			require.NoError(t, err)
			assert.Equalf(t, restoreCase.isDowngrade, isDowngrade,
				"%s onto %s", restoreCase.backupVersion, restoreCase.restoreVersion)
		}
	})

	t.Run("returns an error with a version it cannot order", func(t *testing.T) {
		t.Parallel()

		_, err := IsBackupVersionHigherThanRestoreVersion("", Version9)

		require.Error(t, err)
	})
}
