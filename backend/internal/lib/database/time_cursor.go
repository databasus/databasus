package database

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// TimeCursor is the position after the last row of a page in a list ordered newest first by a time,
// where the ID orders rows with the same time.
type TimeCursor struct {
	Time time.Time
	ID   uuid.UUID
}

func (c TimeCursor) Encode() string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(c.Time.UTC().Format(time.RFC3339Nano) + "|" + c.ID.String()),
	)
}

func DecodeTimeCursor(encoded string) (TimeCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return TimeCursor{}, ErrInvalidCursor
	}

	timeText, idText, isFound := strings.Cut(string(decoded), "|")
	if !isFound {
		return TimeCursor{}, ErrInvalidCursor
	}

	cursorTime, err := time.Parse(time.RFC3339Nano, timeText)
	if err != nil {
		return TimeCursor{}, ErrInvalidCursor
	}

	id, err := uuid.Parse(idText)
	if err != nil {
		return TimeCursor{}, ErrInvalidCursor
	}

	return TimeCursor{Time: cursorTime.UTC(), ID: id}, nil
}
