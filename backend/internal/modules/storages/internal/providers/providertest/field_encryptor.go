// Package providertest holds what the tests of the storage providers share: a field encryptor with a
// key of its own, the address of a container and a logger that writes nowhere.
package providertest

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"

	"databasus-backend/internal/lib/crypto"
	containers "databasus-backend/internal/lib/testcontainers"
)

// NewFieldEncryptor returns an encryptor under a master key of the test's own.
func NewFieldEncryptor(tb testing.TB) *crypto.FieldEncryptor {
	tb.Helper()

	masterKey, err := crypto.OpenOrCreateMasterKey(filepath.Join(tb.TempDir(), "secret.key"))
	if err != nil {
		tb.Fatalf("create the master key: %v", err)
	}

	return crypto.NewFieldEncryptor(masterKey)
}

// Encrypt encrypts the secret the way a repository stores it, so a test hands a provider the shape
// it reads from the database.
func Encrypt(tb testing.TB, fieldEncryptor *crypto.FieldEncryptor, secret string) string {
	tb.Helper()

	encryptedSecret, err := fieldEncryptor.Encrypt(secret)
	if err != nil {
		tb.Fatalf("encrypt the secret: %v", err)
	}

	return encryptedSecret
}

func GetAddress(endpoint containers.Endpoint) string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

func NewDiscardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}
