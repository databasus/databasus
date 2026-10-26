package providertest

import (
	"path/filepath"
	"testing"

	"databasus-backend/internal/lib/crypto"
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
