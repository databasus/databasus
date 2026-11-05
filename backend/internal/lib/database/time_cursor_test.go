package database_test

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
)

func TestTimeCursor(t *testing.T) {
	t.Parallel()

	t.Run("decodes to the position it was encoded from", func(t *testing.T) {
		t.Parallel()
		position := database.TimeCursor{Time: time.Date(2026, 5, 30, 12, 0, 0, 123456000, time.UTC), ID: uuid.New()}

		decodedPosition, err := database.DecodeTimeCursor(position.Encode())

		require.NoError(t, err)
		assert.Equal(t, position, decodedPosition)
	})

	t.Run("decodes the time in UTC when it was encoded in another zone", func(t *testing.T) {
		t.Parallel()
		zone := time.FixedZone("UTC+3", 3*60*60)
		position := database.TimeCursor{Time: time.Date(2026, 5, 30, 15, 0, 0, 0, zone), ID: uuid.New()}

		decodedPosition, err := database.DecodeTimeCursor(position.Encode())

		require.NoError(t, err)
		assert.Equal(t, time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC), decodedPosition.Time)
	})

	malformedCursors := []struct {
		name   string
		cursor string
	}{
		{name: "refuses a cursor that is not base64", cursor: "not a cursor!"},
		{
			name:   "refuses a cursor without the separator",
			cursor: base64.RawURLEncoding.EncodeToString([]byte("2026-05-30T12:00:00Z")),
		},
		{
			name:   "refuses a cursor whose time is not RFC 3339",
			cursor: base64.RawURLEncoding.EncodeToString([]byte("yesterday|" + uuid.NewString())),
		},
		{
			name:   "refuses a cursor whose ID is not a UUID",
			cursor: base64.RawURLEncoding.EncodeToString([]byte("2026-05-30T12:00:00Z|not-a-uuid")),
		},
	}

	for _, malformedCursor := range malformedCursors {
		t.Run(malformedCursor.name, func(t *testing.T) {
			t.Parallel()

			_, err := database.DecodeTimeCursor(malformedCursor.cursor)

			require.ErrorIs(t, err, database.ErrInvalidCursor)
		})
	}
}
