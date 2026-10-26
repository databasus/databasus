package telegram

import "errors"

var (
	errInvalidProxyURL            = errors.New("proxy URL must be a valid http/https/socks5/socks5h URL")
	errUnexpectedDefaultTransport = errors.New("unexpected default HTTP transport type")
)
