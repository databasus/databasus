//go:build testhelpers

package verification_config

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
)

// SetupDependencies subscribes the verification configs to the database copies of the module the
// bridge builds without the application, as internal/app does for the module it builds.
var SetupDependencies = sync.OnceFunc(func() {
	databases_bridge.GetModule().SubscribeToDatabaseCopied(getVerificationConfigService().OnDatabaseCopied)
})
