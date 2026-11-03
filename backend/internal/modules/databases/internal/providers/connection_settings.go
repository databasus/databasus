package providers

import (
	"strconv"

	"github.com/google/uuid"

	"databasus-backend/internal/engines/mariadb"
	"databasus-backend/internal/engines/mongodb"
	"databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	pgshared "databasus-backend/internal/engines/postgresql/shared"
)

// ConnectionSettings are the decrypted settings of one database with the options of its type, for
// the code that opens the engine's tunnel and calls the engine itself. Each type has its own struct.
type ConnectionSettings interface {
	GetType() Type
}

type PostgresqlLogicalConnectionSettings struct {
	DatabaseID uuid.UUID
	// Engine carries the password, the certificates and the tunnel secrets in clear text.
	Engine             pglogical.Settings
	Version            pgshared.Version
	IncludeSchemas     []string
	ExcludeTables      []string
	ExcludeTableData   []string
	CPUCount           int
	IsSkipUserMappings bool
}

func (PostgresqlLogicalConnectionSettings) GetType() Type {
	return TypePostgresqlLogical
}

// GetDumpFilter is the scope of the dump the settings describe.
func (s PostgresqlLogicalConnectionSettings) GetDumpFilter() pglogical.DumpFilter {
	return pglogical.DumpFilter{
		IncludeSchemas:           s.IncludeSchemas,
		ExcludeTablePatterns:     s.ExcludeTables,
		ExcludeTableDataPatterns: s.ExcludeTableData,
	}
}

type PostgresqlPhysicalConnectionSettings struct {
	DatabaseID uuid.UUID
	// Engine carries the password, the certificates and the tunnel secrets in clear text.
	Engine  pgphysical.Settings
	Version pgshared.Version
	// SystemIdentifier is nil until the first connection detected it.
	SystemIdentifier *string
	// WalSegmentSizeBytes is nil until the first connection detected it.
	WalSegmentSizeBytes *int64
	BackupType          PhysicalBackupType
	WalStreamSlotName   string
	// BackupSlotName names the replication slot each pg_basebackup holds while it runs.
	BackupSlotName string
}

func (PostgresqlPhysicalConnectionSettings) GetType() Type {
	return TypePostgresqlPhysical
}

// GetSystemIdentifierUint64 returns 0 when the identifier is not known yet.
func (s PostgresqlPhysicalConnectionSettings) GetSystemIdentifierUint64() uint64 {
	if s.SystemIdentifier == nil {
		return 0
	}

	if value, err := strconv.ParseUint(*s.SystemIdentifier, 10, 64); err == nil {
		return value
	}

	// pg_control_system() returns system_identifier as a signed bigint; the cast restores its unsigned
	// value bit for bit.
	// nosemgrep: trailofbits.go.string-to-int-signedness-cast.string-to-int-signedness-cast
	if value, err := strconv.ParseInt(*s.SystemIdentifier, 10, 64); err == nil {
		return uint64(value)
	}

	return 0
}

type MysqlConnectionSettings struct {
	DatabaseID uuid.UUID
	// Engine carries the password and the tunnel secrets in clear text.
	Engine        mysql.Settings
	Version       mysql.Version
	ExcludeTables []string
	// Privileges are the backup privileges the account held at the last save.
	Privileges []string
}

func (MysqlConnectionSettings) GetType() Type {
	return TypeMysql
}

type MariadbConnectionSettings struct {
	DatabaseID uuid.UUID
	// Engine carries the password and the tunnel secrets in clear text.
	Engine        mariadb.Settings
	Version       mariadb.Version
	ExcludeTables []string
	// Privileges are the backup privileges the account held at the last save.
	Privileges          []string
	IsExcludeEvents     bool
	IsSkipGaleraDisable bool
}

func (MariadbConnectionSettings) GetType() Type {
	return TypeMariadb
}

type MongodbConnectionSettings struct {
	DatabaseID uuid.UUID
	// Engine carries the password and the tunnel secrets in clear text.
	Engine             mongodb.Settings
	Version            mongodb.Version
	ExcludeCollections []string
	CPUCount           int
}

func (MongodbConnectionSettings) GetType() Type {
	return TypeMongodb
}

// CastConnectionSettings returns connection settings as the type S of one provider, so a caller
// that asks for one type gets an error for a database of another instead of switching on the type.
func CastConnectionSettings[S ConnectionSettings](connectionSettings ConnectionSettings) (S, error) {
	typedSettings, isOfType := connectionSettings.(S)
	if !isOfType {
		return typedSettings, ErrSettingsOfAnotherType
	}

	return typedSettings, nil
}
