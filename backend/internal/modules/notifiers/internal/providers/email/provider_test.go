package email_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/mailer"
	mailpit "databasus-backend/internal/lib/mailpittest"
	containers "databasus-backend/internal/lib/testcontainers"
	"databasus-backend/internal/lib/testwait"
	"databasus-backend/internal/modules/notifiers/internal/notification"
	"databasus-backend/internal/modules/notifiers/internal/providers/email"
	"databasus-backend/internal/modules/notifiers/internal/providers/providertest"
)

var errMessageNotArrived = errors.New("the message has not arrived")

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("delivers the message to the recipient without TLS", func(t *testing.T) {
		t.Parallel()
		mailpitEndpoint := containers.StartMailpit(t)
		mailpitClient := mailpit.NewClient(fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port))

		err := newProvider(t).Send(t.Context(), newSettings(mailpitEndpoint, mailer.SecurityNone), newNotification())
		require.NoError(t, err)

		var deliveredMessages []mailpit.Message

		testwait.Until(t, "the message arrives at Mailpit", func(context.Context) error {
			messages, err := mailpitClient.FetchMessages(t.Context())
			if err != nil {
				return err
			}

			if len(messages) != 1 {
				return errMessageNotArrived
			}

			deliveredMessages = messages

			return nil
		})

		assert.Equal(t, "Backup completed", deliveredMessages[0].Subject)
		assert.Equal(t, "Databasus", deliveredMessages[0].From.Name)
		require.Len(t, deliveredMessages[0].To, 1)
		assert.Equal(t, "recipient@databasus.local", deliveredMessages[0].To[0].Address)
	})

	t.Run("fails when STARTTLS is asked and the server does not offer it", func(t *testing.T) {
		t.Parallel()
		mailpitEndpoint := containers.StartMailpit(t)

		err := newProvider(
			t,
		).Send(t.Context(), newSettings(mailpitEndpoint, mailer.SecurityStartTLS), newNotification())

		require.ErrorIs(t, err, mailer.ErrStartTLSNotOffered)
	})
}

func TestValidate(t *testing.T) {
	t.Parallel()

	t.Run("refuses an SMTP password without a user", func(t *testing.T) {
		t.Parallel()
		settings := newSettings(containers.MailpitEndpoint{}, mailer.SecurityNone)
		settings.SMTPHost = "smtp.example.com"
		settings.SMTPPort = 587
		settings.SMTPPassword = "password"

		err := newProvider(t).Validate(settings)

		assert.NotEmpty(t, providertest.GetInvalidSettingDetail(t, err, "smtpPassword"))
	})
}

func newProvider(t *testing.T) *email.Provider {
	t.Helper()

	return email.New(email.Deps{
		Repositories:   email.Repositories{},
		FieldEncryptor: providertest.NewFieldEncryptor(t),
		PublicURL:      "",
	})
}

func newSettings(mailpitEndpoint containers.MailpitEndpoint, security mailer.Security) email.Settings {
	return email.Settings{
		TargetEmail:          "recipient@databasus.local",
		SMTPHost:             mailpitEndpoint.SMTP.Host,
		SMTPPort:             mailpitEndpoint.SMTP.Port,
		SMTPUser:             "",
		SMTPPassword:         "",
		From:                 "sender@databasus.local",
		IsInsecureSkipVerify: false,
		Security:             security,
		HeloName:             "",
	}
}

func newNotification() notification.Notification {
	return notification.Notification{
		Category: notification.CategoryAll,
		Heading:  "Backup completed",
		Message:  "<b>All good</b>",
	}
}
