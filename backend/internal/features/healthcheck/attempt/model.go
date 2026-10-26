package healthcheck_attempt

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
)

type HealthcheckAttempt struct {
	ID         uuid.UUID              `json:"id"         gorm:"column:id;primaryKey;type:uuid;default:gen_random_uuid()"`
	DatabaseID uuid.UUID              `json:"databaseId" gorm:"column:database_id;type:uuid;not null"`
	Status     databases.HealthStatus `json:"status"     gorm:"column:status;type:text;not null"`
	CreatedAt  time.Time              `json:"createdAt"  gorm:"column:created_at;type:timestamp with time zone;not null"`
}

func (h *HealthcheckAttempt) TableName() string {
	return "healthcheck_attempts"
}
