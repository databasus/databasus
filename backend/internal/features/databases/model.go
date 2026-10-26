package databases

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	"databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/features/sshtunnel"
	"databasus-backend/internal/util/encryption"
)

type Database struct {
	ID uuid.UUID `json:"id" gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`

	// WorkspaceID can be null when a database is created via restore operation
	// outside the context of any workspace
	WorkspaceID *uuid.UUID   `json:"workspaceId" gorm:"column:workspace_id;type:uuid"`
	Name        string       `json:"name"        gorm:"column:name;type:text;not null"`
	Type        DatabaseType `json:"type"        gorm:"column:type;type:text;not null"`

	PostgresqlLogical  *postgresql_logical.PostgresqlLogicalDatabase   `json:"postgresqlLogical,omitzero"  gorm:"foreignKey:DatabaseID"`
	PostgresqlPhysical *postgresql_physical.PostgresqlPhysicalDatabase `json:"postgresqlPhysical,omitzero" gorm:"foreignKey:DatabaseID"`
	Mysql              *mysql.MysqlDatabase                            `json:"mysql,omitzero"              gorm:"foreignKey:DatabaseID"`
	Mariadb            *mariadb.MariadbDatabase                        `json:"mariadb,omitzero"            gorm:"foreignKey:DatabaseID"`
	Mongodb            *mongodb.MongodbDatabase                        `json:"mongodb,omitzero"            gorm:"foreignKey:DatabaseID"`

	Notifiers []DatabaseNotifier `json:"notifiers" gorm:"-"`

	// these fields are not reliable, but
	// they are used for pretty UI
	LastBackupTime         *time.Time `json:"lastBackupTime,omitzero"          gorm:"column:last_backup_time;type:timestamp with time zone"`
	LastBackupErrorMessage *string    `json:"lastBackupErrorMessage,omitempty" gorm:"column:last_backup_error_message;type:text"`

	HealthStatus *HealthStatus `json:"healthStatus" gorm:"column:health_status;type:text;not null"`
}

// GetNotifierIDs returns the IDs of the notifiers the database uses.
func (d *Database) GetNotifierIDs() []uuid.UUID {
	notifierIDs := make([]uuid.UUID, 0, len(d.Notifiers))
	for _, notifier := range d.Notifiers {
		notifierIDs = append(notifierIDs, notifier.ID)
	}

	return notifierIDs
}

func (d *Database) Validate() error {
	if d.Name == "" {
		return errors.New("name is required")
	}

	switch d.Type {
	case DatabaseTypePostgresLogical:
		if d.PostgresqlLogical == nil {
			return errors.New("postgresql database is required")
		}
		return d.PostgresqlLogical.Validate()
	case DatabaseTypePostgresPhysical:
		if d.PostgresqlPhysical == nil {
			return errors.New("postgresql physical database is required")
		}
		return d.PostgresqlPhysical.Validate()
	case DatabaseTypeMysql:
		if d.Mysql == nil {
			return errors.New("mysql database is required")
		}
		return d.Mysql.Validate()
	case DatabaseTypeMariadb:
		if d.Mariadb == nil {
			return errors.New("mariadb database is required")
		}
		return d.Mariadb.Validate()
	case DatabaseTypeMongodb:
		if d.Mongodb == nil {
			return errors.New("mongodb database is required")
		}
		return d.Mongodb.Validate()
	default:
		return errors.New("invalid database type: " + string(d.Type))
	}
}

func (d *Database) ValidateUpdate(old, new Database) error {
	// Database type cannot be changed after creation — the entire backup
	// structure (storage files, schedulers, etc.) is tied to the type at
	// creation time. Recreating that state automatically is error-prone;
	// it is safer for the user to create a new database and remove the old.
	if old.Type != new.Type {
		return errors.New("database type cannot be changed; create a new database instead")
	}

	if old.Type == DatabaseTypePostgresLogical && old.PostgresqlLogical != nil && new.PostgresqlLogical != nil {
		if err := new.PostgresqlLogical.ValidateUpdate(old.PostgresqlLogical); err != nil {
			return err
		}
	}

	if old.Type == DatabaseTypePostgresPhysical &&
		old.PostgresqlPhysical != nil && new.PostgresqlPhysical != nil {
		if err := new.PostgresqlPhysical.ValidateUpdate(old.PostgresqlPhysical); err != nil {
			return err
		}
	}

	return nil
}

func (d *Database) TestConnection(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	switch d.Type {
	case DatabaseTypePostgresLogical:
		if d.PostgresqlLogical == nil {
			return errors.New("postgresql logical config is not set")
		}

		return d.PostgresqlLogical.TestConnection(logger, encryptor)
	case DatabaseTypePostgresPhysical:
		if d.PostgresqlPhysical == nil {
			return errors.New("postgresql physical config is not set")
		}

		return d.PostgresqlPhysical.TestReplicationConnection(logger, encryptor)
	case DatabaseTypeMysql:
		if d.Mysql == nil {
			return errors.New("mysql config is not set")
		}

		return d.Mysql.TestConnection(logger, encryptor)
	case DatabaseTypeMariadb:
		if d.Mariadb == nil {
			return errors.New("mariadb config is not set")
		}

		return d.Mariadb.TestConnection(logger, encryptor)
	case DatabaseTypeMongodb:
		if d.Mongodb == nil {
			return errors.New("mongodb config is not set")
		}

		return d.Mongodb.TestConnection(logger, encryptor)
	default:
		return errors.New("connection test not supported for database type: " + string(d.Type))
	}
}

