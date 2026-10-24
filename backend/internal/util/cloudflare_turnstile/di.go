package cloudflare_turnstile

import (
	"sync"

	"databasus-backend/internal/config"
)

var getCloudflareTurnstileService = sync.OnceValue(func() *CloudflareTurnstileService {
	return &CloudflareTurnstileService{
		config.GetEnv().CloudflareTurnstileSecretKey,
		config.GetEnv().CloudflareTurnstileSiteKey,
	}
})

func GetCloudflareTurnstileService() *CloudflareTurnstileService {
	return getCloudflareTurnstileService()
}
