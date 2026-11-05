package config

import (
	"encoding/base64"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/database"
)

const (
	defaultConfigPageLimit = 20
	maxConfigPageLimit     = 100
)

// ListQuery takes exactly one of WorkspaceID and StorageID.
type ListQuery struct {
	WorkspaceID *uuid.UUID
	StorageID   *uuid.UUID
	// Limit is nil for the default page size.
	Limit *int
	// After is the cursor of the previous page, nil for the first page.
	After *string
}

// Page holds saved configs ordered by database ID.
type Page struct {
	Configs []AnyBackupConfig
	// NextCursor is nil on the last page.
	NextCursor *string
}

func encodeCursor(lastDatabaseID uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString(lastDatabaseID[:])
}

func decodeCursor(encoded string) (uuid.UUID, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return uuid.UUID{}, database.ErrInvalidCursor
	}

	databaseID, err := uuid.FromBytes(decoded)
	if err != nil {
		return uuid.UUID{}, database.ErrInvalidCursor
	}

	return databaseID, nil
}
