// Package module implements the databases.Module contract on top of the module's services.
package module

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"databasus-backend/internal/lib/apierror"
	"databasus-backend/internal/modules/databases/internal/connection"
	databaseshttpapi "databasus-backend/internal/modules/databases/internal/httpapi"
	"databasus-backend/internal/modules/databases/internal/providers"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqllogical"
	"databasus-backend/internal/modules/databases/internal/providers/postgresqlphysical"
	"databasus-backend/internal/modules/databases/internal/userdatabase"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/users"
	"databasus-backend/internal/modules/workspaces"
)

type Deps struct {
	Databases         *userdatabase.Service
	Connections       *connection.Service
	PhysicalDatabases *postgresqlphysical.Provider
	LogicalDatabases  *postgresqllogical.Provider
	Handler           *databaseshttpapi.Handler
}

type Module struct {
	databases         *userdatabase.Service
	connections       *connection.Service
	physicalDatabases *postgresqlphysical.Provider
	logicalDatabases  *postgresqllogical.Provider
	handler           *databaseshttpapi.Handler
}

func New(deps Deps) *Module {
	return &Module{
		databases:         deps.Databases,
		connections:       deps.Connections,
		physicalDatabases: deps.PhysicalDatabases,
		logicalDatabases:  deps.LogicalDatabases,
		handler:           deps.Handler,
	}
}

func (m *Module) RegisterRoutes(routes gin.IRouter, hooks *apierror.Hooks) {
	databaseshttpapi.RegisterHandlersWithOptions(
		routes,
		databaseshttpapi.NewStrictHandlerWithOptions(m.handler, nil, databaseshttpapi.StrictGinServerOptions{
			RequestErrorHandlerFunc:  hooks.HandleRequestError,
			HandlerErrorFunc:         hooks.HandleHandlerError,
			ResponseErrorHandlerFunc: hooks.HandleResponseError,
		}),
		databaseshttpapi.GinServerOptions{BaseURL: "", Middlewares: nil, ErrorHandler: hooks.HandleParameterError},
	)
}

func (m *Module) GetDatabases(ctx context.Context, databaseIDs []uuid.UUID) ([]userdatabase.Database, error) {
	return m.databases.GetDatabases(ctx, databaseIDs)
}

func (m *Module) GetDatabaseForUpdate(ctx context.Context, databaseID uuid.UUID) (userdatabase.Database, bool, error) {
	return m.databases.GetDatabaseForUpdate(ctx, databaseID)
}

func (m *Module) GetPhysicalBackupTypes(
	ctx context.Context,
	databaseIDs []uuid.UUID,
) (map[uuid.UUID]providers.PhysicalBackupType, error) {
	return m.physicalDatabases.GetBackupTypes(ctx, databaseIDs)
}

func (m *Module) ListAllDatabases(ctx context.Context) ([]userdatabase.Database, error) {
	return m.databases.ListAllDatabases(ctx)
}

func (m *Module) GetTelemetry(ctx context.Context) ([]userdatabase.TelemetryEntry, error) {
	return m.databases.GetTelemetry(ctx)
}

func (m *Module) GetPostgresqlLogicalConnectionSettings(
	ctx context.Context,
	databaseID uuid.UUID,
) (providers.PostgresqlLogicalConnectionSettings, error) {
	return getConnectionSettings[providers.PostgresqlLogicalConnectionSettings](ctx, m.databases, databaseID)
}

func (m *Module) GetPostgresqlPhysicalConnectionSettings(
	ctx context.Context,
	databaseID uuid.UUID,
) (providers.PostgresqlPhysicalConnectionSettings, error) {
	return getConnectionSettings[providers.PostgresqlPhysicalConnectionSettings](ctx, m.databases, databaseID)
}

func (m *Module) GetMysqlConnectionSettings(
	ctx context.Context,
	databaseID uuid.UUID,
) (providers.MysqlConnectionSettings, error) {
	return getConnectionSettings[providers.MysqlConnectionSettings](ctx, m.databases, databaseID)
}

