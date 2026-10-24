package encryption

import (
	"sync"

	"databasus-backend/internal/features/encryption/secrets"
)

var getFieldEncryptor = sync.OnceValue(func() *SecretKeyFieldEncryptor {
	return &SecretKeyFieldEncryptor{
		secrets.GetSecretKeyService(),
	}
})

func GetFieldEncryptor() FieldEncryptor {
	return getFieldEncryptor()
}
