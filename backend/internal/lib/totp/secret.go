// Package totp computes and checks time-based one-time codes of RFC 6238 with the parameters every
// common authenticator app reads without manual settings: HMAC-SHA1, six digits, a 30-second step.
package totp

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"strings"
)

// secretByteCount is 160 bits, the key length RFC 4226 recommends for HMAC-SHA1.
const secretByteCount = 20

// GenerateSecret returns a new random secret as unpadded base32, the form apps accept by hand.
func GenerateSecret() (string, error) {
	secretBytes := make([]byte, secretByteCount)

	_, err := rand.Read(secretBytes)
	if err != nil {
		return "", fmt.Errorf("generate the secret: %w", err)
	}

	return getSecretEncoding().EncodeToString(secretBytes), nil
}

func decodeSecret(secret string) ([]byte, error) {
	key, err := getSecretEncoding().DecodeString(strings.ToUpper(secret))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrSecretMalformed, err)
	}

	return key, nil
}

func getSecretEncoding() *base32.Encoding {
	return base32.StdEncoding.WithPadding(base32.NoPadding)
}
