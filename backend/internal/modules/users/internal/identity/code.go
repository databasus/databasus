package identity

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
)

// GenerateSixDigitCode returns a code for a sign-in or a password reset from a cryptographically
// secure source.
func GenerateSixDigitCode() (string, error) {
	randomBytes := make([]byte, 4)

	_, err := io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		return "", fmt.Errorf("generate a code: %w", err)
	}

	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(randomBytes)%1_000_000), nil
}
