package walstreaming

import (
	"maps"
	"time"

	"github.com/google/uuid"
)

type chainAlertThrottle map[chainAlertKey]time.Time

type chainAlertAttempt struct {
	Key         chainAlertKey
	Now         time.Time
	MinInterval time.Duration
}

func (t chainAlertThrottle) RecordIfDue(attempt chainAlertAttempt) bool {
	if sentAt, wasNotified := t[attempt.Key]; wasNotified && attempt.Now.Sub(sentAt) < attempt.MinInterval {
		return false
	}
	t[attempt.Key] = attempt.Now
	return true
}

func (t chainAlertThrottle) ForgetDatabase(databaseID uuid.UUID) {
	maps.DeleteFunc(t, func(key chainAlertKey, _ time.Time) bool { return key.DatabaseID == databaseID })
}
