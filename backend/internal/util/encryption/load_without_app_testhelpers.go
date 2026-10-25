//go:build testhelpers

package encryption

import (
	"fmt"

	"databasus-backend/internal/config"
	"databasus-backend/internal/lib/crypto"
)

func loadWithoutApp() *installedEncryption {
	masterKey, err := crypto.OpenOrCreateMasterKey(config.GetEnv().SecretKeyPath)
	if err != nil {
		panic(fmt.Sprintf("open the test master key: %v", err))
	}

	return &installedEncryption{masterKey: masterKey, fieldEncryptor: crypto.NewFieldEncryptor(masterKey)}
}
