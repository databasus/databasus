//go:build testhelpers

package healthcheck_config

import (
	"sync"

	databases_bridge "databasus-backend/internal/features/databases"
)

// SetupDependencies subscribes the health check configs to the database creations of the module the
// bridge builds without the application, as internal/app does for the module it builds.
var SetupDependencies = sync.OnceFunc(func() {
	databases_bridge.GetModule().SubscribeToDatabaseCreated(getHealthcheckConfigService().OnDatabaseCreated)
})
