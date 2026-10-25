// Package workspaces_bridge hands the workspaces module that internal/app built to the code that has
// not moved into internal/modules yet. A test binary built with the testhelpers tag builds the
// module on first use from the old access points instead.
package workspaces_bridge

import (
	"sync"
	"sync/atomic"

	"databasus-backend/internal/modules/workspaces"
)

type installedModule struct {
	module workspaces.Module
}

var installed atomic.Pointer[installedModule]

var getModuleWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(module workspaces.Module) {
	installed.Store(&installedModule{module: module})
}

func GetModule() workspaces.Module {
	if module := installed.Load(); module != nil {
		return module.module
	}

	return getModuleWithoutApp()
}
