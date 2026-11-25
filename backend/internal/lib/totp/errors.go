package totp

import "errors"

var ErrSecretMalformed = errors.New("the secret is not unpadded base32")
