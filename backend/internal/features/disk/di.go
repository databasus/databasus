package disk

import "sync"

var getDiskService = sync.OnceValue(func() *DiskService {
	return &DiskService{}
})

var getDiskController = sync.OnceValue(func() *DiskController {
	return &DiskController{
		getDiskService(),
	}
})

func GetDiskService() *DiskService {
	return getDiskService()
}

func GetDiskController() *DiskController {
	return getDiskController()
}
