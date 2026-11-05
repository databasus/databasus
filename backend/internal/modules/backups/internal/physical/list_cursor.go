package physical

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/backups/internal/physical/catalog"
)

type physicalListCursor struct {
	SortTime time.Time `json:"t"`
	ID       uuid.UUID `json:"id"`
}

func encodePhysicalListCursor(position catalog.BackupListPosition) (string, error) {
	cursorJSON, err := json.Marshal(physicalListCursor{SortTime: position.SortTime, ID: position.ID})
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(cursorJSON), nil
}

func decodePhysicalListCursor(encodedCursor string) (*catalog.BackupListPosition, error) {
	if encodedCursor == "" {
		return nil, nil
	}

	cursorJSON, err := base64.RawURLEncoding.DecodeString(encodedCursor)
	if err != nil {
		return nil, ErrInvalidBackupsCursor
	}

	var cursor physicalListCursor
	if err := json.Unmarshal(cursorJSON, &cursor); err != nil {
		return nil, ErrInvalidBackupsCursor
	}

	if cursor.SortTime.IsZero() || cursor.ID == uuid.Nil {
		return nil, ErrInvalidBackupsCursor
	}

	return &catalog.BackupListPosition{SortTime: cursor.SortTime, ID: cursor.ID}, nil
}
