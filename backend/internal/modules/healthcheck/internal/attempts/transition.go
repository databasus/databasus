package attempts

import "databasus-backend/internal/modules/databases"

type TransitionInput struct {
	PreviousStatus                  *databases.HealthStatus
	Status                          databases.HealthStatus
	FailedAttemptsBeforeUnavailable int
	RecentAttempts                  []Attempt
}

func getTransition(input TransitionInput) *databases.HealthStatus {
	if input.PreviousStatus != nil && *input.PreviousStatus == input.Status {
		return nil
	}

	if input.Status == databases.HealthStatusAvailable {
		return &input.Status
	}

	if len(input.RecentAttempts) < input.FailedAttemptsBeforeUnavailable {
		return nil
	}

	for _, attempt := range input.RecentAttempts[:input.FailedAttemptsBeforeUnavailable] {
		if attempt.Status != databases.HealthStatusUnavailable {
			return nil
		}
	}

	return &input.Status
}
