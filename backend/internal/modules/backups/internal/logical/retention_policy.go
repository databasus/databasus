package logical

import (
	"time"

	"databasus-backend/internal/lib/gfs"
	"databasus-backend/internal/modules/backups/internal/config"
)

// recentBackupShield keeps every backup younger than this, whatever the policy says, so a backup is
// never removed right after it was taken.
const recentBackupShield = time.Hour

// getExpiredBackups returns the backups of one database that the retention no longer keeps.
// databaseBackups holds every backup of the database, newest first.
func getExpiredBackups(
	retention config.LogicalRetention,
	databaseBackups []LogicalBackup,
	now time.Time,
) []LogicalBackup {
	var droppedBackups []LogicalBackup

	switch retention.Policy {
	case config.LogicalRetentionPolicyCount:
		droppedBackups = getDroppedByCount(retention.Count, databaseBackups)
	case config.LogicalRetentionPolicyGfs:
		droppedBackups = getDroppedByGfs(retention.Gfs, databaseBackups)
	case config.LogicalRetentionPolicyTimePeriod:
		droppedBackups = getDroppedByTimePeriod(retention.Period, databaseBackups, now)
	}

	expiredBackups := make([]LogicalBackup, 0, len(droppedBackups))

	for _, droppedBackup := range droppedBackups {
		if now.Sub(droppedBackup.CreatedAt) >= recentBackupShield {
			expiredBackups = append(expiredBackups, droppedBackup)
		}
	}

	return expiredBackups
}

// getDroppedByTimePeriod drops every backup older than the period. FOREVER keeps every backup.
func getDroppedByTimePeriod(
	period config.RetentionPeriod,
	databaseBackups []LogicalBackup,
	now time.Time,
) []LogicalBackup {
	periodDuration := period.GetDuration()
	if periodDuration == 0 {
		return nil
	}

	cutoff := now.Add(-periodDuration)

	var droppedBackups []LogicalBackup

	for _, databaseBackup := range databaseBackups {
		if databaseBackup.CreatedAt.Before(cutoff) {
			droppedBackups = append(droppedBackups, databaseBackup)
		}
	}

	return droppedBackups
}

// getDroppedByCount keeps the newest completed backups up to the count and drops the older completed
// ones. Failed and cancelled backups are not counted and stay.
func getDroppedByCount(count int, databaseBackups []LogicalBackup) []LogicalBackup {
	if count <= 0 {
		return nil
	}

	completedBackups := getCompletedBackups(databaseBackups)
	if len(completedBackups) <= count {
		return nil
	}

	return completedBackups[count:]
}

// getDroppedByGfs drops the completed backups that no grandfather-father-son tier keeps. Failed and
// cancelled backups stay. A retention without any tier keeps everything rather than dropping it all.
func getDroppedByGfs(gfsRetention config.GfsRetention, databaseBackups []LogicalBackup) []LogicalBackup {
	if !gfsRetention.HasAnyPeriod() {
		return nil
	}

	completedBackups := getCompletedBackups(databaseBackups)

	gfsItems := make([]gfs.Item, 0, len(completedBackups))
	for _, completedBackup := range completedBackups {
		gfsItems = append(gfsItems, gfs.Item{ID: completedBackup.ID, CreatedAt: completedBackup.CreatedAt})
	}

	retainedTiers := gfs.GetItemsToRetain(gfsItems,
		gfsRetention.Hours, gfsRetention.Days, gfsRetention.Weeks, gfsRetention.Months, gfsRetention.Years)

	var droppedBackups []LogicalBackup

	for _, completedBackup := range completedBackups {
		if len(retainedTiers[completedBackup.ID]) == 0 {
			droppedBackups = append(droppedBackups, completedBackup)
		}
	}

	return droppedBackups
}

func getCompletedBackups(databaseBackups []LogicalBackup) []LogicalBackup {
	var completedBackups []LogicalBackup

	for _, databaseBackup := range databaseBackups {
		if databaseBackup.Status == StatusCompleted {
			completedBackups = append(completedBackups, databaseBackup)
		}
	}

	return completedBackups
}
