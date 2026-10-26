package restores_core

import (
	"context"
	"log/slog"
	"time"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
	"databasus-backend/internal/modules/databases"
)

const targetVersionDetectionTimeout = 15 * time.Second

type TargetConnectionSpec struct {
	Type           databases.Type
	Request        RestoreBackupRequest
	ClientToolsDir string
	Logger         *slog.Logger
}

// TargetConnection is the restore target reached through its SSH tunnel. Target holds the target
// as a client tool reaches it, so one tunnel serves version detection and the restore itself.
type TargetConnection struct {
	Target RestoreBackupRequest

	logger         *slog.Logger
	clientToolsDir string
	closeTunnel    func()
}

func OpenTargetConnection(ctx context.Context, spec TargetConnectionSpec) (*TargetConnection, error) {
	connection := &TargetConnection{
		Target:         RestoreBackupRequest{},
		logger:         spec.Logger,
		clientToolsDir: spec.ClientToolsDir,
		closeTunnel:    func() {},
	}

	switch spec.Type {
	case databases.TypePostgresqlLogical:
		if spec.Request.PostgresqlLogicalDatabase == nil {
			return nil, ErrRestoreTargetMissing
		}

		tunnel, err := pglogical.NewSource(spec.Request.PostgresqlLogicalDatabase.GetEngineSettings()).
			OpenTunnel(ctx, spec.Logger)
		if err != nil {
			return nil, err
		}

		connection.closeTunnel = tunnel.Close
		connection.Target.PostgresqlLogicalDatabase = spec.Request.PostgresqlLogicalDatabase.
			withConnection(tunnel.GetSource().GetSettings())
	case databases.TypeMysql:
		if spec.Request.MysqlDatabase == nil {
			return nil, ErrRestoreTargetMissing
		}

		tunnel, err := mysql_engine.NewSource(
			spec.Request.MysqlDatabase.GetEngineSettings(),
			mysql_engine.ClientTools{Root: spec.ClientToolsDir},
		).OpenTunnel(ctx, spec.Logger)
		if err != nil {
			return nil, err
		}

		connection.closeTunnel = tunnel.Close
		connection.Target.MysqlDatabase = spec.Request.MysqlDatabase.withConnection(tunnel.GetSource().GetSettings())
	case databases.TypeMariadb:
		if spec.Request.MariadbDatabase == nil {
			return nil, ErrRestoreTargetMissing
		}

		tunnel, err := mariadb_engine.NewSource(
			spec.Request.MariadbDatabase.GetEngineSettings(),
			mariadb_engine.ClientTools{Root: spec.ClientToolsDir},
		).OpenTunnel(ctx, spec.Logger)
		if err != nil {
			return nil, err
		}

		connection.closeTunnel = tunnel.Close
		connection.Target.MariadbDatabase = spec.Request.MariadbDatabase.withConnection(
			tunnel.GetSource().GetSettings(),
		)
	case databases.TypeMongodb:
		if spec.Request.MongodbDatabase == nil {
			return nil, ErrRestoreTargetMissing
		}

		tunnel, err := mongodb_engine.NewSource(spec.Request.MongodbDatabase.GetEngineSettings()).
			OpenTunnel(ctx, spec.Logger)
		if err != nil {
			return nil, err
		}

		connection.closeTunnel = tunnel.Close
		connection.Target.MongodbDatabase = spec.Request.MongodbDatabase.withConnection(
			tunnel.GetSource().GetSettings(),
		)
	default:
		return nil, ErrRestoreTypeNotSupported
	}

	return connection, nil
}

// DetectVersion asks the target server for its version and writes it into Target. A target
// without a database name keeps the version from the request: the restore refuses it later.
func (c *TargetConnection) DetectVersion(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, targetVersionDetectionTimeout)
	defer cancel()

	switch {
	case c.Target.PostgresqlLogicalDatabase != nil:
		target := c.Target.PostgresqlLogicalDatabase
		if getDatabaseName(target.Database) == "" {
			return nil
		}

		version, err := pglogical.NewSource(target.GetEngineSettings()).DetectVersion(ctx, c.logger)
		if err != nil {
			return err
		}

		target.Version = version
	case c.Target.MysqlDatabase != nil:
		target := c.Target.MysqlDatabase
		if getDatabaseName(target.Database) == "" {
			return nil
		}

		version, err := mysql_engine.NewSource(target.GetEngineSettings(), mysql_engine.ClientTools{Root: c.clientToolsDir}).
			DetectVersion(ctx, c.logger)
		if err != nil {
			return err
		}

		target.Version = version
	case c.Target.MariadbDatabase != nil:
		target := c.Target.MariadbDatabase
		if getDatabaseName(target.Database) == "" {
			return nil
		}

		version, err := mariadb_engine.NewSource(target.GetEngineSettings(), mariadb_engine.ClientTools{Root: c.clientToolsDir}).
			DetectVersion(ctx, c.logger)
		if err != nil {
			return err
		}

		target.Version = version
	case c.Target.MongodbDatabase != nil:
		target := c.Target.MongodbDatabase

		version, err := mongodb_engine.NewSource(target.GetEngineSettings()).DetectVersion(ctx, c.logger)
		if err != nil {
			return err
		}

		target.Version = version
	}

	return nil
}

func (c *TargetConnection) Close() {
	c.closeTunnel()
}
