package email_notifier

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	notifier_models "databasus-backend/internal/features/notifiers/models"
	"databasus-backend/internal/lib/mailer"
	"databasus-backend/internal/util/encryption"
	"databasus-backend/internal/util/logger"
	"databasus-backend/internal/util/testing/containers"
	"databasus-backend/internal/util/testing/mailpit"
)

func Test_EmailNotifierSend_WithNoneAgainstMailpit_DeliversMessageToRecipient(t *testing.T) {
	mailpitEndpoint := containers.StartMailpit(t)
	mailpitClient := mailpit.NewClient(
		fmt.Sprintf("%s:%d", mailpitEndpoint.HTTP.Host, mailpitEndpoint.HTTP.Port),
	)

	notifier := newMailpitEmailNotifier(mailpitEndpoint, mailer.SecurityNone)

	err := notifier.Send(encryption.GetFieldEncryptor(), logger.GetLogger(), newBackupNotification())
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

	assert.Equal(t, "Backup completed", delivered[0].Subject)
	assert.Equal(t, "Databasus", delivered[0].From.Name)
	require.Len(t, delivered[0].To, 1)
	assert.Equal(t, "recipient@databasus.local", delivered[0].To[0].Address)
}

func Test_EmailNotifierSend_WithStartTLSAgainstMailpit_FailsBecauseUpgradeIsNotOffered(t *testing.T) {
	mailpitEndpoint := containers.StartMailpit(t)

	notifier := newMailpitEmailNotifier(mailpitEndpoint, mailer.SecurityStartTLS)

	err := notifier.Send(encryption.GetFieldEncryptor(), logger.GetLogger(), newBackupNotification())

	require.ErrorIs(t, err, mailer.ErrStartTLSNotOffered)
}

func newMailpitEmailNotifier(
	mailpitEndpoint containers.MailpitEndpoint,
	security mailer.Security,
) *EmailNotifier {
	return &EmailNotifier{
		NotifierID:  uuid.New(),
		TargetEmail: "recipient@databasus.local",
		SMTPHost:    mailpitEndpoint.SMTP.Host,
		SMTPPort:    mailpitEndpoint.SMTP.Port,
		From:        "sender@databasus.local",
		Security:    security,
	}
}

func newBackupNotification() notifier_models.Notification {
	return notifier_models.Notification{
		Type:    notifier_models.NotificationTypeAll,
		Heading: "Backup completed",
		Message: "<b>All good</b>",
	}
}
