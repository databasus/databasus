// Package databases owns the user databases that Databasus backs up: their settings per type, their
// connection tests, the limited users created on them, the notifiers attached to them, and their last
// backup time, last backup error and health status. Other modules reach it only through this file.
package databases

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/lib/audit"
	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/database"
	"databasus-backend/internal/modules/databases/internal/connection"
	databaseshttpapi "databasus-backend/internal/modules/databases/internal/httpapi"
	"databasus-backend/internal/modules/databases/internal/limiteduser"
	"databasus-backend/internal/modules/databases/internal/module"
	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/providers/mariadb"
	"databasus-backend/internal/modules/databases/internal/providers/mongodb"
	"databasus-backend/internal/modules/databases/internal/providers/mysql"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqllogical"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqlphysical"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type (
	// Type is the kind of database. It is fixed when the database is created.
	Type = userdatabase.Type
	// HealthStatus is the outcome of the last scheduled health check, nil while health checks are off.
	HealthStatus = userdatabase.HealthStatus
	// PhysicalBackupType is what the physical backups of a physical PostgreSQL database take.
	PhysicalBackupType = providers.PhysicalBackupType
	// Database is a database as other modules see it, without its settings.
	Database = userdatabase.Database
	// TelemetryEntry is what the anonymous usage statistics report about one database.
	TelemetryEntry = userdatabase.TelemetryEntry

	// The connection settings carry the database's ID, the engine settings with every secret in clear
	// text and the options of the type. The caller opens the engine's tunnel and calls the engine.
	PostgresqlLogicalConnectionSettings  = providers.PostgresqlLogicalConnectionSettings
	PostgresqlPhysicalConnectionSettings = providers.PostgresqlPhysicalConnectionSettings
	MysqlConnectionSettings              = providers.MysqlConnectionSettings
	MariadbConnectionSettings            = providers.MariadbConnectionSettings
	MongodbConnectionSettings            = providers.MongodbConnectionSettings

	// DatabaseCreated is published inside the transaction that creates a database, a copy included.
	DatabaseCreated = userdatabase.Created
	// DatabaseCopied is published inside the copy's transaction, after DatabaseCreated of the copy.
	DatabaseCopied = userdatabase.Copied
	// DatabaseBackupTypeChanged is published inside the transaction that changes the physical backup
	// type of a physical PostgreSQL database.
	DatabaseBackupTypeChanged = userdatabase.BackupTypeChanged
	// DatabaseDeleting is published inside the deletion's transaction, while the database and its
	// settings still exist.
	DatabaseDeleting = userdatabase.Deleting

	// The handlers run in the publisher's transaction; an error rolls it back. An effect outside the
	// database registers itself with database.AfterCommit.
	DatabaseCreatedHandler           = userdatabase.CreatedHandler
	DatabaseCopiedHandler            = userdatabase.CopiedHandler
	DatabaseBackupTypeChangedHandler = userdatabase.BackupTypeChangedHandler
	DatabaseDeletingHandler          = userdatabase.DeletingHandler
)

// ErrDatabaseNotFound answers an operation on a database that does not exist or that the actor may
// not see.
var ErrDatabaseNotFound = userdatabase.ErrDatabaseNotFound

const (
	TypePostgresqlLogical  = providers.TypePostgresqlLogical
	TypePostgresqlPhysical = providers.TypePostgresqlPhysical
	TypeMysql              = providers.TypeMysql
	TypeMariadb            = providers.TypeMariadb
	TypeMongodb            = providers.TypeMongodb

	HealthStatusAvailable   = userdatabase.HealthStatusAvailable
	HealthStatusUnavailable = userdatabase.HealthStatusUnavailable

	PhysicalBackupTypeFull                     = providers.PhysicalBackupTypeFull
	PhysicalBackupTypeFullAndIncremental       = providers.PhysicalBackupTypeFullAndIncremental
	PhysicalBackupTypeFullIncrementalWalStream = providers.PhysicalBackupTypeFullIncrementalWalStream
)

