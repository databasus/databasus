package pipeline

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	mariadb_engine "databasus-backend/internal/engines/mariadb"
	mongodb_engine "databasus-backend/internal/engines/mongodb"
	mysql_engine "databasus-backend/internal/engines/mysql"
	pglogical "databasus-backend/internal/engines/postgresql/logical"
)

// RestoreTarget is the database a restore writes into: the connection settings of its engine and
// that engine's restore options. Its secrets are in clear text and live only in the memory of the
// running restore.
type RestoreTarget interface {
	isRestoreTarget()
}

type PostgresqlRestoreTarget struct {
	Settings            pglogical.Settings
	CPUCount            int
	IsExcludeExtensions bool
	IsRestoreOwnership  bool
	IsRestorePrivileges bool
}

type MysqlRestoreTarget struct {
	Settings mysql_engine.Settings
}

type MariadbRestoreTarget struct {
	Settings            mariadb_engine.Settings
	IsSkipGaleraDisable bool
}

type MongodbRestoreTarget struct {
	Settings mongodb_engine.Settings
	CPUCount int
}

func (PostgresqlRestoreTarget) isRestoreTarget() {}
func (MysqlRestoreTarget) isRestoreTarget()      {}
func (MariadbRestoreTarget) isRestoreTarget()    {}
func (MongodbRestoreTarget) isRestoreTarget()    {}

// restoreTargetSpec is what an engine needs to open a target: the target, the database the backup
// was taken from, whose settings name the dumped database and its version, and the backup's traits.
type restoreTargetSpec struct {
	newCleanupContext func(context.Context) (context.Context, context.CancelFunc)
	target            RestoreTarget
	sourceDatabaseID  uuid.UUID
	isTimescaleDB     bool
	logger            *slog.Logger
}
