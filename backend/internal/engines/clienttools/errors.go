package clienttools

import "errors"

var (
	ErrBundleRootNotFound     = errors.New("client tool bundle root not found")
	ErrUnsupportedPlatform    = errors.New("no client tools are shipped for this platform")
	ErrClientToolsBroken      = errors.New("client tools broken")
	ErrForeignClientVersion   = errors.New("client reports a version of another release line")
	ErrEmptyVersionIdentity   = errors.New("empty version identity")
	ErrInvalidVersionIdentity = errors.New("version identity is not a dotted number")
)
