package walmath

const (
	DefaultSegmentSizeBytes  = uint64(16 * 1024 * 1024)
	walFileFormat            = "%08X%08X%08X"
	walHistoryFileFormat     = "%08X.history"
	hexUint32Bits            = 32
	hexUint64Bits            = 64
	walSegmentFilenameLength = 24
)
