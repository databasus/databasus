package restorestream

import "errors"

var ErrInvalidToken = errors.New("invalid or expired restore token")
