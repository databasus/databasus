// Package databases_bridge hands the databases module that internal/app built to the code that has not
// moved into internal/modules yet. A test binary built with the testhelpers tag builds the module on
// first use from the old access points instead.
package databases_bridge

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/modules/databases"
)

type installedModule struct {
	module databases.Module
}

var installed atomic.Pointer[installedModule]

var getModuleWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(module databases.Module) {
	installed.Store(&installedModule{module: module})
}

func GetModule() databases.Module {
	if module := installed.Load(); module != nil {
		return module.module
	}

	return getModuleWithoutApp()
}
