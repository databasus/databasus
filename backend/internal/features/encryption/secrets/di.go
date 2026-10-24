package secrets

import "sync"

var getSecretKeyService = sync.OnceValue(func() *SecretKeyService {
	return &SecretKeyService{}
})

func GetSecretKeyService() *SecretKeyService {
	return getSecretKeyService()
}
