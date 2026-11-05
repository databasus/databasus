package mailer

import (
	"net/smtp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendAuthentication(t *testing.T) {
	t.Parallel()

	t.Run("uses PLAIN when the server offers PLAIN and LOGIN", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanismLogin, mechanismPlain}})

		err := Send(t.Context(), server.getConnection(SecurityNone), newTestMessage(t))
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		authCommandIndex := slices.Index(server.getCommandVerbs(), "AUTH")
		require.GreaterOrEqual(t, authCommandIndex, 0)
		assert.True(t, strings.HasPrefix(server.getCommands()[authCommandIndex], "AUTH PLAIN "))
	})

	t.Run("answers known prompts regardless of case when the server offers only LOGIN", func(t *testing.T) {
		t.Parallel()
		promptSets := [][2]string{
			{"Username:", "Password:"},
			{"User Name\x00", "Password\x00"},
			{"USERNAME:", "password:"},
		}

		for _, prompts := range promptSets {
			t.Run(prompts[0], func(t *testing.T) {
				server := startFakeServer(t, fakeServerOptions{
					authMechanisms: []string{mechanismLogin},
					loginPrompts:   prompts,
				})

				err := Send(t.Context(), server.getConnection(SecurityNone), newTestMessage(t))
				require.NoError(t, err)

				server.waitForSessionEnd(t)
				assert.Contains(t, server.getCommands(), "AUTH LOGIN")
				assert.Len(t, server.getMessages(), 1)
			})
		}
	})

	t.Run("fails without guessing when the LOGIN prompt is unknown", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{
			authMechanisms: []string{mechanismLogin},
			loginPrompts:   [2]string{"Secret:", "Password:"},
		})

		err := Send(t.Context(), server.getConnection(SecurityNone), newTestMessage(t))
		require.ErrorIs(t, err, ErrUnknownLoginPrompt)

		server.waitForSessionEnd(t)
		assert.Empty(t, server.getMessages())
	})

	t.Run(
		"returns the server response without the password when the server rejects the credentials",
		func(t *testing.T) {
			t.Parallel()
			for _, mechanism := range []string{mechanismPlain, mechanismLogin} {
				t.Run(mechanism, func(t *testing.T) {
					server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{mechanism}})

					rejectedConnection := server.getConnection(SecurityNone)
					rejectedConnection.Password = "wrong-password-value"

					err := Send(t.Context(), rejectedConnection, newTestMessage(t))
					require.Error(t, err)

					assert.Contains(t, err.Error(), "535")
					assert.Contains(t, err.Error(), "Authentication credentials invalid")
					assert.Contains(t, err.Error(), mechanism)
					assert.NotContains(t, err.Error(), "wrong-password-value")
				})
			}
		},
	)

	t.Run("fails when the server offers no known mechanism", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{authMechanisms: []string{"CRAM-MD5"}})

		err := Send(t.Context(), server.getConnection(SecurityNone), newTestMessage(t))

		require.ErrorIs(t, err, ErrAuthMechanismNotOffered)
	})

	t.Run("delivers without authentication with STARTTLS and no credentials", func(t *testing.T) {
		t.Parallel()
		server := startFakeServer(t, fakeServerOptions{startTLSConfig: newSelfSignedTLSConfig(t)})

		relayConnection := server.getConnection(SecurityStartTLS)
		relayConnection.Username = ""
		relayConnection.Password = ""

		err := Send(t.Context(), relayConnection, newTestMessage(t))
		require.NoError(t, err)

		server.waitForSessionEnd(t)
		assert.Equal(
			t,
			[]string{"EHLO", "STARTTLS", "EHLO", "MAIL", "RCPT", "DATA", "QUIT"},
			server.getCommandVerbs(),
		)
		assert.Len(t, server.getMessages(), 1)
	})
}

func TestAuth(t *testing.T) {
	t.Parallel()

	t.Run("refuses to start on an unencrypted connection without an opt-in", func(t *testing.T) {
		t.Parallel()
		unencryptedServer := &smtp.ServerInfo{Name: "relay.example.com", TLS: false}
		userCredentials := credentials{fakeServerUsername, fakeServerPassword, false}

		for _, auth := range []smtp.Auth{&plainAuth{userCredentials}, &loginAuth{userCredentials}} {
			_, initialResponse, err := auth.Start(unencryptedServer)

			require.ErrorIs(t, err, ErrUnencryptedAuth)
			assert.Empty(t, initialResponse)
		}
	})
}
