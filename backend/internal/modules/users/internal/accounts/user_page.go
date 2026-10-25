package accounts

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/users/internal/identity"
)

const (
	defaultUserPageLimit = 20
	maxUserPageLimit     = 100
)

type UserListQuery struct {
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After  *string
	Search string
}

type UserPage struct {
	Users []identity.User
	// NextCursor is nil on the last page.
	NextCursor *string
}

// userCursor is the position after the last user of a page. The list is ordered by creation time,
// newest first, and the ID orders users created at the same moment.
type userCursor struct {
	createdAt time.Time
	userID    uuid.UUID
}

func (c userCursor) encode() string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(c.createdAt.UTC().Format(time.RFC3339Nano) + "|" + c.userID.String()),
	)
}

func decodeUserCursor(encoded string) (userCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return userCursor{}, ErrInvalidCursor
	}

	createdAtText, userIDText, isFound := strings.Cut(string(decoded), "|")
	if !isFound {
		return userCursor{}, ErrInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		return userCursor{}, ErrInvalidCursor
	}

	userID, err := uuid.Parse(userIDText)
	if err != nil {
		return userCursor{}, ErrInvalidCursor
	}

	return userCursor{createdAt: createdAt, userID: userID}, nil
}
