package attempts

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/modules/databases"
)

type Attempt struct {
	ID         uuid.UUID
	DatabaseID uuid.UUID
	Status     databases.HealthStatus
	CreatedAt  time.Time
}

func (Attempt) TableName() string { return "healthcheck_attempts" }

type ListInput struct {
	DatabaseID uuid.UUID
	Since      time.Time
	Limit      int
	After      *string
}
type Page struct {
	Items      []Attempt
	NextCursor *string
}
