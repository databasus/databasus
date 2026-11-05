package mailer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSender(t *testing.T) {
	t.Parallel()

	t.Run("parses supported forms and rejects unsupported ones", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			value           string
			expectedName    string
			expectedAddress string
			isRejected      bool
		}{
			{"noreply@example.com", "Databasus", "noreply@example.com", false},
			{"Acme Backups <noreply@example.com>", "Acme Backups", "noreply@example.com", false},
			{`"Acme, Inc." <noreply@example.com>`, "Acme, Inc.", "noreply@example.com", false},
			{"ops at example", "", "", true},
			{"noreply at example.com", "", "", true},
			{"a@example.com, b@example.com", "", "", true},
			{"", "", "", true},
		}

		for _, testCase := range testCases {
			t.Run(testCase.value, func(t *testing.T) {
				sender, err := ParseSender(testCase.value)

				if testCase.isRejected {
					require.ErrorIs(t, err, ErrInvalidSender)
					return
				}

				require.NoError(t, err)
				assert.Equal(t, testCase.expectedName, sender.Name)
				assert.Equal(t, testCase.expectedAddress, sender.Address)
			})
		}
	})
}

func TestResolveSender(t *testing.T) {
	t.Parallel()

	t.Run("falls back from the sender to an address username to the host", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name            string
			from            string
			username        string
			expectedAddress string
		}{
			{"explicit sender wins", "Ops <ops@example.com>", "mailer@example.com", "ops@example.com"},
			{"username that is an address", "", "mailer@example.com", "mailer@example.com"},
			{"username apikey", "", "apikey", "noreply@smtp.sendgrid.net"},
			{"access key ID username", "", "AKIAIOSFODNN7EXAMPLE", "noreply@smtp.sendgrid.net"},
			{"username with a display name", "", "Mailer <mailer@example.com>", "noreply@smtp.sendgrid.net"},
			{"no username", "", "", "noreply@smtp.sendgrid.net"},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				sender, err := ResolveSender(testCase.from, testCase.username, "smtp.sendgrid.net")

				require.NoError(t, err)
				assert.Equal(t, testCase.expectedAddress, sender.Address)
			})
		}
	})

	t.Run("fails with an unreadable explicit sender", func(t *testing.T) {
		t.Parallel()
		_, err := ResolveSender("ops at example", "mailer@example.com", "smtp.example.com")

		require.ErrorIs(t, err, ErrInvalidSender)
	})
}

func TestParseRecipient(t *testing.T) {
	t.Parallel()

	t.Run("accepts only a bare address", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			value      string
			isRejected bool
		}{
			{"admin@example.com", false},
			{"Admin <admin@example.com>", true},
			{"ops at example", true},
			{"a@example.com, b@example.com", true},
			{"admin@example.com\r\nBcc: attacker@example.com", true},
		}

		for _, testCase := range testCases {
			t.Run(testCase.value, func(t *testing.T) {
				_, err := ParseRecipient(testCase.value)

				if testCase.isRejected {
					require.ErrorIs(t, err, ErrInvalidRecipient)
					return
				}

				require.NoError(t, err)
			})
		}
	})
}
