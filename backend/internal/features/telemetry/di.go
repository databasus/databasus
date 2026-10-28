package telemetry

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/config"
	backups_bridge "databasus-backend/internal/features/backups/config"
	databases_bridge "databasus-backend/internal/features/databases"
	notifiers_bridge "databasus-backend/internal/features/notifiers"
	storages_bridge "databasus-backend/internal/features/storages"
	system_version "databasus-backend/internal/features/system/version"
	users_bridge "databasus-backend/internal/features/users"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	"databasus-backend/internal/modules/backups"
	"databasus-backend/internal/modules/databases"
	"databasus-backend/internal/modules/notifiers"
	"databasus-backend/internal/modules/storages"
	"databasus-backend/internal/util/logger"
)

const productionEndpoint = "https://metrics.databasus.com/api/anonymous/collect"

var (
	getTelemetryLogger = sync.OnceValue(logger.GetLogger)

	getInstanceLoader = sync.OnceValue(func() *InstanceFileLoader {
		return NewInstanceFileLoader("", getTelemetryLogger())
	})
	getHttpSender = sync.OnceValue(func() *HTTPTelemetrySender {
		return NewHTTPTelemetrySender(productionEndpoint, system_version.GetAppVersion())
	})

	getTelemetryService = sync.OnceValue(func() *TelemetryService {
		return NewTelemetryService(
			getInstanceLoader(),
			getHttpSender(),
			installedDatabases{},
			installedStorageTypes{},
			installedNotifierTypes{},
			installedLogicalBackups{},
			installedPhysicalBackups{},
			verification_agents.GetAgentService(),
			verification_config.GetVerificationConfigService(),
			installedUserCounter{},
			system_version.GetAppVersion(),
			getTelemetryLogger(),
		)
	})

	getTelemetryBackgroundService = sync.OnceValue(func() *TelemetryBackgroundService {
		return NewTelemetryBackgroundService(
			getTelemetryService(),
			getTelemetryLogger(),
		)
	})
)

func GetTelemetryService() *TelemetryService {
	return getTelemetryService()
}

func GetTelemetryBackgroundService() *TelemetryBackgroundService {
	return getTelemetryBackgroundService()
}

var SetupDependencies = sync.OnceFunc(func() {
	getInstanceLoader().path = config.GetEnv().TelemetryInstancePath
})

// installedLogicalBackups reads the module per call: each test application installs its own module.
type installedLogicalBackups struct{}

func (installedLogicalBackups) HasSuccessfulLogicalBackupSince(
	ctx context.Context,
	databaseID uuid.UUID,
	since time.Time,
) (bool, error) {
	return backups_bridge.GetModule().HasSuccessfulLogicalBackupSince(ctx, databaseID, since)
}

func (installedLogicalBackups) GetLatestCompletedLogicalBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) (backups.LogicalBackup, bool, error) {
	return backups_bridge.GetModule().GetLatestCompletedLogicalBackup(ctx, databaseID)
}

// installedUserCounter reads the module per call: each test application installs its own module.
type installedUserCounter struct{}

func (installedUserCounter) GetUserCount(ctx context.Context) (int64, error) {
	return users_bridge.GetModule().GetUserCount(ctx)
}

// installedDatabases reads the module per call: each test application installs its own module.
type installedDatabases struct{}

func (installedDatabases) GetTelemetry(ctx context.Context) ([]databases.TelemetryEntry, error) {
	return databases_bridge.GetModule().GetTelemetry(ctx)
}

// installedStorageTypes reads the module per call: each test application installs its own module.
type installedStorageTypes struct{}

func (installedStorageTypes) GetStorageTypes(ctx context.Context) ([]storages.Type, error) {
	return storages_bridge.GetModule().GetStorageTypes(ctx)
}

// installedNotifierTypes reads the module per call: each test application installs its own module.
type installedNotifierTypes struct{}

func (installedNotifierTypes) GetNotifierTypes(ctx context.Context) ([]notifiers.Type, error) {
	return notifiers_bridge.GetModule().GetNotifierTypes(ctx)
}

type installedPhysicalBackups struct{}

func (installedPhysicalBackups) GetLatestCompletedPhysicalFullBackup(
	ctx context.Context,
	databaseID uuid.UUID,
) (*backups.PhysicalFullBackup, error) {
	return backups_bridge.GetModule().GetLatestCompletedPhysicalFullBackup(ctx, databaseID)
}
