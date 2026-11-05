package providers

import (
	"fmt"

	"databasus-backend/internal/lib/crypto"
)

// DecryptSecret wraps a failure in ErrSecretUnreadable, so the caller tells it apart from a server
// that refuses the connection.
func DecryptSecret(fieldEncryptor *crypto.FieldEncryptor, storedValue, secretName string) (string, error) {
	decryptedValue, err := fieldEncryptor.Decrypt(storedValue)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrSecretUnreadable, secretName, err)
	}

	return decryptedValue, nil
}
