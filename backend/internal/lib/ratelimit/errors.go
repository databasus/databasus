package ratelimit

import "errors"

var ErrInvalidSettings = errors.New("rate limit and window must be positive")
