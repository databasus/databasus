package physical_repositories

import "sync"

var (
	getFullBackupRepository = sync.OnceValue(func() *PhysicalFullBackupRepository {
		return &PhysicalFullBackupRepository{}
	})
	getIncrementalBackupRepository = sync.OnceValue(func() *PhysicalIncrementalBackupRepository {
		return &PhysicalIncrementalBackupRepository{}
	})
	getInFlightBackupRepository = sync.OnceValue(func() *PhysicalInFlightBackupRepository {
		return &PhysicalInFlightBackupRepository{}
	})
	getWalSegmentRepository = sync.OnceValue(func() *PhysicalWalSegmentRepository {
		return &PhysicalWalSegmentRepository{}
	})
	getWalHistoryRepository = sync.OnceValue(func() *PhysicalWalHistoryRepository {
		return &PhysicalWalHistoryRepository{}
	})
	getWalStreamerRepository = sync.OnceValue(func() *PhysicalWalStreamerRepository {
		return &PhysicalWalStreamerRepository{}
	})
	getWalStreamSampleRepository = sync.OnceValue(func() *PhysicalWalStreamSampleRepository {
		return &PhysicalWalStreamSampleRepository{}
	})
)

func GetFullBackupRepository() *PhysicalFullBackupRepository {
	return getFullBackupRepository()
}

func GetIncrementalBackupRepository() *PhysicalIncrementalBackupRepository {
	return getIncrementalBackupRepository()
}

func GetInFlightBackupRepository() *PhysicalInFlightBackupRepository {
	return getInFlightBackupRepository()
}

func GetWalSegmentRepository() *PhysicalWalSegmentRepository {
	return getWalSegmentRepository()
}

func GetWalHistoryRepository() *PhysicalWalHistoryRepository {
	return getWalHistoryRepository()
}

func GetWalStreamerRepository() *PhysicalWalStreamerRepository {
	return getWalStreamerRepository()
}

func GetWalStreamSampleRepository() *PhysicalWalStreamSampleRepository {
	return getWalStreamSampleRepository()
}
