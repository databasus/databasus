package process

import "errors"

var ErrUnsupportedSignal = errors.New("signal is not a unix signal")
