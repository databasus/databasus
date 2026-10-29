//go:build testhelpers

package verification_config

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
)

// Test bridges build their own modules, so their subscriptions live separately from app wiring.
var SetupDependencies = sync.OnceFunc(func() {
	databases_bridge.GetModule().SubscribeToDatabaseCopied(getVerificationConfigService().OnDatabaseCopied)
	databases_bridge.GetModule().SubscribeToDatabaseDeleting(getVerificationConfigService().OnDatabaseDeleting)
})
