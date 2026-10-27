// Package backups_bridge hands the backups module that internal/app built to the code that has not
// moved into internal/modules yet. A test binary built with the testhelpers tag builds the module on
// first use from the old access points instead.
package backups_bridge

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/lib/background"
	"databasus-backend/internal/modules/backups"
)

type installedModule struct {
	module backups.Module
}

var installed atomic.Pointer[installedModule]

var getModuleWithoutApp = sync.OnceValue(loadWithoutApp)

// The registry is one per process, like the old code that registers in it: internal/app hands it to the
// backups module and the shutdown, and the old physical code reads it here, so every test application
// of a process shares the instance the old singletons captured.
var getCancellationRegistry = sync.OnceValue(background.NewCancellationRegistry)

func Install(module backups.Module) {
	installed.Store(&installedModule{module: module})
}

func GetModule() backups.Module {
	if module := installed.Load(); module != nil {
		return module.module
	}

	return getModuleWithoutApp()
}

func GetCancellationRegistry() *background.CancellationRegistry {
	return getCancellationRegistry()
}
