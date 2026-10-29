package app

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	caching "databasus-backend/internal/features/caching"
	disk "databasus-backend/internal/features/disk"
	email "databasus-backend/internal/features/email"
	system_healthcheck "databasus-backend/internal/features/system/healthcheck"
	telemetry "databasus-backend/internal/features/telemetry"
	users_bridge "databasus-backend/internal/features/users"
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
		{"features/disk.GetDiskService", func() any { return disk.GetDiskService() }},
		{"features/disk.GetDiskController", func() any { return disk.GetDiskController() }},
		{"features/caching.GetStore", func() any { return caching.GetStore() }},
		{"features/caching.GetRateLimiter", func() any { return caching.GetRateLimiter() }},
		{"features/email.GetInstanceSender", func() any { return email.GetInstanceSender() }},
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
		{"features/workspaces.GetModule", func() any { return workspaces_bridge.GetModule() }},
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
