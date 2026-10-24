package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs "databasus-backend/internal/features/audit_logs"
	backups_backups_backuping_logical "databasus-backend/internal/features/backups/backups/backuping/logical"
	backups_backups_backuping_physical "databasus-backend/internal/features/backups/backups/backuping/physical"
	backups_backups_controllers_logical "databasus-backend/internal/features/backups/backups/controllers/logical"
	backups_backups_controllers_physical "databasus-backend/internal/features/backups/backups/controllers/physical"
	backups_backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_backups_core_physical_chain_view "databasus-backend/internal/features/backups/backups/core/physical/chain_view"
	backups_backups_core_physical_repositories "databasus-backend/internal/features/backups/backups/core/physical/repositories"
	backups_backups_core_physical_service "databasus-backend/internal/features/backups/backups/core/physical/service"
	backups_backups_download "databasus-backend/internal/features/backups/backups/download"
	backups_backups_services "databasus-backend/internal/features/backups/backups/services"
	backups_backups_usecases_logical "databasus-backend/internal/features/backups/backups/usecases/logical"
	backups_backups_usecases_logical_mariadb "databasus-backend/internal/features/backups/backups/usecases/logical/mariadb"
	backups_backups_usecases_logical_mongodb "databasus-backend/internal/features/backups/backups/usecases/logical/mongodb"
	backups_backups_usecases_logical_mysql "databasus-backend/internal/features/backups/backups/usecases/logical/mysql"
	backups_backups_usecases_logical_postgresql "databasus-backend/internal/features/backups/backups/usecases/logical/postgresql"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	backups_config_physical "databasus-backend/internal/features/backups/config/physical"
	databases "databasus-backend/internal/features/databases"
	disk "databasus-backend/internal/features/disk"
	email "databasus-backend/internal/features/email"
	encryption_secrets "databasus-backend/internal/features/encryption/secrets"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	notifiers "databasus-backend/internal/features/notifiers"
	restores "databasus-backend/internal/features/restores"
	restores_restoring "databasus-backend/internal/features/restores/restoring"
	restores_usecases "databasus-backend/internal/features/restores/usecases"
	restores_usecases_mariadb "databasus-backend/internal/features/restores/usecases/mariadb"
	restores_usecases_mongodb "databasus-backend/internal/features/restores/usecases/mongodb"
	restores_usecases_mysql "databasus-backend/internal/features/restores/usecases/mysql"
	restores_usecases_postgresql "databasus-backend/internal/features/restores/usecases/postgresql"
	storages "databasus-backend/internal/features/storages"
	system_agent "databasus-backend/internal/features/system/agent"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	system_version "databasus-backend/internal/features/system/version"
	tasks_cancellation "databasus-backend/internal/features/tasks/cancellation"
	telemetry "databasus-backend/internal/features/telemetry"
	users_controllers "databasus-backend/internal/features/users/controllers"
	users_repositories "databasus-backend/internal/features/users/repositories"
	users_services "databasus-backend/internal/features/users/services"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	workspaces_controllers "databasus-backend/internal/features/workspaces/controllers"
	workspaces_services "databasus-backend/internal/features/workspaces/services"
	util_cloudflare_turnstile "databasus-backend/internal/util/cloudflare_turnstile"
	util_encryption "databasus-backend/internal/util/encryption"
)

type oldCodeGetter struct {
	name string
	get  func() any
}

