package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const encryptedPrefix = "enc:"

// FieldEncryptor encrypts single values, such as a provider's password, with AES-GCM under the first
// 32 bytes of the master key. An encrypted value reads enc:<nonce>:<ciphertext>, both in base64, so
// it carries its own nonce and values from every earlier release decrypt.
type FieldEncryptor struct {
	key []byte
}

func NewFieldEncryptor(masterKey *MasterKey) *FieldEncryptor {
	return &FieldEncryptor{key: []byte(masterKey.GetSecret())[:minMasterKeyBytes]}
}

// Encrypt returns an empty or already encrypted value unchanged.
func (e *FieldEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" || isEncrypted(plaintext) {
		return plaintext, nil
	}

	aead, err := e.newAEAD()
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())

	_, err = rand.Read(nonce)
	if err != nil {
		return "", fmt.Errorf("generate a nonce: %w", err)
	}

	ciphertext := aead.Seal(nil, nonce, []byte(plaintext), nil)

	return encryptedPrefix + base64.StdEncoding.EncodeToString(nonce) + ":" +
		base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt returns an empty or unencrypted value unchanged.
func (e *FieldEncryptor) Decrypt(value string) (string, error) {
	if value == "" || !isEncrypted(value) {
		return value, nil
	}

	parts := strings.SplitN(value, ":", 3)
	if len(parts) != 3 {
		return "", ErrMalformedCiphertext
	}

	nonce, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode the nonce: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("decode the ciphertext: %w", err)
	}

	aead, err := e.newAEAD()
	if err != nil {
		return "", err
	}

	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt the value: %w", err)
	}

	return string(plaintext), nil
}

func (e *FieldEncryptor) newAEAD() (cipher.AEAD, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("create the cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	return aead, nil
}

func isEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}
