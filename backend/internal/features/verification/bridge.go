package verification_bridge

import (
	"sync/atomic"

	"databasus-backend/internal/modules/verification"
)

type installedModule struct{ module verification.Module }

var installed atomic.Pointer[installedModule]

func Install(module verification.Module) { installed.Store(&installedModule{module: module}) }
func GetModule() verification.Module     { return installed.Load().module }