// getOldCodeGetters lists every public getter of the di.go files, so a getter that the bridge does
// not feed, or a cycle between lazy values, shows up right after New.
func getOldCodeGetters() []oldCodeGetter {
	return []oldCodeGetter{
		{"features/audit_logs.GetAuditLogService", func() any { return audit_logs.GetAuditLogService() }},
		{"features/audit_logs.GetAuditLogController", func() any { return audit_logs.GetAuditLogController() }},
		{
			"features/audit_logs.GetAuditLogBackgroundService",
			func() any { return audit_logs.GetAuditLogBackgroundService() },
		},
		{
			"features/backups/backups/backuping/logical.GetBackupsScheduler",
			func() any { return backups_backups_backuping_logical.GetBackupsScheduler() },
		},
		{
			"features/backups/backups/backuping/logical.GetBackuper",
			func() any { return backups_backups_backuping_logical.GetBackuper() },
		},
		{
			"features/backups/backups/backuping/logical.GetBackupCleaner",
			func() any { return backups_backups_backuping_logical.GetBackupCleaner() },
		},
		{
			"features/backups/backups/backuping/physical.GetPhysicalBackuper",
			func() any { return backups_backups_backuping_physical.GetPhysicalBackuper() },
		},
		{
			"features/backups/backups/backuping/physical.GetPhysicalBackupsScheduler",
			func() any { return backups_backups_backuping_physical.GetPhysicalBackupsScheduler() },
		},
		{
			"features/backups/backups/backuping/physical.GetPhysicalBackupCleaner",
			func() any { return backups_backups_backuping_physical.GetPhysicalBackupCleaner() },
		},
		{
			"features/backups/backups/backuping/physical.GetPhysicalWalStreamSupervisor",
			func() any { return backups_backups_backuping_physical.GetPhysicalWalStreamSupervisor() },
		},
		{
			"features/backups/backups/backuping/physical.GetPhysicalBackupCanceller",
			func() any { return backups_backups_backuping_physical.GetPhysicalBackupCanceller() },
		},
		{
			"features/backups/backups/controllers/logical.GetBackupController",
			func() any { return backups_backups_controllers_logical.GetBackupController() },
		},
		{
			"features/backups/backups/controllers/physical.GetPhysicalBackupController",
			func() any { return backups_backups_controllers_physical.GetPhysicalBackupController() },
		},
		{
			"features/backups/backups/core/logical.GetBackupRepository",
			func() any { return backups_backups_core_logical.GetBackupRepository() },
		},
		{
			"features/backups/backups/core/physical/chain_view.GetChainViewService",
			func() any { return backups_backups_core_physical_chain_view.GetChainViewService() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetFullBackupRepository",
			func() any { return backups_backups_core_physical_repositories.GetFullBackupRepository() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetIncrementalBackupRepository",
			func() any { return backups_backups_core_physical_repositories.GetIncrementalBackupRepository() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetInFlightBackupRepository",
			func() any { return backups_backups_core_physical_repositories.GetInFlightBackupRepository() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetWalSegmentRepository",
			func() any { return backups_backups_core_physical_repositories.GetWalSegmentRepository() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetWalHistoryRepository",
			func() any { return backups_backups_core_physical_repositories.GetWalHistoryRepository() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetWalStreamerRepository",
			func() any { return backups_backups_core_physical_repositories.GetWalStreamerRepository() },
		},
		{
			"features/backups/backups/core/physical/repositories.GetWalStreamSampleRepository",
			func() any { return backups_backups_core_physical_repositories.GetWalStreamSampleRepository() },
		},
		{
			"features/backups/backups/core/physical/service.GetPhysicalBackupService",
			func() any { return backups_backups_core_physical_service.GetPhysicalBackupService() },
		},
		{
			"features/backups/backups/download.GetDownloadTokenService",
			func() any { return backups_backups_download.GetDownloadTokenService() },
		},
		{
			"features/backups/backups/download.GetRestoreTokenService",
			func() any { return backups_backups_download.GetRestoreTokenService() },
		},
		{
			"features/backups/backups/download.GetDownloadTokenBackgroundService",
			func() any { return backups_backups_download.GetDownloadTokenBackgroundService() },
		},
		{
			"features/backups/backups/download.GetRestoreStreamWriter",
			func() any { return backups_backups_download.GetRestoreStreamWriter() },
		},
		{
			"features/backups/backups/services.GetBackupService",
			func() any { return backups_backups_services.GetBackupService() },
		},
		{
			"features/backups/backups/services.GetPhysicalBackupService",
			func() any { return backups_backups_services.GetPhysicalBackupService() },
		},
		{
			"features/backups/backups/usecases/logical.GetCreateBackupUsecase",
			func() any { return backups_backups_usecases_logical.GetCreateBackupUsecase() },
		},
		{
			"features/backups/backups/usecases/logical/mariadb.GetCreateMariadbBackupUsecase",
			func() any { return backups_backups_usecases_logical_mariadb.GetCreateMariadbBackupUsecase() },
		},
		{
			"features/backups/backups/usecases/logical/mongodb.GetCreateMongodbBackupUsecase",
			func() any { return backups_backups_usecases_logical_mongodb.GetCreateMongodbBackupUsecase() },
		},
		{
			"features/backups/backups/usecases/logical/mysql.GetCreateMysqlBackupUsecase",
			func() any { return backups_backups_usecases_logical_mysql.GetCreateMysqlBackupUsecase() },
		},
		{
			"features/backups/backups/usecases/logical/postgresql.GetCreatePostgresqlBackupUsecase",
			func() any { return backups_backups_usecases_logical_postgresql.GetCreatePostgresqlBackupUsecase() },
		},
		{
			"features/backups/config/logical.GetBackupConfigController",
			func() any { return backups_config_logical.GetBackupConfigController() },
		},
		{
			"features/backups/config/logical.GetBackupConfigService",
			func() any { return backups_config_logical.GetBackupConfigService() },
		},
		{
			"features/backups/config/physical.GetBackupConfigController",
			func() any { return backups_config_physical.GetBackupConfigController() },
		},
		{
			"features/backups/config/physical.GetBackupConfigService",
			func() any { return backups_config_physical.GetBackupConfigService() },
		},
		{"features/databases.GetDatabaseService", func() any { return databases.GetDatabaseService() }},
		{"features/databases.GetDatabaseController", func() any { return databases.GetDatabaseController() }},
		{"features/disk.GetDiskService", func() any { return disk.GetDiskService() }},
		{"features/disk.GetDiskController", func() any { return disk.GetDiskController() }},
		{"features/email.GetEmailSMTPSender", func() any { return email.GetEmailSMTPSender() }},
		{
			"features/encryption/secrets.GetSecretKeyService",
			func() any { return encryption_secrets.GetSecretKeyService() },
		},
		{
			"features/healthcheck/attempt.GetHealthcheckAttemptRepository",
			func() any { return healthcheck_attempt.GetHealthcheckAttemptRepository() },
		},
		{
			"features/healthcheck/attempt.GetHealthcheckAttemptService",
			func() any { return healthcheck_attempt.GetHealthcheckAttemptService() },
		},
		{
			"features/healthcheck/attempt.GetHealthcheckAttemptBackgroundService",
			func() any { return healthcheck_attempt.GetHealthcheckAttemptBackgroundService() },
		},
		{
			"features/healthcheck/attempt.GetHealthcheckAttemptController",
			func() any { return healthcheck_attempt.GetHealthcheckAttemptController() },
		},
		{
			"features/healthcheck/config.GetHealthcheckConfigService",
			func() any { return healthcheck_config.GetHealthcheckConfigService() },
		},
		{
			"features/healthcheck/config.GetHealthcheckConfigController",
			func() any { return healthcheck_config.GetHealthcheckConfigController() },
		},
		{"features/notifiers.GetNotifierController", func() any { return notifiers.GetNotifierController() }},
		{"features/notifiers.GetNotifierService", func() any { return notifiers.GetNotifierService() }},
		{"features/notifiers.GetNotifierRepository", func() any { return notifiers.GetNotifierRepository() }},
		{"features/restores.GetRestoreController", func() any { return restores.GetRestoreController() }},
		{
			"features/restores/restoring.GetRestoresScheduler",
			func() any { return restores_restoring.GetRestoresScheduler() },
		},
		{"features/restores/restoring.GetRestorer", func() any { return restores_restoring.GetRestorer() }},
		{
			"features/restores/usecases.GetRestoreBackupUsecase",
			func() any { return restores_usecases.GetRestoreBackupUsecase() },
		},
		{
			"features/restores/usecases/mariadb.GetRestoreMariadbBackupUsecase",
			func() any { return restores_usecases_mariadb.GetRestoreMariadbBackupUsecase() },
		},
		{
			"features/restores/usecases/mongodb.GetRestoreMongodbBackupUsecase",
			func() any { return restores_usecases_mongodb.GetRestoreMongodbBackupUsecase() },
		},
		{
			"features/restores/usecases/mysql.GetRestoreMysqlBackupUsecase",
			func() any { return restores_usecases_mysql.GetRestoreMysqlBackupUsecase() },
		},
		{
			"features/restores/usecases/postgresql.GetRestorePostgresqlBackupUsecase",
			func() any { return restores_usecases_postgresql.GetRestorePostgresqlBackupUsecase() },
		},
		{"features/storages.GetStorageService", func() any { return storages.GetStorageService() }},
		{"features/storages.GetStorageController", func() any { return storages.GetStorageController() }},
		{"features/storages.GetStorageFileStore", func() any { return storages.GetStorageFileStore() }},
		{
			"features/storages.GetStorageFileDeletionWorker",
			func() any { return storages.GetStorageFileDeletionWorker() },
		},
		{"features/system/agent.GetAgentController", func() any { return system_agent.GetAgentController() }},
		{
			"features/system/healthcheck.GetHealthcheckController",
			func() any { return system_healthcheck.GetHealthcheckController() },
		},
		{"features/system/version.GetVersionController", func() any { return system_version.GetVersionController() }},
		{"features/tasks/cancellation.GetRegistry", func() any { return tasks_cancellation.GetRegistry() }},
		{"features/tasks/cancellation.GetRequester", func() any { return tasks_cancellation.GetRequester() }},
		{"features/tasks/cancellation.GetListener", func() any { return tasks_cancellation.GetListener() }},
		{"features/telemetry.GetTelemetryService", func() any { return telemetry.GetTelemetryService() }},
		{
			"features/telemetry.GetTelemetryBackgroundService",
			func() any { return telemetry.GetTelemetryBackgroundService() },
		},
		{"features/users/controllers.GetUserController", func() any { return users_controllers.GetUserController() }},
		{
			"features/users/controllers.GetSettingsController",
			func() any { return users_controllers.GetSettingsController() },
		},
		{
			"features/users/controllers.GetManagementController",
			func() any { return users_controllers.GetManagementController() },
		},
		{"features/users/repositories.GetUserRepository", func() any { return users_repositories.GetUserRepository() }},
		{
			"features/users/repositories.GetUsersSettingsRepository",
			func() any { return users_repositories.GetUsersSettingsRepository() },
		},
		{
			"features/users/repositories.GetPasswordResetRepository",
			func() any { return users_repositories.GetPasswordResetRepository() },
		},
		{
			"features/users/repositories.GetTwoFactorRepository",
			func() any { return users_repositories.GetTwoFactorRepository() },
		},
		{"features/users/services.GetUserService", func() any { return users_services.GetUserService() }},
		{"features/users/services.GetSettingsService", func() any { return users_services.GetSettingsService() }},
		{"features/users/services.GetManagementService", func() any { return users_services.GetManagementService() }},
		{
			"features/users/services.GetSignInCodeBackgroundService",
			func() any { return users_services.GetSignInCodeBackgroundService() },
		},
		{"features/verification/agents.GetAgentService", func() any { return verification_agents.GetAgentService() }},
		{
			"features/verification/agents.GetAgentController",
			func() any { return verification_agents.GetAgentController() },
		},
		{
			"features/verification/agents.GetAgentFacingController",
			func() any { return verification_agents.GetAgentFacingController() },
		},
		{
			"features/verification/config.GetVerificationConfigService",
			func() any { return verification_config.GetVerificationConfigService() },
		},
		{
			"features/verification/config.GetVerificationConfigController",
			func() any { return verification_config.GetVerificationConfigController() },
		},
		{
			"features/verification/runs.GetVerificationService",
			func() any { return verification_runs.GetVerificationService() },
		},
		{
			"features/verification/runs.GetVerificationScheduler",
			func() any { return verification_runs.GetVerificationScheduler() },
		},
		{
			"features/verification/runs.GetVerificationController",
			func() any { return verification_runs.GetVerificationController() },
		},
		{
			"features/verification/runs.GetVerificationAgentController",
			func() any { return verification_runs.GetVerificationAgentController() },
		},
		{
			"features/workspaces/controllers.GetWorkspaceController",
			func() any { return workspaces_controllers.GetWorkspaceController() },
		},
		{
			"features/workspaces/controllers.GetMembershipController",
			func() any { return workspaces_controllers.GetMembershipController() },
		},
		{
			"features/workspaces/services.GetWorkspaceService",
			func() any { return workspaces_services.GetWorkspaceService() },
		},
		{
			"features/workspaces/services.GetMembershipService",
			func() any { return workspaces_services.GetMembershipService() },
		},
		{
			"util/cloudflare_turnstile.GetCloudflareTurnstileService",
			func() any { return util_cloudflare_turnstile.GetCloudflareTurnstileService() },
		},
		{"util/encryption.GetFieldEncryptor", func() any { return util_encryption.GetFieldEncryptor() }},
	}
}

func requireGetterReturnsValue(t *testing.T, getter oldCodeGetter) {
	t.Helper()

	returnedValue := make(chan any, 1)

	go func() { returnedValue <- getter.get() }()

	select {
	case value := <-returnedValue:
		require.NotNil(t, value)

		reflectedValue := reflect.ValueOf(value)
		if reflectedValue.Kind() == reflect.Pointer || reflectedValue.Kind() == reflect.Interface {
			assert.False(t, reflectedValue.IsNil(), "%s returned a nil value", getter.name)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("%s did not return within 10 seconds, likely a cycle between lazy values", getter.name)
	}
}
