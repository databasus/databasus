package report

type DatabaseEntry struct {
	Type               string                     `json:"type"`
	Version            string                     `json:"version"`
	BackupType         string                     `json:"backupType,omitzero"`
	IsSSHTunnelEnabled bool                       `json:"isSshTunnelEnabled"`
	RawSizeMb          int64                      `json:"rawSizeMb,omitzero"`
	BackupSizeMb       int64                      `json:"backupSizeMb,omitzero"`
	Verification       *DatabaseVerificationEntry `json:"verification,omitempty"`
}

type DatabaseVerificationEntry struct {
	IsEnabled    bool   `json:"isEnabled"`
	ScheduleType string `json:"scheduleType"`
	IntervalType string `json:"intervalType,omitempty"`
}

type VerificationAgentEntry struct {
	MaxCPU            int `json:"maxCpu"`
	MaxRAMMb          int `json:"maxRamMb"`
	MaxDiskGb         int `json:"maxDiskGb"`
	MaxConcurrentJobs int `json:"maxConcurrentJobs"`
}

type CollectRequest struct {
	InstanceID         string                   `json:"instanceID"`
	AppVersion         string                   `json:"appVersion"`
	OS                 string                   `json:"os"`
	Arch               string                   `json:"arch"`
	InstalledAt        string                   `json:"installedAt,omitempty"`
	UserCount          int                      `json:"userCount,omitzero"`
	Databases          []DatabaseEntry          `json:"databases"`
	Storages           []string                 `json:"storages"`
	Notifiers          []string                 `json:"notifiers"`
	VerificationAgents []VerificationAgentEntry `json:"verificationAgents"`
}
