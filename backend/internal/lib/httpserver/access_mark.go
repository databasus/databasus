package httpserver

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// AccessMark is the x-access value of a contract operation. The users middleware enforces it.
type AccessMark string

const (
	// AccessMarkSignedIn is the mark of an operation without x-access, so a request the guard did
	// not see reads as the strictest mark short of admin.
	AccessMarkSignedIn AccessMark = ""
	AccessMarkPublic   AccessMark = "public"
	AccessMarkAdmin    AccessMark = "admin"
)

const accessMarkExtension = "x-access"

type accessMarkKey struct{}

func withAccessMark(ctx context.Context, mark AccessMark) context.Context {
	return context.WithValue(ctx, accessMarkKey{}, mark)
}

// GetAccessMark returns the mark of the contract operation the request calls.
func GetAccessMark(ctx context.Context) AccessMark {
	mark, _ := ctx.Value(accessMarkKey{}).(AccessMark)

	return mark
}

func readAccessMark(operation *openapi3.Operation) (AccessMark, error) {
	rawMark, hasMark := operation.Extensions[accessMarkExtension]
	if !hasMark {
		return AccessMarkSignedIn, nil
	}

	switch mark := AccessMark(fmt.Sprint(rawMark)); mark {
	case AccessMarkPublic, AccessMarkAdmin:
		return mark, nil
	case AccessMarkSignedIn:
	}

	return "", fmt.Errorf("%w: operation %s has %s %v", ErrUnknownAccessMark, operation.OperationID,
		accessMarkExtension, rawMark)
}
