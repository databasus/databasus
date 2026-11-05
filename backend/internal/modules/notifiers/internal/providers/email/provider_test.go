package email_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

		message := newNotification()
		message.Message = "Backup error: <a href=\"https://attacker.invalid\">inspect</a> & retry\nNext line"
		err := newProvider(t).Send(t.Context(), newSettings(mailpitEndpoint, mailer.SecurityNone), message)
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

		request, err := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			fmt.Sprintf(
				"http://%s:%d/api/v1/message/%s",
				mailpitEndpoint.HTTP.Host,
				mailpitEndpoint.HTTP.Port,
				deliveredMessages[0].ID,
			),
			nil,
		)
		require.NoError(t, err)
		response, err := http.DefaultClient.Do(request)
		require.NoError(t, err)
		defer func() { require.NoError(t, response.Body.Close()) }()
		require.Equal(t, http.StatusOK, response.StatusCode)

		var deliveredContent struct {
			HTML string `json:"HTML"`
		}
		require.NoError(t, json.NewDecoder(response.Body).Decode(&deliveredContent))
		assert.Contains(
			t,
			deliveredContent.HTML,
			"&lt;a href=&#34;https://attacker.invalid&#34;&gt;inspect&lt;/a&gt; &amp; retry<br>Next line",
		)
		assert.NotContains(t, deliveredContent.HTML, "<a href=")
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
