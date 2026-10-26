package database_test

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"databasus-backend/internal/lib/database"
)

func TestNameCursor(t *testing.T) {
	t.Parallel()

	t.Run("decodes to the position it was encoded from", func(t *testing.T) {
		t.Parallel()
		position := database.NameCursor{Name: "Production", ID: uuid.New()}

		decodedPosition, err := database.DecodeNameCursor(position.Encode())

		require.NoError(t, err)
		assert.Equal(t, position, decodedPosition)
	})

	t.Run("keeps the name whole when it contains the separator", func(t *testing.T) {
		t.Parallel()
		position := database.NameCursor{Name: "team|prod|eu", ID: uuid.New()}

		decodedPosition, err := database.DecodeNameCursor(position.Encode())

		require.NoError(t, err)
		assert.Equal(t, position, decodedPosition)
	})

	malformedCursors := []struct {
		name   string
		cursor string
	}{
		{name: "refuses a cursor that is not base64", cursor: "not a cursor!"},
		{name: "refuses a cursor without the separator", cursor: base64.RawURLEncoding.EncodeToString([]byte("name"))},
		{
			name:   "refuses a cursor whose ID is not a UUID",
			cursor: base64.RawURLEncoding.EncodeToString([]byte("not-a-uuid|name")),
		},
	}

	for _, malformedCursor := range malformedCursors {
		t.Run(malformedCursor.name, func(t *testing.T) {
			t.Parallel()

			_, err := database.DecodeNameCursor(malformedCursor.cursor)

			require.ErrorIs(t, err, database.ErrInvalidCursor)
		})
	}
}
