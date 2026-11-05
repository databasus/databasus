package agents

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID        uuid.UUID `gorm:"column:id"`
	Name      string    `gorm:"column:name"`
	TokenHash string    `gorm:"column:token_hash"`

	MaxCPU            int `gorm:"column:max_cpu"`
	MaxRAMMb          int `gorm:"column:max_ram_mb"`
	MaxDiskGb         int `gorm:"column:max_disk_gb"`
	MaxConcurrentJobs int `gorm:"column:max_concurrent_jobs"`

	LastSeenAt *time.Time `gorm:"column:last_seen_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	DeletedAt  *time.Time `gorm:"column:deleted_at"`
}

func (Agent) TableName() string {
	return "verification_agents"
}

func (a *Agent) IsDeleted() bool {
	return a.DeletedAt != nil
}

type AgentCapacity struct {
	MaxCPU            int
	MaxRAMMb          int
	MaxDiskGb         int
	MaxConcurrentJobs int
}
