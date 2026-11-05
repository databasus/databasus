package mailer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSend(t *testing.T) {
	t.Parallel()

	t.Run("delivers over an encrypted connection with tls", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{
			implicitTLSConfig: newSelfSignedTLSConfig(t),
			authMechanisms:    []string{mechanismPlain},
		})

		err := Send(t.Context(), server.getConnection(SecurityTLS), newTestMessage(t))
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		assert.Len(t, server.getMessages(), 1)
		assert.Equal(t, "EHLO backup.example.com", server.getCommands()[0])
	})

	t.Run("upgrades before any other command with starttls", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{
			startTLSConfig: newSelfSignedTLSConfig(t),
			authMechanisms: []string{mechanismPlain},
		})

		err := Send(t.Context(), server.getConnection(SecurityStartTLS), newTestMessage(t))
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		assert.Equal(
			t,
			[]string{"EHLO", "STARTTLS", "EHLO", "AUTH", "MAIL", "RCPT", "DATA", "QUIT"},
			server.getCommandVerbs(),
		)
		assert.Equal(t, "EHLO backup.example.com", server.getCommands()[2])
		assert.Len(t, server.getMessages(), 1)
	})

	t.Run("fails with nothing sent after the greeting when the server does not offer STARTTLS", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{
			authMechanisms: []string{mechanismPlain, mechanismLogin},
		})

		err := Send(t.Context(), server.getConnection(SecurityStartTLS), newTestMessage(t))
		require.ErrorIs(t, err, ErrStartTLSNotOffered)

		server.waitForSessionEnd(t)
		assert.Equal(t, []string{"EHLO backup.example.com"}, server.getCommands())
		assert.Empty(t, server.getMessages())
	})

	t.Run("never upgrades and authenticates with none", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{
			startTLSConfig: newSelfSignedTLSConfig(t),
			authMechanisms: []string{mechanismPlain},
		})

		err := Send(t.Context(), server.getConnection(SecurityNone), newTestMessage(t))
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		assert.Equal(t, []string{"EHLO", "AUTH", "MAIL", "RCPT", "DATA", "QUIT"}, server.getCommandVerbs())
		assert.Len(t, server.getMessages(), 1)
	})

	t.Run("fails with an untrusted certificate unless verification is off", func(t *testing.T) {
		t.Parallel()
		testCases := []struct {
			name     string
			security Security
			options  fakeServerOptions
		}{
			{"tls", SecurityTLS, fakeServerOptions{implicitTLSConfig: newSelfSignedTLSConfig(t)}},
			{"starttls", SecurityStartTLS, fakeServerOptions{startTLSConfig: newSelfSignedTLSConfig(t)}},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				testCase.options.authMechanisms = []string{mechanismPlain}
				server := startFakeServer(t, testCase.options)

				verifyingConnection := server.getConnection(testCase.security)
				verifyingConnection.IsInsecureSkipVerify = false

				err := Send(t.Context(), verifyingConnection, newTestMessage(t))
				require.Error(t, err)
				assert.Contains(t, err.Error(), "certificate")

				server.waitForSessionEnd(t)
				assert.NotContains(t, server.getCommandVerbs(), "AUTH")

				err = Send(t.Context(), server.getConnection(testCase.security), newTestMessage(t))
				require.NoError(t, err)
			})
		}
	})

	t.Run(
		"fails with a timeout within the deadline when the server goes silent after the greeting",
		func(t *testing.T) {
			t.Parallel()
			server := startFakeServer(t, fakeServerOptions{isSilentAfterGreeting: true})

			callerCtx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
			defer cancel()

			startedAt := time.Now().UTC()
			err := Send(callerCtx, server.getConnection(SecurityNone), newTestMessage(t))

			require.ErrorIs(t, err, ErrSessionTimeout)
			assert.Less(t, time.Since(startedAt), 2*time.Second)
		},
	)

	t.Run("greets with the fallback name without a greeting name", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanismPlain}})

		connection := server.getConnection(SecurityNone)
		connection.HeloName = ""

		err := Send(t.Context(), connection, newTestMessage(t))
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		assert.Equal(t, "EHLO "+fallbackHeloName, server.getCommands()[0])
	})

	t.Run("fails with an unknown security mode", func(t *testing.T) {
		t.Parallel()
		err := Send(t.Context(), Connection{Host: "127.0.0.1", Port: 25, Security: "ssl"}, newTestMessage(t))

		require.ErrorIs(t, err, ErrUnknownSecurity)
	})
}

func newTestMessage(t *testing.T) Message {
	t.Helper()

	sender, err := ParseSender("noreply@example.com")
	require.NoError(t, err)

	return Message{
		Sender:    sender,
		Recipient: "admin@example.com",
		Subject:   "Test",
		HTMLBody:  "<p>Hello</p>",
	}
}
