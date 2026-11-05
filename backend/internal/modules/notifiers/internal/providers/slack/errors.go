package slack

import "errors"

var errRateLimited = errors.New("slack kept rate-limiting the message")
