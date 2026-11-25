package totp_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/totp"
)

// rfcSecret is the base32 form of the ASCII seed "12345678901234567890" of the RFC 6238 SHA-1
// test vectors.
const rfcSecret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

func TestComputeCode(t *testing.T) {
	t.Parallel()

	rfcVectors := []struct {
		name         string
		unixSeconds  int64
		expectedCode string
	}{
		{name: "matches the rfc vector at 59 seconds", unixSeconds: 59, expectedCode: "287082"},
		{name: "matches the rfc vector at 1111111109", unixSeconds: 1111111109, expectedCode: "081804"},
		{name: "matches the rfc vector at 1111111111", unixSeconds: 1111111111, expectedCode: "050471"},
		{name: "matches the rfc vector at 1234567890", unixSeconds: 1234567890, expectedCode: "005924"},
		{name: "matches the rfc vector at 2000000000", unixSeconds: 2000000000, expectedCode: "279037"},
		{name: "matches the rfc vector at 20000000000", unixSeconds: 20000000000, expectedCode: "353130"},
	}

	for _, vector := range rfcVectors {
		t.Run(vector.name, func(t *testing.T) {
			t.Parallel()

			computedCode, err := totp.ComputeCode(rfcSecret, totp.GetStep(time.Unix(vector.unixSeconds, 0)))

			require.NoError(t, err)
			assert.Equal(t, vector.expectedCode, computedCode)
		})
	}

	t.Run("refuses a secret that is not base32", func(t *testing.T) {
		t.Parallel()

		_, err := totp.ComputeCode("not base32!", 1)

		require.ErrorIs(t, err, totp.ErrSecretMalformed)
	})
}

func TestFindMatchingStep(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_700_000_015, 0)
	currentStep := totp.GetStep(now)

	t.Run("accepts the code of the previous, current and next step", func(t *testing.T) {
		t.Parallel()

		for _, step := range []int64{currentStep - 1, currentStep, currentStep + 1} {
			code, err := totp.ComputeCode(rfcSecret, step)
			require.NoError(t, err)

			matchingStep, isMatched, err := totp.FindMatchingStep(rfcSecret, code, now)

			require.NoError(t, err)
			assert.True(t, isMatched)
			assert.Equal(t, step, matchingStep)
		}
	})

	t.Run("refuses the code of a step two periods away", func(t *testing.T) {
		t.Parallel()

		for _, step := range []int64{currentStep - 2, currentStep + 2, currentStep - 4} {
			code, err := totp.ComputeCode(rfcSecret, step)
			require.NoError(t, err)

			_, isMatched, err := totp.FindMatchingStep(rfcSecret, code, now)

			require.NoError(t, err)
			assert.False(t, isMatched)
		}
	})

	t.Run("refuses a code of another secret", func(t *testing.T) {
		t.Parallel()
		code, err := totp.ComputeCode("JBSWY3DPEHPK3PXP", currentStep)
		require.NoError(t, err)

		_, isMatched, err := totp.FindMatchingStep(rfcSecret, code, now)

		require.NoError(t, err)
		assert.False(t, isMatched)
	})
}

func TestIsCodeFormat(t *testing.T) {
	t.Parallel()

	formats := []struct {
		name         string
		value        string
		isCodeFormat bool
	}{
		{name: "accepts six digits", value: "012345", isCodeFormat: true},
		{name: "refuses five digits", value: "12345", isCodeFormat: false},
		{name: "refuses seven digits", value: "1234567", isCodeFormat: false},
		{name: "refuses letters", value: "12a456", isCodeFormat: false},
		{name: "refuses a recovery code", value: "abcde-fghij", isCodeFormat: false},
	}

	for _, format := range formats {
		t.Run(format.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, format.isCodeFormat, totp.IsCodeFormat(format.value))
		})
	}
}
