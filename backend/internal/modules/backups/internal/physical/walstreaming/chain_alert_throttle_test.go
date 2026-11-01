package walstreaming

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChainAlertThrottle(t *testing.T) {
	t.Run("records separate incidents when the kinds or databases differ", func(t *testing.T) {
		throttle := make(chainAlertThrottle)
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		databaseID := uuid.New()
		incident := chainAlertAttempt{
			Key:         chainAlertKey{DatabaseID: databaseID, Kind: chainAlertChainAtRisk},
			Now:         now,
			MinInterval: time.Hour,
		}

		assert.True(t, throttle.RecordIfDue(incident))
		assert.False(t, throttle.RecordIfDue(incident))
		incident.Key.Kind = chainAlertSlotRebuilt
		assert.True(t, throttle.RecordIfDue(incident))
		incident.Key.DatabaseID = uuid.New()
		assert.True(t, throttle.RecordIfDue(incident))
	})
	t.Run("records an incident again when its throttle window expires", func(t *testing.T) {
		throttle := make(chainAlertThrottle)
		incident := chainAlertAttempt{
			Key:         chainAlertKey{DatabaseID: uuid.New(), Kind: chainAlertStreamerFailed},
			Now:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			MinInterval: time.Hour,
		}

		assert.True(t, throttle.RecordIfDue(incident))
		incident.Now = incident.Now.Add(time.Hour - time.Nanosecond)
		assert.False(t, throttle.RecordIfDue(incident))
		incident.Now = incident.Now.Add(time.Nanosecond)
		assert.True(t, throttle.RecordIfDue(incident))
	})
}

func TestForgetChainAlerts(t *testing.T) {
	t.Run("allows a new incident when a database streamer stops", func(t *testing.T) {
		throttle := make(chainAlertThrottle)
		incident := chainAlertAttempt{
			Key:         chainAlertKey{DatabaseID: uuid.New(), Kind: chainAlertStreamerFailed},
			Now:         time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			MinInterval: time.Hour,
		}
		otherIncident := incident
		otherIncident.Key.DatabaseID = uuid.New()
		assert.True(t, throttle.RecordIfDue(incident))
		assert.True(t, throttle.RecordIfDue(otherIncident))

		throttle.ForgetDatabase(incident.Key.DatabaseID)

		assert.True(t, throttle.RecordIfDue(incident))
		assert.False(t, throttle.RecordIfDue(otherIncident))
	})
}