// ProbeLiveness backs the scheduled healthcheck. Only the physical type swaps its full connection
// test for a plain liveness probe, because only its test opens a replication connection that
// competes with backups for WAL sender slots; the other types share nothing with their backups and
// keep the full test.
func (d *Database) ProbeLiveness(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if d.Type != DatabaseTypePostgresPhysical {
		return d.TestConnection(logger, encryptor)
	}

	if d.PostgresqlPhysical == nil {
		return errors.New("postgresql physical config is not set")
	}

	return d.PostgresqlPhysical.ProbeLiveness(ctx, logger, encryptor)
}

func (d *Database) GetRawDbSizeMb(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (float64, error) {
	switch d.Type {
	case DatabaseTypePostgresLogical:
		if d.PostgresqlLogical == nil {
			return 0, errors.New("postgresql logical config is not set")
		}

		return d.PostgresqlLogical.GetRawDbSizeMb(ctx, logger, encryptor)
	case DatabaseTypeMysql:
		if d.Mysql == nil {
			return 0, errors.New("mysql config is not set")
		}

		return d.Mysql.GetRawDbSizeMb(ctx, logger, encryptor)
	case DatabaseTypeMariadb:
		if d.Mariadb == nil {
			return 0, errors.New("mariadb config is not set")
		}

		return d.Mariadb.GetRawDbSizeMb(ctx, logger, encryptor)
	case DatabaseTypeMongodb:
		if d.Mongodb == nil {
			return 0, errors.New("mongodb config is not set")
		}

		return d.Mongodb.GetRawDbSizeMb(ctx, logger, encryptor)
	default:
		return 0, errors.New("logical backup not supported for database type: " + string(d.Type))
	}
}

func (d *Database) ShouldSuggestReadOnlyUser(
	ctx context.Context,
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) (bool, []string, error) {
	switch d.Type {
	case DatabaseTypePostgresLogical:
		return d.PostgresqlLogical.ShouldSuggestReadOnlyUser(ctx, logger, encryptor)
	case DatabaseTypePostgresPhysical:
		return d.PostgresqlPhysical.ShouldSuggestReplicationOnlyUser(ctx, logger, encryptor)
	case DatabaseTypeMysql:
		return d.Mysql.ShouldSuggestReadOnlyUser(ctx, logger, encryptor)
	case DatabaseTypeMariadb:
		return d.Mariadb.ShouldSuggestReadOnlyUser(ctx, logger, encryptor)
	case DatabaseTypeMongodb:
		return d.Mongodb.ShouldSuggestReadOnlyUser(ctx, logger, encryptor)
	default:
		return false, nil, errors.New("limited user suggestion not supported for this database type")
	}
}

func (d *Database) HideSensitiveData() {
	if d.PostgresqlLogical != nil {
		d.PostgresqlLogical.HideSensitiveData()
	}
	if d.PostgresqlPhysical != nil {
		d.PostgresqlPhysical.HideSensitiveData()
	}
	if d.Mysql != nil {
		d.Mysql.HideSensitiveData()
	}
	if d.Mariadb != nil {
		d.Mariadb.HideSensitiveData()
	}
	if d.Mongodb != nil {
		d.Mongodb.HideSensitiveData()
	}
}

// Here and on every engine model the copy starts from the struct rather than a field list, so a
// field added later travels with it instead of being silently dropped; only identity, per-operation
// state and shared pointers are spelled out. Only the sub-model matching Type is carried over: the
// shallow copy would otherwise alias all five engine pointers onto the new row.
func (d *Database) CopyForNewDatabase() *Database {
	if d == nil {
		return nil
	}

	copiedDatabase := *d
	copiedDatabase.ID = uuid.Nil
	// The copy has no backup history of its own, and showing the source's would be a lie.
	copiedDatabase.LastBackupTime = nil
	copiedDatabase.LastBackupErrorMessage = nil
	copiedDatabase.Notifiers = slices.Clone(d.Notifiers)

	copiedDatabase.PostgresqlLogical = nil
	copiedDatabase.PostgresqlPhysical = nil
	copiedDatabase.Mysql = nil
	copiedDatabase.Mariadb = nil
	copiedDatabase.Mongodb = nil

	if d.HealthStatus != nil {
		copiedDatabase.HealthStatus = new(*d.HealthStatus)
	}

	switch d.Type {
	case DatabaseTypePostgresLogical:
		copiedDatabase.PostgresqlLogical = d.PostgresqlLogical.CopyForNewDatabase()
	case DatabaseTypePostgresPhysical:
		copiedDatabase.PostgresqlPhysical = d.PostgresqlPhysical.CopyForNewDatabase()
	case DatabaseTypeMysql:
		copiedDatabase.Mysql = d.Mysql.CopyForNewDatabase()
	case DatabaseTypeMariadb:
		copiedDatabase.Mariadb = d.Mariadb.CopyForNewDatabase()
	case DatabaseTypeMongodb:
		copiedDatabase.Mongodb = d.Mongodb.CopyForNewDatabase()
	}

	return &copiedDatabase
}

// The agent restores into a container it owns, so it never dials the source and has no use for the
// bastion's address or account.
func (d *Database) ClearSshTunnelConfig() {
	if d.PostgresqlLogical != nil {
		d.PostgresqlLogical.SshTunnel = sshtunnel.Config{}
	}

	if d.PostgresqlPhysical != nil {
		d.PostgresqlPhysical.SshTunnel = sshtunnel.Config{}
	}

	if d.Mysql != nil {
		d.Mysql.SshTunnel = sshtunnel.Config{}
	}

	if d.Mariadb != nil {
		d.Mariadb.SshTunnel = sshtunnel.Config{}
	}

	if d.Mongodb != nil {
		d.Mongodb.SshTunnel = sshtunnel.Config{}
	}
}

func (d *Database) EncryptSensitiveFields(encryptor encryption.FieldEncryptor) error {
	if d.PostgresqlLogical != nil {
		return d.PostgresqlLogical.EncryptSensitiveFields(encryptor)
	}
	if d.PostgresqlPhysical != nil {
		return d.PostgresqlPhysical.EncryptSensitiveFields(encryptor)
	}
	if d.Mysql != nil {
		return d.Mysql.EncryptSensitiveFields(encryptor)
	}
	if d.Mariadb != nil {
		return d.Mariadb.EncryptSensitiveFields(encryptor)
	}
	if d.Mongodb != nil {
		return d.Mongodb.EncryptSensitiveFields(encryptor)
	}
	return nil
}

func (d *Database) PopulateDbData(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	if d.PostgresqlLogical != nil {
		return d.PostgresqlLogical.PopulateDbData(logger, encryptor)
	}
	if d.PostgresqlPhysical != nil {
		return d.PostgresqlPhysical.PopulateDbData(logger, encryptor)
	}
	if d.Mysql != nil {
		return d.Mysql.PopulateDbData(logger, encryptor)
	}
	if d.Mariadb != nil {
		return d.Mariadb.PopulateDbData(logger, encryptor)
	}
	if d.Mongodb != nil {
		return d.Mongodb.PopulateDbData(logger, encryptor)
	}
	return nil
}

// Dispatches on Type rather than on the first non-nil sub-model, because OpenTunnel dispatches on
// Type too: a row carrying two engine configs would otherwise open the tunnel for one and connect
// with the other, straight past the bastion.
func (d *Database) PopulateVersion(
	logger *slog.Logger,
	encryptor encryption.FieldEncryptor,
) error {
	switch d.Type {
	case DatabaseTypePostgresLogical:
		if d.PostgresqlLogical == nil {
			return errors.New("postgresql logical config is not set")
		}

		return d.PostgresqlLogical.PopulateVersion(logger, encryptor)
	case DatabaseTypeMysql:
		if d.Mysql == nil {
			return errors.New("mysql config is not set")
		}

		return d.Mysql.PopulateVersion(logger, encryptor)
	case DatabaseTypeMariadb:
		if d.Mariadb == nil {
			return errors.New("mariadb config is not set")
		}

		return d.Mariadb.PopulateVersion(logger, encryptor)
	case DatabaseTypeMongodb:
		if d.Mongodb == nil {
			return errors.New("mongodb config is not set")
		}

		return d.Mongodb.PopulateVersion(logger, encryptor)
	default:
		return errors.New("version detection not supported for database type: " + string(d.Type))
	}
}

func (d *Database) Update(incoming *Database) {
	d.Name = incoming.Name
	d.Type = incoming.Type
	d.Notifiers = incoming.Notifiers

	switch d.Type {
	case DatabaseTypePostgresLogical:
		if d.PostgresqlLogical != nil && incoming.PostgresqlLogical != nil {
			d.PostgresqlLogical.Update(incoming.PostgresqlLogical)
		}
	case DatabaseTypePostgresPhysical:
		if d.PostgresqlPhysical != nil && incoming.PostgresqlPhysical != nil {
			d.PostgresqlPhysical.Update(incoming.PostgresqlPhysical)
		}
	case DatabaseTypeMysql:
		if d.Mysql != nil && incoming.Mysql != nil {
			d.Mysql.Update(incoming.Mysql)
		}
	case DatabaseTypeMariadb:
		if d.Mariadb != nil && incoming.Mariadb != nil {
			d.Mariadb.Update(incoming.Mariadb)
		}
	case DatabaseTypeMongodb:
		if d.Mongodb != nil && incoming.Mongodb != nil {
			d.Mongodb.Update(incoming.Mongodb)
		}
	}
}
