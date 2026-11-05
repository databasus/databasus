package mailer

import (
	"io"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendMessage(t *testing.T) {
	t.Parallel()

	t.Run("adds no header and no recipient with line breaks in the subject", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanismPlain}})

		message := newTestMessage(t)
		message.Subject = "Ops\r\nBcc: attacker@example.com"

		err := Send(t.Context(), server.getConnection(SecurityNone), message)
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		receivedMessage := readReceivedMessage(t, server)

		assert.Empty(t, receivedMessage.Header.Get("Bcc"))
		assert.Equal(t, 1, strings.Count(strings.Join(server.getCommandVerbs(), " "), "RCPT"))
	})

	t.Run("keeps the original name with a non-ASCII display name", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanismPlain}})

		sender, err := ParseSender("Резервные копии <ops@example.com>")
		require.NoError(t, err)

		message := newTestMessage(t)
		message.Sender = sender

		err = Send(t.Context(), server.getConnection(SecurityNone), message)
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		receivedFrom, err := readReceivedMessage(t, server).Header.AddressList("From")
		require.NoError(t, err)

		assert.Equal(t, "Резервные копии", receivedFrom[0].Name)
		assert.Equal(t, "ops@example.com", receivedFrom[0].Address)
		assert.Contains(t, server.getCommands(), "MAIL FROM:<ops@example.com>")
	})

	t.Run("delivers a long body line unchanged within the wire line limit", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanismPlain}})

		longLine := strings.Repeat("<span>é</span>", 400)
		require.Greater(t, len(longLine), 5000)

		message := newTestMessage(t)
		message.HTMLBody = longLine

		err := Send(t.Context(), server.getConnection(SecurityNone), message)
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		rawMessage := server.getMessages()[0]
		for wireLine := range strings.SplitSeq(rawMessage, "\r\n") {
			assert.LessOrEqual(t, len(wireLine), 998)
		}

		receivedMessage := readReceivedMessage(t, server)
		assert.Equal(t, "quoted-printable", receivedMessage.Header.Get("Content-Transfer-Encoding"))

		decodedBody, err := io.ReadAll(quotedprintable.NewReader(receivedMessage.Body))
		require.NoError(t, err)
		assert.Equal(t, longLine, strings.TrimSuffix(string(decodedBody), "\r\n"))
	})

	t.Run("gives two messages different message IDs on the sender domain", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanismPlain}})

		var messageIDs []string
		for range 2 {
			err := Send(t.Context(), server.getConnection(SecurityNone), newTestMessage(t))
			require.NoError(t, err)

			server.waitForSessionEnd(t)
		}

		for _, rawMessage := range server.getMessages() {
			receivedMessage, err := mail.ReadMessage(strings.NewReader(rawMessage))
			require.NoError(t, err)

			messageIDs = append(messageIDs, receivedMessage.Header.Get("Message-ID"))
			assert.Equal(t, "1.0", receivedMessage.Header.Get("MIME-Version"))
			assert.Equal(t, `text/html; charset="UTF-8"`, receivedMessage.Header.Get("Content-Type"))
		}

		require.Len(t, messageIDs, 2)
		assert.NotEqual(t, messageIDs[0], messageIDs[1])
		assert.True(t, strings.HasSuffix(messageIDs[0], "@example.com>"))
	})
}

func readReceivedMessage(t *testing.T, server *fakeServer) *mail.Message {
	t.Helper()

	receivedMessages := server.getMessages()
	require.Len(t, receivedMessages, 1)

	receivedMessage, err := mail.ReadMessage(strings.NewReader(receivedMessages[0]))
	require.NoError(t, err)

	return receivedMessage
}
