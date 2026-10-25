package crypto

import "errors"

var ErrMasterKeyTooShort = errors.New("the master key is shorter than 32 bytes")

var ErrMalformedCiphertext = errors.New("the encrypted value is not in the enc:<nonce>:<ciphertext> format")
