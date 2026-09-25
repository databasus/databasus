package dashboard

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/storages"
)

type WorkspaceDashboard struct {
	Databases []DashboardDatabase `json:"databases"`
	Totals    DashboardTotals     `json:"totals"`
}

type DashboardDatabase struct {
	ID                        uuid.UUID                     `json:"id"`
	Name                      string                        `json:"name"`
	Type                      databases.DatabaseType        `json:"type"`
	HealthStatus              *databases.HealthStatus       `json:"healthStatus,omitempty"`
	LastBackupTime            *time.Time                    `json:"lastBackupTime,omitempty"`
	LastBackupErrorMessage    *string                       `json:"lastBackupErrorMessage,omitempty"`
	Storage                   *DashboardStorage             `json:"storage,omitempty"`
	BackupsCount              int64                         `json:"backupsCount"`
	CompletedBackupsCount     int64                         `json:"completedBackupsCount"`
	FailedBackupsCount        int64                         `json:"failedBackupsCount"`
	MeanBackupSizeMb          float64                       `json:"meanBackupSizeMb"`
	TotalBackupSizeMb         float64                       `json:"totalBackupSizeMb"`
	RecentHealthcheckAttempts []DashboardHealthcheckAttempt `json:"recentHealthcheckAttempts"`
}

type DashboardStorage struct {
	ID   uuid.UUID            `json:"id"`
	Name string               `json:"name"`
	Type storages.StorageType `json:"type"`
}

type DashboardHealthcheckAttempt struct {
	Status    databases.HealthStatus `json:"status"`
	CreatedAt time.Time              `json:"createdAt"`
}

type DashboardTotals struct {
	DatabasesCount    int64   `json:"databasesCount"`
	BackupsCount      int64   `json:"backupsCount"`
	TotalBackupSizeMb float64 `json:"totalBackupSizeMb"`
}
