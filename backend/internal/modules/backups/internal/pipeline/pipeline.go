// Package pipeline streams a logical dump into a storage and a stored backup back into a database,
// through the engine of the database's type, which only the registry picks.
package pipeline

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"databasus-backend/internal/lib/crypto"
	"databasus-backend/internal/lib/shutdown"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/storages"
)

type Deps struct {
	ShutdownBudget *shutdown.Budget
	Databases      connectionSettingsReader
	Storages       fileStorage
	MasterKey      *crypto.MasterKey
	// ClientToolsDir holds the client tool bundles of every engine.
	ClientToolsDir string
	// TempDir holds the archive of a PostgreSQL restore that needs it as a file.
	TempDir string
}

type Pipeline struct {
	shutdownBudget *shutdown.Budget
	engines        map[databases.Type]logicalEngine
	storages       fileStorage
	masterKey      *crypto.MasterKey
}

func New(deps Deps) *Pipeline {
	return &Pipeline{
		shutdownBudget: deps.ShutdownBudget,
		engines:        newEngineRegistry(deps),
		storages:       deps.Storages,
		masterKey:      deps.MasterKey,
	}
}

// IsLogical reports whether databases of the type have logical backups.
func (p *Pipeline) IsLogical(databaseType databases.Type) bool {
	_, isFound := p.engines[databaseType]

	return isFound
}

// GetFileExtension is the extension a downloaded backup file of the database type carries.
func (p *Pipeline) GetFileExtension(databaseType databases.Type) (string, error) {
	engine, err := p.getEngine(databaseType)
	if err != nil {
		return "", err
	}

	return engine.GetFileExtension(), nil
}

func (p *Pipeline) getEngine(databaseType databases.Type) (logicalEngine, error) {
	engine, isFound := p.engines[databaseType]
	if !isFound {
		return nil, ErrNotLogicalDatabase
	}

	return engine, nil
}

// connectionSettingsReader is what the engines read from the databases module: the decrypted
// settings of the database a backup dumps or a restore was taken from.
type connectionSettingsReader interface {
	GetPostgresqlLogicalConnectionSettings(
		ctx context.Context, databaseID uuid.UUID,
	) (databases.PostgresqlLogicalConnectionSettings, error)
	GetMysqlConnectionSettings(ctx context.Context, databaseID uuid.UUID) (databases.MysqlConnectionSettings, error)
	GetMariadbConnectionSettings(ctx context.Context, databaseID uuid.UUID) (databases.MariadbConnectionSettings, error)
	GetMongodbConnectionSettings(ctx context.Context, databaseID uuid.UUID) (databases.MongodbConnectionSettings, error)
}

// fileStorage is the file store and the storage reader of the storages module.
type fileStorage interface {
	StartBackgroundWrite(ctx context.Context, spec storages.BackgroundWriteSpec) storages.BackgroundWrite
	GetStorageReader(ctx context.Context, storageID uuid.UUID) (storages.StorageReader, error)
}

// logicalEngine is one database engine as the pipelines use it.
type logicalEngine interface {
	OpenDumpSource(ctx context.Context, databaseID uuid.UUID, logger *slog.Logger) (logicalDumpSource, error)
	// OpenRestoreTarget connects to the target through its tunnel, detects its server version and
	// refuses one that cannot take a backup of the source database.
	OpenRestoreTarget(ctx context.Context, spec restoreTargetSpec) (logicalRestoreSink, error)
	GetStoredCompression() storedCompression
	// GetFileExtension names the format of the stored file for a downloaded copy of it.
	GetFileExtension() string
}
