package databases

import (
	"context"
	"log/slog"

	"databasus-backend/internal/features/databases/databases/mariadb"
	"databasus-backend/internal/features/databases/databases/mongodb"
	mysql_database "databasus-backend/internal/features/databases/databases/mysql"
	postgresql_logical "databasus-backend/internal/features/databases/databases/postgresql/logical"
	postgresql_physical "databasus-backend/internal/features/databases/databases/postgresql/physical"
	"databasus-backend/internal/util/encryption"
)

type OpenTunnelSpec struct {
	Database  *Database
	Logger    *slog.Logger
	Encryptor encryption.FieldEncryptor
}

// GetDatabaseThroughTunnel is deliberately absent: every engine hands back its own model type, and
// the swap happens inside the type switch anyway.
type engineTunnel interface {
	CopyDiscoveredMetadataToOriginal()
	Close()
}

type TunneledDatabase struct {
	databaseThroughTunnel *Database
	tunneledEngine        engineTunnel
}

// Ctx bounds opening the tunnel; the forwarder then lives until Close.
func OpenTunnel(ctx context.Context, spec OpenTunnelSpec) (*TunneledDatabase, error) {
	if spec.Database == nil {
		return &TunneledDatabase{}, nil
	}

	if err := validatePostgresqlTarget(spec.Database); err != nil {
		return nil, err
	}

	// The copy carries the forwarded port and a disabled tunnel, so it must never reach Save.
	databaseThroughTunnel := *spec.Database

	tunneledDatabase := &TunneledDatabase{databaseThroughTunnel: &databaseThroughTunnel}

	var err error

	switch spec.Database.Type {
	case DatabaseTypePostgresLogical:
		err = tunneledDatabase.openPostgresqlLogicalTunnel(ctx, spec)
	case DatabaseTypePostgresPhysical:
		err = tunneledDatabase.openPostgresqlPhysicalTunnel(ctx, spec)
	case DatabaseTypeMysql:
		err = tunneledDatabase.openMysqlTunnel(ctx, spec)
	case DatabaseTypeMariadb:
		err = tunneledDatabase.openMariadbTunnel(ctx, spec)
	case DatabaseTypeMongodb:
		err = tunneledDatabase.openMongodbTunnel(ctx, spec)
	}

	if err != nil {
		return nil, err
	}

	if tunneledDatabase.tunneledEngine == nil {
		tunneledDatabase.databaseThroughTunnel = spec.Database
	}

	return tunneledDatabase, nil
}

func validatePostgresqlTarget(database *Database) error {
	switch database.Type {
	case DatabaseTypePostgresLogical:
		if database.PostgresqlLogical != nil {
			return database.PostgresqlLogical.ValidateNotEmbeddedTarget()
		}
	case DatabaseTypePostgresPhysical:
		if database.PostgresqlPhysical != nil {
			return database.PostgresqlPhysical.ValidateNotEmbeddedTarget()
		}
	}

	return nil
}

func (t *TunneledDatabase) GetDatabaseThroughTunnel() *Database {
	if t == nil {
		return nil
	}

	return t.databaseThroughTunnel
}

func (t *TunneledDatabase) CopyDiscoveredMetadataToOriginal() {
	if t == nil || t.tunneledEngine == nil {
		return
	}

	t.tunneledEngine.CopyDiscoveredMetadataToOriginal()
}

func (t *TunneledDatabase) Close() {
	if t == nil || t.tunneledEngine == nil {
		return
	}

	t.tunneledEngine.Close()
}

// A nil tunneledEngine is what makes OpenTunnel hand back spec.Database rather than the copy.
func (t *TunneledDatabase) openPostgresqlLogicalTunnel(ctx context.Context, spec OpenTunnelSpec) error {
	if spec.Database.PostgresqlLogical == nil || !spec.Database.PostgresqlLogical.SshTunnel.IsEnabled {
		return nil
	}

	tunneledEngine, err := postgresql_logical.OpenTunnel(ctx, postgresql_logical.OpenTunnelSpec{
		Database:  spec.Database.PostgresqlLogical,
		Logger:    spec.Logger,
		Encryptor: spec.Encryptor,
	})
	if err != nil {
		return err
	}

	t.tunneledEngine = tunneledEngine
	t.databaseThroughTunnel.PostgresqlLogical = tunneledEngine.GetDatabaseThroughTunnel()

	return nil
}

func (t *TunneledDatabase) openPostgresqlPhysicalTunnel(ctx context.Context, spec OpenTunnelSpec) error {
	if spec.Database.PostgresqlPhysical == nil || !spec.Database.PostgresqlPhysical.SshTunnel.IsEnabled {
		return nil
	}

	tunneledEngine, err := postgresql_physical.OpenTunnel(ctx, postgresql_physical.OpenTunnelSpec{
		Database:  spec.Database.PostgresqlPhysical,
		Logger:    spec.Logger,
		Encryptor: spec.Encryptor,
	})
	if err != nil {
		return err
	}

	t.tunneledEngine = tunneledEngine
	t.databaseThroughTunnel.PostgresqlPhysical = tunneledEngine.GetDatabaseThroughTunnel()

	return nil
}

func (t *TunneledDatabase) openMysqlTunnel(ctx context.Context, spec OpenTunnelSpec) error {
	if spec.Database.Mysql == nil || !spec.Database.Mysql.SshTunnel.IsEnabled {
		return nil
	}

	tunneledEngine, err := mysql_database.OpenTunnel(ctx, mysql_database.OpenTunnelSpec{
		Database:  spec.Database.Mysql,
		Logger:    spec.Logger,
		Encryptor: spec.Encryptor,
	})
	if err != nil {
		return err
	}

	t.tunneledEngine = tunneledEngine
	t.databaseThroughTunnel.Mysql = tunneledEngine.GetDatabaseThroughTunnel()

	return nil
}

func (t *TunneledDatabase) openMariadbTunnel(ctx context.Context, spec OpenTunnelSpec) error {
	if spec.Database.Mariadb == nil || !spec.Database.Mariadb.SshTunnel.IsEnabled {
		return nil
	}

	tunneledEngine, err := mariadb.OpenTunnel(ctx, mariadb.OpenTunnelSpec{
		Database:  spec.Database.Mariadb,
		Logger:    spec.Logger,
		Encryptor: spec.Encryptor,
	})
	if err != nil {
		return err
	}

	t.tunneledEngine = tunneledEngine
	t.databaseThroughTunnel.Mariadb = tunneledEngine.GetDatabaseThroughTunnel()

	return nil
}

func (t *TunneledDatabase) openMongodbTunnel(ctx context.Context, spec OpenTunnelSpec) error {
	if spec.Database.Mongodb == nil || !spec.Database.Mongodb.SshTunnel.IsEnabled {
		return nil
	}

	tunneledEngine, err := mongodb.OpenTunnel(ctx, mongodb.OpenTunnelSpec{
		Database:  spec.Database.Mongodb,
		Logger:    spec.Logger,
		Encryptor: spec.Encryptor,
	})
	if err != nil {
		return err
	}

	t.tunneledEngine = tunneledEngine
	t.databaseThroughTunnel.Mongodb = tunneledEngine.GetDatabaseThroughTunnel()

	return nil
}
