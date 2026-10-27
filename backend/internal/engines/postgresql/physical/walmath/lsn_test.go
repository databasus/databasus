package walmath_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/engines/postgresql/physical/walmath"
)

func TestParseLSNHex(t *testing.T) {
	t.Parallel()
	t.Run("round trips valid hex values through their string form", func(t *testing.T) {
		t.Parallel()

		cases := []string{
			"0/0",
			"0/1",
			"1/0",
			"0/2A33FF50",
			"12345678/90ABCDEF",
			"FFFFFFFF/FFFFFFFF",
		}

		for _, encodedLSN := range cases {
			lsn, err := walmath.ParseLSN(encodedLSN)
			require.NoError(t, err, encodedLSN)
			assert.Equal(t, encodedLSN, lsn.String(), encodedLSN)
		}
	})
}

func TestParseLSNMalformed(t *testing.T) {
	t.Parallel()
	t.Run("returns an error with malformed input", func(t *testing.T) {
		t.Parallel()

		cases := []string{
			"",
			"/0",
			"0/",
			"GGG/0",
			"deadbeef",
			"0",
		}

		for _, encodedLSN := range cases {
			_, err := walmath.ParseLSN(encodedLSN)
			require.Error(t, err, encodedLSN)
		}
	})
}

func TestLSNString(t *testing.T) {
	t.Parallel()
	t.Run("formats canonical hex with no zero padding", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "12345678/90ABCDEF", walmath.LSN(0x1234567890ABCDEF).String())
		assert.Equal(t, "0/A", walmath.LSN(0xA).String())
		assert.Equal(t, "1/0", walmath.LSN(0x100000000).String())
	})
}

func TestParseLSNLowercase(t *testing.T) {
	t.Parallel()
	t.Run("parses correctly with lowercase hex", func(t *testing.T) {
		t.Parallel()

		lsn, err := walmath.ParseLSN("0/2a33ff50")
		require.NoError(t, err)

		assert.Equal(t, walmath.LSN(0x2A33FF50), lsn)
	})
}
