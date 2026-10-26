// Package storages_bridge hands the storages module that internal/app built to the code that has not
// moved into internal/modules yet. A test binary built with the testhelpers tag builds the module on
// first use from the old access points instead.
package storages_bridge

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/modules/storages"
)

type installedModule struct {
	module storages.Module
}

var installed atomic.Pointer[installedModule]

var getModuleWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(module storages.Module) {
	installed.Store(&installedModule{module: module})
}

func GetModule() storages.Module {
	if module := installed.Load(); module != nil {
		return module.module
	}

	return getModuleWithoutApp()
}