func (m *Module) GetMariadbConnectionSettings(
	ctx context.Context,
	databaseID uuid.UUID,
) (providers.MariadbConnectionSettings, error) {
	return getConnectionSettings[providers.MariadbConnectionSettings](ctx, m.databases, databaseID)
}

func (m *Module) GetMongodbConnectionSettings(
	ctx context.Context,
	databaseID uuid.UUID,
) (providers.MongodbConnectionSettings, error) {
	return getConnectionSettings[providers.MongodbConnectionSettings](ctx, m.databases, databaseID)
}

func (m *Module) ProbeLiveness(ctx context.Context, databaseID uuid.UUID) error {
	return m.connections.ProbeLiveness(ctx, databaseID)
}

func (m *Module) RecordBackupSuccess(ctx context.Context, databaseID uuid.UUID, backupTime time.Time) error {
	return m.databases.RecordBackupSuccess(ctx, databaseID, backupTime)
}

func (m *Module) RecordBackupFailure(ctx context.Context, databaseID uuid.UUID, message string) error {
	return m.databases.RecordBackupFailure(ctx, databaseID, message)
}

func (m *Module) SetHealthStatus(
	ctx context.Context,
	databaseID uuid.UUID,
	status *userdatabase.HealthStatus,
) error {
	return m.databases.SetHealthStatus(ctx, databaseID, status)
}

func (m *Module) TransferDatabaseToWorkspace(
	ctx context.Context,
	actor users.Actor,
	databaseID, workspaceID uuid.UUID,
) error {
	return m.databases.TransferDatabaseToWorkspace(ctx, actor, databaseID, workspaceID)
}

func (m *Module) ReplaceDatabaseNotifiers(ctx context.Context, databaseID uuid.UUID, notifierIDs []uuid.UUID) error {
	return m.databases.ReplaceDatabaseNotifiers(ctx, databaseID, notifierIDs)
}

func (m *Module) OnWorkspaceDeleting(ctx context.Context, event workspaces.WorkspaceDeleting) error {
	return m.databases.OnWorkspaceDeleting(ctx, event)
}

func (m *Module) OnNotifierTransferring(ctx context.Context, event notifiers.NotifierTransferring) error {
	return m.databases.OnNotifierTransferring(ctx, event)
}

func (m *Module) SubscribeToDatabaseCreated(handler userdatabase.CreatedHandler) {
	m.databases.SubscribeToCreated(handler)
}

func (m *Module) SubscribeToDatabaseCopied(handler userdatabase.CopiedHandler) {
	m.databases.SubscribeToCopied(handler)
}

func (m *Module) SubscribeToDatabaseBackupTypeChanged(handler userdatabase.BackupTypeChangedHandler) {
	m.databases.SubscribeToBackupTypeChanged(handler)
}

func (m *Module) SubscribeToDatabaseDeleting(handler userdatabase.DeletingHandler) {
	m.databases.SubscribeToDeleting(handler)
}

// getConnectionSettings refuses a database of another type than S, so a caller never switches on the
// type.
func getConnectionSettings[S providers.ConnectionSettings](
	ctx context.Context,
	databases *userdatabase.Service,
	databaseID uuid.UUID,
) (S, error) {
	connectionSettings, err := databases.GetConnectionSettings(ctx, databaseID)
	if err != nil {
		var zeroSettings S

		return zeroSettings, err
	}

	return providers.CastConnectionSettings[S](connectionSettings)
}

func (m *Module) GetPostgresqlLogicalVersions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	return m.logicalDatabases.GetVersions(ctx, ids)
}

func (m *Module) LockDatabaseForBackup(ctx context.Context, databaseID uuid.UUID) error {
	return m.databases.LockDatabaseForBackup(ctx, databaseID)
}

func (m *Module) ResetBackupHistory(ctx context.Context, databaseID uuid.UUID) error {
	return m.databases.ResetBackupHistory(ctx, databaseID)
}
