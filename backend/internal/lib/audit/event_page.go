package audit

import (
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EventListQuery leaves a filter nil to not filter by it. After is the NextCursor of the previous
// page, empty for the first page.
type EventListQuery struct {
	WorkspaceID  *uuid.UUID
	AuthorUserID *uuid.UUID
	Limit        int
	After        string
}

type EventPage struct {
	Events []RecordedEvent
	// NextCursor is empty on the last page.
	NextCursor string
}

// eventCursor is the position after the last event of a page. Events are ordered by creation time,
// newest first, and the ID orders events created at the same moment. Neither ever changes, so an
// event written while a client pages sorts before every cursor and never shifts the next pages.
type eventCursor struct {
	createdAt time.Time
	eventID   uuid.UUID
}

func (c eventCursor) encode() string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(c.createdAt.UTC().Format(time.RFC3339Nano) + "|" + c.eventID.String()),
	)
}

func decodeEventCursor(encoded string) (eventCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return eventCursor{}, ErrInvalidCursor
	}

	createdAtText, eventIDText, isFound := strings.Cut(string(decoded), "|")
	if !isFound {
		return eventCursor{}, ErrInvalidCursor
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		return eventCursor{}, ErrInvalidCursor
	}

	eventID, err := uuid.Parse(eventIDText)
	if err != nil {
		return eventCursor{}, ErrInvalidCursor
	}

	return eventCursor{createdAt: createdAt, eventID: eventID}, nil
}
