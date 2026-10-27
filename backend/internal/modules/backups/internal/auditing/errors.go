package auditing

import "errors"

var errActorNotFound = errors.New("the signed-in user does not exist")
