package enginetest

import "github.com/google/uuid"

// NewUniqueSuffix gives the objects a test creates on a shared server names no other test takes.
func NewUniqueSuffix() string {
	return uuid.NewString()[:8]
}
