package mongodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBackupVersionHigherThanRestoreVersion(t *testing.T) {
	t.Parallel()

	t.Run("refuses downgrades across supported versions", func(t *testing.T) {
		t.Parallel()
		restoreCases := []struct {
			backupVersion, restoreVersion Version
			isDowngrade                   bool
		}{
			{Version9, Version8, true},
			{Version8, Version9, false},
			{Version9, Version9, false},
			{Version4, Version9, false},
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

func TestMapVersion(t *testing.T) {
	t.Parallel()

	supportedCases := []struct {
		name            string
		reportedVersion string
		mappedVersion   Version
	}{
		{"maps 4.2 to 4", "4.2.0", Version4},
		{"maps 4.4 to 4", "4.4.29", Version4},
		{"maps 5.0 to 5", "5.0.0", Version5},
		{"maps 6.0 to 6", "6.0.0", Version6},
		{"maps 7.0 to 7", "7.0.0", Version7},
		{"maps 8.0 to 8", "8.0.0", Version8},
		{"maps 8.10 to 8", "8.10.1", Version8},
		{"maps 9.0 to 9", "9.0.2", Version9},
		{"maps 9.2 to 9", "9.2.0", Version9},
	}

	for _, supportedCase := range supportedCases {
		t.Run(supportedCase.name, func(t *testing.T) {
			t.Parallel()

			mappedVersion, err := mapVersion(supportedCase.reportedVersion)

			require.NoError(t, err)
			assert.Equal(t, supportedCase.mappedVersion, mappedVersion)
		})
	}

	refusedCases := []struct {
		name            string
		reportedVersion string
		refusal         error
		errorFragments  []string
	}{
		{"refuses 3.6 as too old", "3.6.23", ErrUnsupportedVersion, []string{"3.6", "minimum supported: 4.2"}},
		{"refuses 4.0 as too old", "4.0.28", ErrUnsupportedVersion, []string{"4.0", "minimum supported: 4.2"}},
		{"refuses 4.1 as too old", "4.1.0", ErrUnsupportedVersion, []string{"4.1", "minimum supported: 4.2"}},
		{"refuses 10.0 as too new", "10.0.0", ErrUnsupportedVersion, []string{"10.0", "supported: 4.2 through 9.x"}},
		{
			"refuses an unreadable version",
			"abc",
			ErrUnreadableVersion,
			[]string{"could not parse MongoDB version", "abc"},
		},
		{"refuses an empty version", "", ErrUnreadableVersion, []string{"could not parse MongoDB version", `""`}},
	}

	for _, refusedCase := range refusedCases {
		t.Run(refusedCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := mapVersion(refusedCase.reportedVersion)

			require.ErrorIs(t, err, refusedCase.refusal)

			for _, fragment := range refusedCase.errorFragments {
				assert.Contains(t, err.Error(), fragment)
			}
		})
	}
}
