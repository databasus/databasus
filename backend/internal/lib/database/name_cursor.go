package database

import (
	"encoding/base64"
	"strings"

	"github.com/google/uuid"
)

// NameCursor is the position after the last row of a page in a list ordered by name, where the ID
// orders rows with the same name.
type NameCursor struct {
	Name string
	ID   uuid.UUID
}

// Encode puts the ID first, because a name may contain the separator.
func (c NameCursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(c.ID.String() + "|" + c.Name))
}

func DecodeNameCursor(encoded string) (NameCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return NameCursor{}, ErrInvalidCursor
	}

	idText, name, isFound := strings.Cut(string(decoded), "|")
	if !isFound {
		return NameCursor{}, ErrInvalidCursor
	}

	id, err := uuid.Parse(idText)
	if err != nil {
		return NameCursor{}, ErrInvalidCursor
	}

	return NameCursor{Name: name, ID: id}, nil
}
