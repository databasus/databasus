package httpapi

import (
	"fmt"

	"databasus-backend/internal/modules/backups/internal/config"
)

// The contract spells the periods out, because its enum values start with a letter; the stored values
// keep the digits they have always had.
var storedRetentionPeriods = map[RetentionPeriod]config.RetentionPeriod{
	RetentionPeriodDAY:         config.RetentionPeriodDay,
	RetentionPeriodWEEK:        config.RetentionPeriodWeek,
	RetentionPeriodMONTH:       config.RetentionPeriodMonth,
	RetentionPeriodTHREEMONTHS: config.RetentionPeriod3Months,
	RetentionPeriodSIXMONTHS:   config.RetentionPeriod6Months,
	RetentionPeriodYEAR:        config.RetentionPeriodYear,
	RetentionPeriodTWOYEARS:    config.RetentionPeriod2Years,
	RetentionPeriodTHREEYEARS:  config.RetentionPeriod3Years,
	RetentionPeriodFOURYEARS:   config.RetentionPeriod4Years,
	RetentionPeriodFIVEYEARS:   config.RetentionPeriod5Years,
	RetentionPeriodFOREVER:     config.RetentionPeriodForever,
}

// toRetentionPeriod passes an unknown value through, so the config's validation names the field.
func toRetentionPeriod(body RetentionPeriod) config.RetentionPeriod {
	storedPeriod, isKnown := storedRetentionPeriods[body]
	if !isKnown {
		return config.RetentionPeriod(body)
	}

	return storedPeriod
}

func fromRetentionPeriod(storedPeriod config.RetentionPeriod) (RetentionPeriod, error) {
	for bodyPeriod, knownPeriod := range storedRetentionPeriods {
		if knownPeriod == storedPeriod {
			return bodyPeriod, nil
		}
	}

	return "", fmt.Errorf("%w: retention period %q", errUnknownStoredValue, storedPeriod)
}
