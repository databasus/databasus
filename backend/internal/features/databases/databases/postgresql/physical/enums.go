package postgresql_physical

type BackupType string

const (
	BackupTypeFullOnly                    BackupType = "FULL"
	BackupTypeFullAndIncremental          BackupType = "FULL_INCREMENTAL"
	BackupTypeFullIncrementalAndWalStream BackupType = "FULL_INCREMENTAL_WAL_STREAM"
)

func (t BackupType) IsRequireWalSummary() bool {
	return t == BackupTypeFullAndIncremental || t == BackupTypeFullIncrementalAndWalStream
}

func (t BackupType) IsWalStreaming() bool {
	return t == BackupTypeFullIncrementalAndWalStream
}
