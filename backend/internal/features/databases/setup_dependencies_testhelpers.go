//go:build testhelpers

package databases

import (
	"sync"

	notifiers_bridge "databasus-backend/internal/features/notifiers"
)

// SetupDependencies subscribes the database service to the notifier transfers of the module the
// bridge builds without the application, as internal/app does for the module it builds.
var SetupDependencies = sync.OnceFunc(func() {
	notifiers_bridge.GetModule().SubscribeToNotifierTransferring(getDatabaseService().OnNotifierTransferring)
})
