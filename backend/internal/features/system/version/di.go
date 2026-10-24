package system_version

import "sync"

var getVersionController = sync.OnceValue(func() *VersionController {
	return &VersionController{}
})

func GetVersionController() *VersionController {
	return getVersionController()
}
