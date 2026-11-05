package mailer

import (
	"log/slog"
	"net/mail"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceSenderSendEmail(t *testing.T) {
	t.Parallel()

	t.Run("delivers the message with the sender name and a message ID when the server accepts", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{})
		instanceSender := NewInstanceSender(Deps{
			Settings: Settings{
				Host:     server.host,
				Port:     server.port,
				Security: SecurityNone,
				From:     "sender@databasus.local",
				HeloName: "backup.example.com",
			},
			Logger: slog.New(slog.DiscardHandler),
		})

		err := instanceSender.SendEmail(
			t.Context(), "recipient@databasus.local", "Password Reset Code", "<b>123456</b>",
		)
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		receivedMessage := readReceivedMessage(t, server)
		sender, err := mail.ParseAddress(receivedMessage.Header.Get("From"))
		require.NoError(t, err)
		assert.Equal(t, "Databasus", sender.Name)
		assert.Equal(t, "sender@databasus.local", sender.Address)
		assert.Equal(t, "Password Reset Code", receivedMessage.Header.Get("Subject"))
		assert.NotEmpty(t, receivedMessage.Header.Get("Message-ID"))
		recipients, err := receivedMessage.Header.AddressList("To")
		require.NoError(t, err)
		require.Len(t, recipients, 1)
		assert.Equal(t, "recipient@databasus.local", recipients[0].Address)
	})

	t.Run("skips the message without an error when no server is configured", func(t *testing.T) {
		t.Parallel()
		instanceSender := NewInstanceSender(Deps{Settings: Settings{}, Logger: slog.New(slog.DiscardHandler)})

		err := instanceSender.SendEmail(t.Context(), "recipient@databasus.local", "Subject", "<p>body</p>")

		require.NoError(t, err)
		assert.False(t, instanceSender.IsConfigured())
	})
}