type Module interface {
	RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks)

	// GetDatabases leaves out the IDs that match no database.
	GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]Database, error)
	// GetDatabaseForUpdate holds the row until the caller's transaction ends, so a healthcheck result,
	// its config update and database deletion all lock the database before the healthcheck rows.
	GetDatabaseForUpdate(ctx context.Context, databaseID uuid.UUID) (Database, bool, error)
	// GetPhysicalBackupTypes reads the types without the settings, so it decrypts nothing. It leaves out
	// the IDs that match no physical PostgreSQL database.
	GetPhysicalBackupTypes(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]PhysicalBackupType, error)
	GetPostgresqlLogicalVersions(ctx context.Context, databaseIDs []uuid.UUID) (map[uuid.UUID]string, error)
	ListAllDatabases(ctx context.Context) ([]Database, error)
	GetTelemetry(ctx context.Context) ([]TelemetryEntry, error)

	// The getters hand out the decrypted settings of one database of a known type. A database of
	// another type is an error, so a caller never switches on the type.
	GetPostgresqlLogicalConnectionSettings(
		ctx context.Context, databaseID uuid.UUID,
	) (PostgresqlLogicalConnectionSettings, error)
	GetPostgresqlPhysicalConnectionSettings(
		ctx context.Context, databaseID uuid.UUID,
	) (PostgresqlPhysicalConnectionSettings, error)
	GetMysqlConnectionSettings(ctx context.Context, databaseID uuid.UUID) (MysqlConnectionSettings, error)
	GetMariadbConnectionSettings(ctx context.Context, databaseID uuid.UUID) (MariadbConnectionSettings, error)
	GetMongodbConnectionSettings(ctx context.Context, databaseID uuid.UUID) (MongodbConnectionSettings, error)

	// ProbeLiveness is the trusted path of scheduled health checks; it has no actor.
	ProbeLiveness(ctx context.Context, databaseID uuid.UUID) error

	// The status writes change one column each, and refuse to run on a missing database. A success
	// never moves the last backup time backwards and clears the last backup error.
	LockDatabaseForBackup(ctx context.Context, databaseID uuid.UUID) error
	ResetBackupHistory(ctx context.Context, databaseID uuid.UUID) error
	RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, backupTime time.Time) error
	RecordBackupFailure(ctx context.Context, databaseID uuid.UUID, message string) error
	SetHealthStatus(ctx context.Context, databaseID uuid.UUID, status *HealthStatus) error

	// TransferDatabaseToWorkspace and ReplaceDatabaseNotifiers serve the database transfer of the backups
	// module, which checks access in both workspaces itself and runs them in its transaction.
	TransferDatabaseToWorkspace(ctx context.Context, actor users.Actor, databaseID, workspaceID uuid.UUID) error
	ReplaceDatabaseNotifiers(ctx context.Context, databaseID uuid.UUID, notifierIDs []uuid.UUID) error

	OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error
	OnNotifierTransferring(ctx context.Context, event notifiers.NotifierTransferring) error

	// The subscriptions are made by internal/app while it builds the application. The handlers run in
	// the order they subscribed.
	SubscribeToDatabaseCreated(handler DatabaseCreatedHandler)
	SubscribeToDatabaseCopied(handler DatabaseCopiedHandler)
	SubscribeToDatabaseBackupTypeChanged(handler DatabaseBackupTypeChangedHandler)
	SubscribeToDatabaseDeleting(handler DatabaseDeletingHandler)
}

type Deps struct {
	DB             *gorm.DB
	Transactor     *database.Transactor
	Logger         *slog.Logger
	AuditLog       *audit.Log
	FieldEncryptor *crypto.FieldEncryptor
	// ClientToolsDir holds the client tool bundles; the MySQL and MariaDB version detection refuses a
	// server version without one.
	ClientToolsDir string
	Users          users.Module
	Workspaces     workspaces.Module
	Notifiers      notifiers.Module
}

func New(deps Deps) Module {
	physicalDatabases := postgresqlphysical.New(postgresqlphysical.Deps{
		Repositories:   postgresqlphysical.NewRepositories(deps.DB, deps.FieldEncryptor),
		FieldEncryptor: deps.FieldEncryptor,
		Logger:         deps.Logger,
	})

	logicalDatabases := postgresqllogical.New(
		postgresqllogical.Deps{
			Repositories:   postgresqllogical.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			Logger:         deps.Logger,
		},
	)
	registry := providers.NewRegistry(
		logicalDatabases,
		physicalDatabases,
		mysql.New(mysql.Deps{
			Repositories:   mysql.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			Logger:         deps.Logger,
			ClientToolsDir: deps.ClientToolsDir,
		}),
		mariadb.New(mariadb.Deps{
			Repositories:   mariadb.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			Logger:         deps.Logger,
			ClientToolsDir: deps.ClientToolsDir,
		}),
		mongodb.New(mongodb.Deps{
			Repositories:   mongodb.NewRepositories(deps.DB, deps.FieldEncryptor),
			FieldEncryptor: deps.FieldEncryptor,
			Logger:         deps.Logger,
		}),
	)

	databaseService := userdatabase.NewService(userdatabase.Deps{
		Repositories: userdatabase.NewRepositories(deps.DB),
		Registry:     registry,
		Transactor:   deps.Transactor,
		Logger:       deps.Logger,
		AuditLog:     deps.AuditLog,
		Users:        deps.Users,
		Workspaces:   deps.Workspaces,
		Notifiers:    deps.Notifiers,
	})

	connectionService := connection.NewService(connection.Deps{
		Databases: databaseService,
		Registry:  registry,
		Logger:    deps.Logger,
	})

	return module.New(module.Deps{
		Databases:         databaseService,
		Connections:       connectionService,
		PhysicalDatabases: physicalDatabases,
		LogicalDatabases:  logicalDatabases,
		Handler: databaseshttpapi.NewHandler(databaseshttpapi.Deps{
			Databases:   databaseService,
			Connections: connectionService,
			LimitedUsers: limiteduser.NewService(limiteduser.Deps{
				Databases: databaseService,
				Registry:  registry,
				Logger:    deps.Logger,
			}),
		}),
	})
}
