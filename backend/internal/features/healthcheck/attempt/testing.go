package healthcheck_attempt

import (
	"time"

	"github.com/google/uuid"

	"databasus-backend/internal/features/databases"
)

func CreateTestHealthcheckAttempt(
	databaseID uuid.UUID,
	status databases.HealthStatus,
	createdAt time.Time,
) *HealthcheckAttempt {
	attempt := &HealthcheckAttempt{
		ID:         uuid.New(),
		DatabaseID: databaseID,
		Status:     status,
		CreatedAt:  createdAt,
	}

	if err := GetHealthcheckAttemptRepository().Create(attempt); err != nil {
		panic("Failed to create test healthcheck attempt: " + err.Error())
	}

	return attempt
}
