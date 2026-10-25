// Package encryption hands the master key and the field encryptor that internal/app built to the code
// that has not moved into internal/modules yet. A test binary built with the testhelpers tag opens
// the key on first use instead.
package encryption

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/lib/crypto"
)

type installedEncryption struct {
	masterKey      *crypto.MasterKey
	fieldEncryptor FieldEncryptor
}

var installed atomic.Pointer[installedEncryption]

var getEncryptionWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(masterKey *crypto.MasterKey, fieldEncryptor FieldEncryptor) {
	installed.Store(&installedEncryption{masterKey: masterKey, fieldEncryptor: fieldEncryptor})
}

func GetFieldEncryptor() FieldEncryptor {
	return getEncryption().fieldEncryptor
}

func GetMasterKey() *crypto.MasterKey {
	return getEncryption().masterKey
}

func getEncryption() *installedEncryption {
	if encryption := installed.Load(); encryption != nil {
		return encryption
	}

	return getEncryptionWithoutApp()
}
