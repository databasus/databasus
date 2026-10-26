package encryption

// DecryptFieldIfNeeded decrypts an encrypted field, or returns it unchanged when encryptor is nil:
// a restore target config is never persisted, so it arrives in clear text.
func DecryptFieldIfNeeded(value string, encryptor FieldEncryptor) (string, error) {
	if encryptor == nil {
		return value, nil
	}

	return encryptor.Decrypt(value)
}
