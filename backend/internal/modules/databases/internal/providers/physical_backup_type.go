package providers

// PhysicalBackupType is what the physical backups of a physical PostgreSQL database take.
type PhysicalBackupType string

const (
	PhysicalBackupTypeFull                     PhysicalBackupType = "FULL"
	PhysicalBackupTypeFullAndIncremental       PhysicalBackupType = "FULL_INCREMENTAL"
	PhysicalBackupTypeFullIncrementalWalStream PhysicalBackupType = "FULL_INCREMENTAL_WAL_STREAM"
)

func (t PhysicalBackupType) IsValid() bool {
	switch t {
	case PhysicalBackupTypeFull, PhysicalBackupTypeFullAndIncremental, PhysicalBackupTypeFullIncrementalWalStream:
		return true
	default:
		return false
	}
}

// IsRequiringWalSummary is true when incremental backups need WAL summarization on the source.
func (t PhysicalBackupType) IsRequiringWalSummary() bool {
	return t == PhysicalBackupTypeFullAndIncremental || t == PhysicalBackupTypeFullIncrementalWalStream
}

func (t PhysicalBackupType) IsWalStreaming() bool {
	return t == PhysicalBackupTypeFullIncrementalWalStream
}
