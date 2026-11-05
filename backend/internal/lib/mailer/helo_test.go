package mailer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateHeloName(t *testing.T) {
	t.Parallel()

	t.Run("accepts host names and address literals and rejects everything else", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name       string
			isRejected bool
		}{
			{"mail.example.com", false},
			{"localhost", false},
			{"relay-01", false},
			{"[192.0.2.1]", false},
			{"[IPv6:2001:db8::1]", false},
			{"my relay", true},
			{"[::1]", true},
			{"[IPv6:192.0.2.1]", true},
			{"[999.0.2.1]", true},
			{"-relay.example.com", true},
			{"relay_01.example.com", true},
			{"mail.example.com.", true},
			{"mail.example.com\r\nRCPT TO:<x@example.com>", true},
			{"", true},
			{strings.Repeat("a.", 128) + "a", true},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				err := ValidateHeloName(testCase.name)

				if testCase.isRejected {
					assert.ErrorIs(t, err, ErrInvalidHeloName)
					return
				}

				assert.NoError(t, err)
			})
		}
	})
}

func TestGetDefaultHeloName(t *testing.T) {
	t.Parallel()

	t.Run("returns the host of the public address", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			databasusURL     string
			expectedHeloName string
		}{
			{"https://backup.example.com", "backup.example.com"},
			{"https://backup.example.com:4005/path", "backup.example.com"},
			{"http://192.0.2.1:4005", "[192.0.2.1]"},
			{"http://[2001:db8::1]:4005", "[IPv6:2001:db8::1]"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.databasusURL, func(t *testing.T) {
				assert.Equal(t, testCase.expectedHeloName, GetDefaultHeloName(testCase.databasusURL))
			})
		}
	})

	t.Run("returns a valid name without a usable public address", func(t *testing.T) {
		t.Parallel()
		for _, databasusURL := range []string{"", "not a url with spaces", "https://under_score.example.com"} {
			t.Run(databasusURL, func(t *testing.T) {
				assert.NoError(t, ValidateHeloName(GetDefaultHeloName(databasusURL)))
			})
		}
	})
}
