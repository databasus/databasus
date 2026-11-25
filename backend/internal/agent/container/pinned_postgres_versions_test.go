package container

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPinnedPostgresVersions(t *testing.T) {
	t.Parallel()

	t.Run("pins a release of its own major for every major from 12 to 18", func(t *testing.T) {
		t.Parallel()

		pinnedVersions, err := LoadPinnedPostgresVersions()
		require.NoError(t, err)

		expectedMajors := make([]string, 0, 7)
		for major := 12; major <= 18; major++ {
			expectedMajors = append(expectedMajors, strconv.Itoa(major))
		}

		assert.ElementsMatch(t, expectedMajors, pinnedVersions.GetMajors())
	})
}

func TestParsePinnedPostgresVersions(t *testing.T) {
	t.Parallel()

	t.Run("reads the version of each major", func(t *testing.T) {
		t.Parallel()

		pinnedVersions, err := parsePinnedPostgresVersions([]byte(`{"17": "17.12", "18": "18.0"}`))
		require.NoError(t, err)

		pinnedVersion, isPinned := pinnedVersions.GetPinnedPostgresVersion("18")
		assert.True(t, isPinned)
		assert.Equal(t, "18.0", pinnedVersion)
		assert.True(t, pinnedVersions.IsPinnedPostgresVersion("17.12"))
		assert.False(t, pinnedVersions.IsPinnedPostgresVersion("17"))

		_, isPinned = pinnedVersions.GetPinnedPostgresVersion("16")
		assert.False(t, isPinned)
	})

	t.Run("rejects the list when it is broken", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			document string
		}{
			{name: "not json", document: `{"18": `},
			{name: "empty list", document: `{}`},
			{name: "version of another major", document: `{"18": "17.12"}`},
			{name: "major alone", document: `{"18": "18"}`},
			{name: "minor with a leading zero", document: `{"18": "18.07"}`},
			{name: "patch level", document: `{"18": "18.7.1"}`},
			{name: "major that is not a number", document: `{"latest": "latest.1"}`},
			{name: "major zero", document: `{"0": "0.1"}`},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				_, err := parsePinnedPostgresVersions([]byte(testCase.document))

				assert.Error(t, err)
			})
		}
	})
}

func newTestPinnedVersions(t *testing.T) PinnedPostgresVersions {
	t.Helper()

	pinnedVersions, err := parsePinnedPostgresVersions([]byte(`{"17": "17.12", "18": "18.7"}`))
	require.NoError(t, err)

	return pinnedVersions
}
