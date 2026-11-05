package apptest

import (
	"testing"

	"databasus-backend/internal/lib/crypto"
)

// EncryptSecret encrypts the value with the master key of the instance, as the application stores
// a secret. A test uses it to write a row in a shape an older release stored.
func (a *App) EncryptSecret(tb testing.TB, plaintext string) string {
	tb.Helper()

	masterKey, err := crypto.OpenOrCreateMasterKey(a.secretKeyPath)
	if err != nil {
		tb.Fatalf("open the master key of the test application: %v", err)
	}

	encryptedValue, err := crypto.NewFieldEncryptor(masterKey).Encrypt(plaintext)
	if err != nil {
		tb.Fatalf("encrypt the secret: %v", err)
	}

	return encryptedValue
}

// DecryptSecret decrypts a value the application stored with the master key of the instance, so a
// test can show that a stored secret is the one the user entered.
func (a *App) DecryptSecret(tb testing.TB, encryptedValue string) string {
	tb.Helper()

	masterKey, err := crypto.OpenOrCreateMasterKey(a.secretKeyPath)
	if err != nil {
		tb.Fatalf("open the master key of the test application: %v", err)
	}

	plaintext, err := crypto.NewFieldEncryptor(masterKey).Decrypt(encryptedValue)
	if err != nil {
		tb.Fatalf("decrypt the secret: %v", err)
	}

	return plaintext
}
