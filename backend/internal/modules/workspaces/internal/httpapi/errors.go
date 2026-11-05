package httpapi

import "errors"

var errNoActor = errors.New("the request reached a signed-in operation without an actor")
