package clienttools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareReleaseLines(t *testing.T) {
	t.Parallel()

	t.Run("orders identities of different depth by their numbers", func(t *testing.T) {
		t.Parallel()
		orderCases := []struct {
			left, right string
			isLeftNewer bool
		}{
			{"26", "9", true},
			{"9", "8.4", true},
			{"8.4", "8.0", true},
			{"13.0", "12.0", true},
			{"12.0", "11.8", true},
			{"10.11", "10.6", true},
			{"5.7", "8.0", false},
			{"10.6", "10.11", false},
		}

		for _, orderCase := range orderCases {
			order, err := CompareReleaseLines(orderCase.left, orderCase.right)

			require.NoError(t, err)
			assert.Equalf(t, orderCase.isLeftNewer, order > 0, "%s against %s", orderCase.left, orderCase.right)
		}
	})

	t.Run("reports no difference with equal identities", func(t *testing.T) {
		t.Parallel()

		order, err := CompareReleaseLines("13.0", "13.0")

		require.NoError(t, err)
		assert.Zero(t, order)
	})

	t.Run("returns an error with an identity that is not a dotted number", func(t *testing.T) {
		t.Parallel()

		_, invalidErr := CompareReleaseLines("13.0", "next")
		_, emptyErr := CompareReleaseLines("", "13.0")

		require.ErrorIs(t, invalidErr, ErrInvalidVersionIdentity)
		require.ErrorIs(t, emptyErr, ErrEmptyVersionIdentity)
	})
}

func TestIsVersionOnReleaseLine(t *testing.T) {
	t.Parallel()

	t.Run("accepts a later patch of the line and rejects another line", func(t *testing.T) {
		t.Parallel()
		lineCases := []struct {
			reported, line string
			isOnLine       bool
		}{
			{"26.7.0", "26", true},
			{"26.10.1", "26", true},
			{"13.0.2", "13.0", true},
			{"13.2.0", "13.0", false},
			{"10.6.28", "10.6", true},
			{"10.11.14", "10.6", false},
			{"12.3.3", "12.3", true},
			{"9.7.2", "9", true},
			{"5.7.44", "5.7", true},
			{"17.10", "17", true},
			{"10.6", "10.6.28", false},
			{"not-a-version", "10.6", false},
		}

		for _, lineCase := range lineCases {
			assert.Equalf(
				t,
				lineCase.isOnLine,
				isVersionOnReleaseLine(lineCase.reported, lineCase.line),
				"version %s on line %s", lineCase.reported, lineCase.line,
			)
		}
	})
}
