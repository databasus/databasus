package crypto

import "errors"

var ErrMasterKeyTooShort = errors.New("the master key is shorter than 32 bytes")

var ErrMalformedCiphertext = errors.New("the encrypted value is not in the enc:<nonce>:<ciphertext> format")

var ErrEmptyMasterKey = errors.New("the master key is empty")

var ErrMalformedStreamHeader = errors.New("the stream header has a wrong length")

var ErrInvalidStreamMagicBytes = errors.New("the stream does not start with the magic bytes of an encrypted stream")

var ErrStreamHeaderMismatch = errors.New("the stream header does not match the header stored with the stream")

var ErrInvalidStreamChunkLength = errors.New("the encrypted stream holds an invalid chunk length")

var ErrStreamChunkAuthentication = errors.New(
	"the encrypted chunk failed authentication, the file may be corrupted or tampered",
)
