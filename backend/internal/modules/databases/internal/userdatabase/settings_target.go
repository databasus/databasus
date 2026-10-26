package userdatabase

import (
	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases/internal/providers"
)

// SettingsTarget names settings that are not saved, in the workspace whose content they belong to.
// DatabaseID names the saved database whose stored secrets fill the ones the settings leave out, nil
// for a new database.
type SettingsTarget struct {
	WorkspaceID uuid.UUID
	DatabaseID  *uuid.UUID
	Settings    providers.Settings
}

// ResolvedTarget are the settings ready to connect with. Database is nil for a new database.
type ResolvedTarget struct {
	Database *Database
	Settings providers.Settings
}
