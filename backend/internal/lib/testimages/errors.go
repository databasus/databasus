package testimages

import "errors"

var (
	ErrInvalidCatalog      = errors.New("invalid immutable test image catalog")
	ErrUnknownImage        = errors.New("unknown test image selector")
	ErrUnavailablePlatform = errors.New("test image has no recorded artifact for this platform")
)
