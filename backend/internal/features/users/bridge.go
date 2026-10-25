// Package users_bridge hands the users module that internal/app built to the code that has not
// moved into internal/modules yet. A test binary built with the testhelpers tag builds the module
// on first use from the old access points instead.
package users_bridge

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/modules/users"
)

type installedModule struct {
	module users.Module
}

var installed atomic.Pointer[installedModule]

var getModuleWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(module users.Module) {
	installed.Store(&installedModule{module: module})
}

func GetModule() users.Module {
	if module := installed.Load(); module != nil {
		return module.module
	}

	return getModuleWithoutApp()
}
