package email

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/config"
	lib_config "databasus-backend/internal/lib/config"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/testing/mailpit"
)

func Test_SendEmail_WhenSmtpServerAccepts_DeliversMessageWithSenderNameAndMessageID(t *testing.T) {
	mailpitEndpoint := containers.StartMailpit(t)
	mailpitClient := mailpit.NewClient(
		fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port),
	)

	sender := newEmailSMTPSenderFromEnv(&config.EnvVariables{Config: lib_config.Config{
		SMTPHost:     mailpitEndpoint.SMTP.Host,
		SMTPPort:     mailpitEndpoint.SMTP.Port,
		SMTPFrom:     "sender@databasus.local",
		SMTPSecurity: "none",
	}})

	err := sender.SendEmail(t.Context(), "recipient@databasus.local", "Password Reset Code", "<b>123456</b>")
	require.NoError(t, err)

	var delivered []mailpit.Message
	require.Eventually(t, func() bool {
		messages, fetchErr := mailpitClient.FetchMessages()
		if fetchErr != nil {
			return false
		}

		delivered = messages

		return len(messages) == 1
	}, 5*time.Second, 100*time.Millisecond, "Mailpit should receive exactly one message")

	assert.Equal(t, "Password Reset Code", delivered[0].Subject)
	assert.Equal(t, "Databasus", delivered[0].From.Name)
	assert.Equal(t, "sender@databasus.local", delivered[0].From.Address)
	assert.NotEmpty(t, delivered[0].MessageID)
	require.Len(t, delivered[0].To, 1)
	assert.Equal(t, "recipient@databasus.local", delivered[0].To[0].Address)
}

func Test_SendEmail_WhenSmtpHostIsUnset_SkipsWithoutError(t *testing.T) {
	sender := newEmailSMTPSenderFromEnv(&config.EnvVariables{})

	assert.False(t, sender.IsConfigured())
	assert.NoError(t, sender.SendEmail(t.Context(), "recipient@databasus.local", "Subject", "<p>body</p>"))
}

func Test_NewEmailSMTPSenderFromEnv_WithoutSecurity_UsesPortDefault(t *testing.T) {
	implicitTLSSender := newEmailSMTPSenderFromEnv(
		&config.EnvVariables{Config: lib_config.Config{SMTPHost: "smtp.example.com", SMTPPort: 465}},
	)
	upgradeSender := newEmailSMTPSenderFromEnv(
		&config.EnvVariables{Config: lib_config.Config{SMTPHost: "smtp.example.com", SMTPPort: 587}},
	)

	assert.True(t, implicitTLSSender.IsConfigured())
	assert.Equal(t, "tls", string(implicitTLSSender.connection.Security))
	assert.Equal(t, "starttls", string(upgradeSender.connection.Security))
}

func Test_NewEmailSMTPSenderFromEnv_ForRelayWithoutCredentials_UsesStartTLSWithoutAuth(t *testing.T) {
	relaySender := newEmailSMTPSenderFromEnv(
		&config.EnvVariables{Config: lib_config.Config{SMTPHost: "relay.example.com", SMTPPort: 25}},
	)

	assert.True(t, relaySender.IsConfigured())
	assert.Equal(t, "starttls", string(relaySender.connection.Security))
	assert.Empty(t, relaySender.connection.Username)
	assert.Empty(t, relaySender.connection.Password)
}
