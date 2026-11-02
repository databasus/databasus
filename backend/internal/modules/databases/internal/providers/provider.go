// Package providers declares what a database provider is and keeps the registry that maps a database
// type to its provider. Each provider lives in a subpackage with its settings, its table and the
// engine it connects through.
package providers

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Settings are the provider-specific settings of one database. Secrets in settings that a repository
// loaded stay encrypted; only Connect and the connection settings handed to other modules decrypt
// them.
type Settings interface {
	GetType() Type
}

type Provider interface {
	GetType() Type
	// Validate answers an apierror problem naming the setting it refuses. It refuses before anything
	// connects, the embedded internal database of Databasus included.
	Validate(settings Settings) error
	// MergeStoredSettings fills the secrets the incoming settings left out from the stored ones, and
	// keeps what the provider owns, such as the replication slot name of a physical database. A secret
	// tied to the database's resource address is filled only while that address stays the stored one,
	// so a user who may not see a secret cannot send it to a server of their choice. Left empty, the
	// secret then gets the refusal of Validate when the settings use it, and is removed when they do not.
	MergeStoredSettings(merge SettingsMerge) (Settings, error)
	// PrepareForCreate sets what the provider owns from the first save on, such as the replication
	// slot name of a physical database.
	PrepareForCreate(settings Settings) Settings
	// CopyForNewDatabase keeps the settings and the secrets, and drops what belongs to the original
	// alone, such as its replication slot and the identity of the cluster it was pointed at.
	CopyForNewDatabase(settings Settings) Settings
	// Connect decrypts the settings, opens the engine's tunnel and builds the engine source. The caller
	// closes the connected database.
	Connect(ctx context.Context, settings Settings) (ConnectedDatabase, error)
	// GetTelemetryFacts reads what the anonymous usage statistics report about a database.
	GetTelemetryFacts(settings Settings) (TelemetryFacts, error)
	// GetConnectionSettings decrypts the settings for the code that connects on its own.
	GetConnectionSettings(databaseID uuid.UUID, settings Settings) (ConnectionSettings, error)
	GetRepository() SettingsRepository
}

// SettingsMerge names both sides, so a swapped call cannot keep the stored settings over the edit.
type SettingsMerge struct {
	Incoming Settings
	Stored   Settings
}

// ConnectedDatabase is an open connection to one user database through its engine.
type ConnectedDatabase interface {
	// TestConnection connects and checks that a backup of this type can run.
	TestConnection(ctx context.Context, logger *slog.Logger) error
	// Detect returns the settings with what the server reports filled in: the version, the
	// privileges of MySQL and MariaDB, and for physical PostgreSQL the system identifier and the WAL
	// segment size.
	Detect(ctx context.Context, logger *slog.Logger) (Settings, error)
	// ProbeLiveness is the check of scheduled health checks.
	ProbeLiveness(ctx context.Context, logger *slog.Logger) error
	CheckLimitedUser(ctx context.Context, logger *slog.Logger) (LimitedUserCheck, error)
	CreateLimitedUser(ctx context.Context, logger *slog.Logger) (LimitedUser, error)
	Close()
}

// LimitedUserCheck tells whether the connecting user may do more than a backup needs, and what.
type LimitedUserCheck struct {
	IsAdvisable bool
	Privileges  []string
}

// LimitedUser is a read-only user, or for physical PostgreSQL a replication-only user, created on
// the user's server. Its password is returned only once.
type LimitedUser struct {
	Username string
	Password string
	// IsForcedWalRotationAvailable is set only for a replication-only user.
	IsForcedWalRotationAvailable bool
}

// TelemetryFacts are what the anonymous usage statistics report about the settings of a database.
type TelemetryFacts struct {
	Version string
	// PhysicalBackupType is empty for every type but physical PostgreSQL.
	PhysicalBackupType PhysicalBackupType
	IsSSHTunnelEnabled bool
}

// SettingsRepository keeps the settings of one provider in its own table. Upsert encrypts the secrets
// that are not encrypted yet.
type SettingsRepository interface {
	Upsert(ctx context.Context, databaseID uuid.UUID, settings Settings) error
	// ListByDatabaseIDs leaves out the IDs that have no row.
	ListByDatabaseIDs(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]Settings, error)
}

// CastSettings returns settings as the settings type S of one provider.
func CastSettings[S Settings](settings Settings) (S, error) {
	typedSettings, isOfType := settings.(S)
	if !isOfType {
		return typedSettings, ErrSettingsOfAnotherType
	}

	return typedSettings, nil
}
