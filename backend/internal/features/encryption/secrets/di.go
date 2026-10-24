package secrets

var secretKeyService = &SecretKeyService{}

func GetSecretKeyService() *SecretKeyService {
	return secretKeyService
}
