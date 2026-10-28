package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	audit_logs "databasus-backend/internal/features/audit_logs"
	caching "databasus-backend/internal/features/caching"
	databases_bridge "databasus-backend/internal/features/databases"
	disk "databasus-backend/internal/features/disk"
	email "databasus-backend/internal/features/email"
	healthcheck_attempt "databasus-backend/internal/features/healthcheck/attempt"
	healthcheck_config "databasus-backend/internal/features/healthcheck/config"
	system_agent "databasus-backend/internal/features/system/agent"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	telemetry "databasus-backend/internal/features/telemetry"
	users_bridge "databasus-backend/internal/features/users"
	verification_agents "databasus-backend/internal/features/verification/agents"
	verification_config "databasus-backend/internal/features/verification/config"
	verification_runs "databasus-backend/internal/features/verification/runs"
	workspaces_bridge "databasus-backend/internal/features/workspaces"
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
		{"features/databases.GetAccess", func() any { return databases_bridge.GetAccess() }},
		{"features/disk.GetDiskService", func() any { return disk.GetDiskService() }},
		{"features/disk.GetDiskController", func() any { return disk.GetDiskController() }},
		{"features/caching.GetStore", func() any { return caching.GetStore() }},
		{"features/caching.GetRateLimiter", func() any { return caching.GetRateLimiter() }},
		{"features/email.GetInstanceSender", func() any { return email.GetInstanceSender() }},
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
		{"features/system/agent.GetAgentController", func() any { return system_agent.GetAgentController() }},
		{
			"features/system/healthcheck.GetHealthcheckController",
			func() any { return system_healthcheck.GetHealthcheckController() },
		},
		{"features/telemetry.GetTelemetryService", func() any { return telemetry.GetTelemetryService() }},
		{
			"features/telemetry.GetTelemetryBackgroundService",
			func() any { return telemetry.GetTelemetryBackgroundService() },
		},
		{"features/users.GetModule", func() any { return users_bridge.GetModule() }},
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
		{"features/workspaces.GetModule", func() any { return workspaces_bridge.GetModule() }},
		{"features/workspaces.GetAccess", func() any { return workspaces_bridge.GetAccess() }},
		{
			"util/cloudflare_turnstile.GetCloudflareTurnstileService",
			func() any { return util_cloudflare_turnstile.GetCloudflareTurnstileService() },
		},
		{"util/encryption.GetFieldEncryptor", func() any { return util_encryption.GetFieldEncryptor() }},
		{"util/encryption.GetMasterKey", func() any { return util_encryption.GetMasterKey() }},
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
