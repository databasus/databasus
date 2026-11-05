package attempts

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"databasus-backend/internal/modules/databases"
)

func TestHealthcheckTransition(t *testing.T) {
	t.Parallel()
	available := databases.HealthStatusAvailable
	unavailable := databases.HealthStatusUnavailable
	scenarios := []struct {
		name       string
		previous   *databases.HealthStatus
		status     databases.HealthStatus
		threshold  int
		history    []databases.HealthStatus
		transition *databases.HealthStatus
	}{
		{
			"keeps available with fewer failures than the threshold",
			&available,
			unavailable,
			3,
			[]databases.HealthStatus{unavailable, unavailable},
			nil,
		},
		{
			"becomes unavailable when the threshold is reached",
			&available,
			unavailable,
			3,
			[]databases.HealthStatus{unavailable, unavailable, unavailable},
			&unavailable,
		},
		{
			"keeps available when a success breaks consecutive failures",
			&available,
			unavailable,
			3,
			[]databases.HealthStatus{unavailable, available, unavailable},
			nil,
		},
		{
			"becomes unavailable with a threshold of one",
			&available,
			unavailable,
			1,
			[]databases.HealthStatus{unavailable},
			&unavailable,
		},
		{
			"keeps unavailable after another failure",
			&unavailable,
			unavailable,
			1,
			[]databases.HealthStatus{unavailable},
			nil,
		},
		{"becomes available after one success", &unavailable, available, 3, nil, &available},
		{"becomes available with no previous status", nil, available, 3, nil, &available},
		{"keeps available after another success", &available, available, 3, nil, nil},
		{
			"becomes unavailable with no previous status after the threshold",
			nil,
			unavailable,
			1,
			[]databases.HealthStatus{unavailable},
			&unavailable,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			history := make([]Attempt, len(scenario.history))
			for index, status := range scenario.history {
				history[index].Status = status
			}
			transition := getTransition(
				TransitionInput{
					PreviousStatus:                  scenario.previous,
					Status:                          scenario.status,
					FailedAttemptsBeforeUnavailable: scenario.threshold,
					RecentAttempts:                  history,
				},
			)
			assert.Equal(t, scenario.transition, transition)
		})
	}
}
