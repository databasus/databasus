package sshtunnel

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestBuildAuthMethods(t *testing.T) {
	t.Parallel()

	// The unusable private key is the point: a builder that still branched on field presence would
	// try to parse it and fail, so this pins the auth type as the only thing that decides.
	t.Run("returns only the password method when auth type is password", func(t *testing.T) {
		t.Parallel()
		settings := Settings{AuthType: AuthTypePassword, Password: "tunnelpassword", PrivateKey: "not a key"}

		authMethods, err := buildAuthMethods(settings)

		require.NoError(t, err)
		require.Len(t, authMethods, 1)
		assert.IsType(t, ssh.Password(""), authMethods[0])
	})

	t.Run("returns only the public key method when auth type is private key", func(t *testing.T) {
		t.Parallel()
		settings := Settings{
			AuthType:   AuthTypePrivateKey,
			Password:   "tunnelpassword",
			PrivateKey: generatePrivateKey(t),
		}

		authMethods, err := buildAuthMethods(settings)

		require.NoError(t, err)
		require.Len(t, authMethods, 1)
		assert.IsType(t, ssh.PublicKeys(), authMethods[0])
	})

	t.Run("returns an error when auth type is unknown", func(t *testing.T) {
		t.Parallel()

		_, err := buildAuthMethods(Settings{AuthType: "CERTIFICATE", Password: "tunnelpassword"})

		require.ErrorIs(t, err, ErrUnknownAuthType)
	})
}

func generatePrivateKey(t *testing.T) string {
	t.Helper()

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	pemBlock, err := ssh.MarshalPrivateKey(privateKey, "")
	require.NoError(t, err)

	return string(pem.EncodeToMemory(pemBlock))
}
