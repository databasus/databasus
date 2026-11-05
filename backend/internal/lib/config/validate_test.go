package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSMTPSettings(t *testing.T) {
	t.Parallel()

	rejectedCases := []struct {
		name             string
		smtpConfig       Config
		expectedVariable string
	}{
		{
			name:             "rejects a host without a port",
			smtpConfig:       Config{SMTPHost: "smtp.example.com"},
			expectedVariable: "SMTP_PORT",
		},
		{
			name:             "rejects an unknown security mode",
			smtpConfig:       Config{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPSecurity: "ssl"},
			expectedVariable: "SMTP_SECURITY",
		},
		{
			name:             "rejects a greeting name with a space",
			smtpConfig:       Config{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPHeloName: "my relay"},
			expectedVariable: "SMTP_HELO_NAME",
		},
		{
			name:             "rejects an unreadable sender",
			smtpConfig:       Config{SMTPHost: "smtp.example.com", SMTPPort: 587, SMTPFrom: "noreply at example.com"},
			expectedVariable: "SMTP_FROM",
		},
	}

	for _, rejectedCase := range rejectedCases {
		t.Run(rejectedCase.name, func(t *testing.T) {
			t.Parallel()

			err := validateSMTPSettings(rejectedCase.smtpConfig)

			require.Error(t, err)
			assert.Contains(t, err.Error(), rejectedCase.expectedVariable)
		})
	}

	t.Run("accepts a sender with a display name", func(t *testing.T) {
		t.Parallel()

		err := validateSMTPSettings(Config{
			SMTPHost:     "smtp.example.com",
			SMTPPort:     587,
			SMTPFrom:     "Acme Backups <noreply@example.com>",
			SMTPSecurity: "none",
			SMTPHeloName: "[IPv6:2001:db8::1]",
		})

		require.NoError(t, err)
	})
}
