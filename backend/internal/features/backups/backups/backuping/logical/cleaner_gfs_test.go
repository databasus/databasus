package backuping_logical

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	backups_core_logical "databasus-backend/internal/features/backups/backups/core/logical"
	backups_config_logical "databasus-backend/internal/features/backups/config/logical"
	"databasus-backend/internal/features/databases"
	"databasus-backend/internal/features/notifiers"
	"databasus-backend/internal/features/storages"
	users_testing "databasus-backend/internal/features/users/testing"
	workspaces_testing "databasus-backend/internal/features/workspaces/testing"
	"databasus-backend/internal/modules/users"
)

func Test_BuildGFSKeepSet(t *testing.T) {
	// Fixed reference time: a Wednesday mid-month to avoid boundary edge cases in the default tests.
	// Use time.Date for determinism across test runs.
	ref := time.Date(2025, 6, 18, 12, 0, 0, 0, time.UTC) // Wednesday, 2025-06-18

	day := 24 * time.Hour
	week := 7 * day

	newBackup := func(createdAt time.Time) *backups_core_logical.LogicalBackup {
		return &backups_core_logical.LogicalBackup{ID: uuid.New(), CreatedAt: createdAt}
	}

	// backupsEveryDay returns n backups, newest-first, each 1 day apart.
	backupsEveryDay := func(n int) []*backups_core_logical.LogicalBackup {
		bs := make([]*backups_core_logical.LogicalBackup, n)
		for i := range n {
			bs[i] = newBackup(ref.Add(-time.Duration(i) * day))
		}
		return bs
	}

	// backupsEveryWeek returns n backups, newest-first, each 7 days apart.
	backupsEveryWeek := func(n int) []*backups_core_logical.LogicalBackup {
		bs := make([]*backups_core_logical.LogicalBackup, n)
		for i := range n {
			bs[i] = newBackup(ref.Add(-time.Duration(i) * week))
		}
		return bs
	}

	hour := time.Hour

	// backupsEveryHour returns n backups, newest-first, each 1 hour apart.
	backupsEveryHour := func(n int) []*backups_core_logical.LogicalBackup {
		bs := make([]*backups_core_logical.LogicalBackup, n)
		for i := range n {
			bs[i] = newBackup(ref.Add(-time.Duration(i) * hour))
		}
		return bs
	}

	// backupsEveryMonth returns n backups, newest-first, each ~1 month apart.
	backupsEveryMonth := func(n int) []*backups_core_logical.LogicalBackup {
		bs := make([]*backups_core_logical.LogicalBackup, n)
		for i := range n {
			bs[i] = newBackup(ref.AddDate(0, -i, 0))
		}
		return bs
	}

	// backupsEveryYear returns n backups, newest-first, each 1 year apart.
	backupsEveryYear := func(n int) []*backups_core_logical.LogicalBackup {
		bs := make([]*backups_core_logical.LogicalBackup, n)
		for i := range n {
			bs[i] = newBackup(ref.AddDate(-i, 0, 0))
		}
		return bs
	}

	tests := []struct {
		name         string
		backups      []*backups_core_logical.LogicalBackup
		hours        int
		days         int
		weeks        int
		months       int
		years        int
		keptIndices  []int   // which indices in backups should be kept
		deletedRange *[2]int // optional: all indices in [from, to) must be deleted
	}{
		{
			name:        "OnlyHourlySlots_KeepsNewest3Of5",
			backups:     backupsEveryHour(5),
			hours:       3,
			keptIndices: []int{0, 1, 2},
		},
		{
			name: "SameHourDedup_OnlyNewestKeptForHourlySlot",
			backups: []*backups_core_logical.LogicalBackup{
				newBackup(ref.Truncate(hour).Add(45 * time.Minute)),
				newBackup(ref.Truncate(hour).Add(10 * time.Minute)),
			},
			hours:       1,
			keptIndices: []int{0},
		},
		{
			name:        "OnlyDailySlots_KeepsNewest3Of5",
			backups:     backupsEveryDay(5),
			days:        3,
			keptIndices: []int{0, 1, 2},
		},
		{
			name:        "OnlyDailySlots_FewerBackupsThanSlots_KeepsAll",
			backups:     backupsEveryDay(2),
			days:        5,
			keptIndices: []int{0, 1},
		},
		{
			name:        "OnlyWeeklySlots_KeepsNewest2Weeks",
			backups:     backupsEveryWeek(4),
			weeks:       2,
			keptIndices: []int{0, 1},
		},
		{
			name: "OnlyMonthlySlots_KeepsNewest2Months",
			backups: []*backups_core_logical.LogicalBackup{
				newBackup(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2025, 5, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2025, 4, 1, 12, 0, 0, 0, time.UTC)),
			},
			months:      2,
			keptIndices: []int{0, 1},
		},
		{
			name: "OnlyYearlySlots_KeepsNewest2Years",
			backups: []*backups_core_logical.LogicalBackup{
				newBackup(time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)),
			},
			years:       2,
			keptIndices: []int{0, 1},
		},
		{
			name: "SameDayDedup_OnlyNewestKeptForDailySlot",
			backups: []*backups_core_logical.LogicalBackup{
				// Two backups on the same day; newest-first order
				newBackup(ref.Truncate(day).Add(10 * time.Hour)),
				newBackup(ref.Truncate(day).Add(2 * time.Hour)),
			},
			days:        1,
			keptIndices: []int{0},
		},
		{
			name: "SameWeekDedup_OnlyNewestKeptForWeeklySlot",
			backups: []*backups_core_logical.LogicalBackup{
				// ref is Wednesday; add Thursday of same week
				newBackup(ref.Add(1 * day)), // Thursday same week
				newBackup(ref),              // Wednesday same week
			},
			weeks:       1,
			keptIndices: []int{0},
		},
		{
			name: "AdditiveSlots_NewestFillsDailyAndWeeklyAndMonthly",
			// Newest backup fills daily + weekly + monthly simultaneously
			backups: []*backups_core_logical.LogicalBackup{
				newBackup(time.Date(2025, 6, 18, 12, 0, 0, 0, time.UTC)), // newest
				newBackup(time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)), // 1 week ago
				newBackup(time.Date(2025, 5, 18, 12, 0, 0, 0, time.UTC)), // 1 month ago
				newBackup(time.Date(2025, 4, 18, 12, 0, 0, 0, time.UTC)), // 2 months ago
			},
			days:        1,
			weeks:       2,
			months:      2,
			keptIndices: []int{0, 1, 2},
		},
		{
			name: "YearBoundary_CorrectlySplitsAcrossYears",
			backups: []*backups_core_logical.LogicalBackup{
				newBackup(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)),
				newBackup(time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)),
			},
			years:       2,
			keptIndices: []int{0, 1}, // 2025 and 2024 kept; 2024-06 and 2023 deleted
		},
		{
			name: "ISOWeekBoundary_Jan1UsesCorrectISOWeek",
			// 2025-01-01 is ISO week 1 of 2025; 2024-12-28 is ISO week 52 of 2024
			backups: []*backups_core_logical.LogicalBackup{
				newBackup(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),   // ISO week 2025-W01
				newBackup(time.Date(2024, 12, 28, 12, 0, 0, 0, time.UTC)), // ISO week 2024-W52
			},
			weeks:       2,
			keptIndices: []int{0, 1}, // different ISO weeks → both kept
		},
		{
			name:        "EmptyBackups_ReturnsEmptyKeepSet",
			backups:     []*backups_core_logical.LogicalBackup{},
			hours:       3,
			days:        3,
			weeks:       2,
			months:      1,
			years:       1,
			keptIndices: []int{},
		},
		{
			name:        "AllZeroSlots_KeepsNothing",
			backups:     backupsEveryDay(5),
			hours:       0,
			days:        0,
			weeks:       0,
			months:      0,
			years:       0,
			keptIndices: []int{},
		},
		{
			name:    "AllSlotsActive_FullCombination",
			backups: backupsEveryWeek(12),
			days:    2,
			weeks:   3,
			months:  2,
			years:   1,
			// 2 daily (indices 0,1) + 3rd weekly slot (index 2) + 2nd monthly slot (index 3 or later).
			// Additive slots: newest fills daily+weekly+monthly+yearly; each subsequent week fills another weekly,
			// and a backup ~4 weeks later fills the 2nd monthly slot.
			keptIndices: []int{0, 1, 2, 3},
		},
		{
			name:    "RealisticGFS_20DailyBackups_7Days4Weeks1Month",
			backups: backupsEveryDay(20),
			days:    7,
			weeks:   4,
			months:  1,
			// 7 daily: indices 0-6 (Jun 18-12). Index 0 also fills week 25 + month 2025-06.
			// Index 3 (Jun 15, Sun) fills week 24. Indices 7-9 are week 24 (seen) → not kept.
			// Index 10 (Jun 8, Sun) fills week 23. Indices 11-16 are week 23 (seen) → not kept.
			// Index 17 (Jun 1, Sun) fills week 22. Index 18-19 are week 22 (seen) → not kept.
			// Monthly slot 1 filled by index 0 (2025-06). Total kept: 9.
			keptIndices: []int{0, 1, 2, 3, 4, 5, 6, 10, 17},
		},

		// Cross-level absorption tests: when backup frequency is lower than slot granularity,
		// each backup gets a unique key at the higher-frequency level, filling slots that
		// should only cover recent time periods. This causes excess backups to be retained.

		// Adjacent-level absorption:
		{
			// Daily backups with hourly slots: each daily backup has a unique hour key
			// (different date-hour combo), so 23 backups fill 23/24 hourly slots.
			// Hourly slots should only cover the most recent 24 hours, not span weeks.
			name:    "HourlyAbsorbsDaily_DailyBackupsWithHourlySlots_KeepsTooMany",
			backups: backupsEveryDay(23),
			hours:   24,
			days:    7,
			weeks:   4,
			months:  12,
			years:   3,
			// Correct behavior: 10 kept (7 daily + 2 extra weekly + 1 monthly for 2025-05).
			// Index 18 (May 31) is the first backup in month "2025-05" within the 12-month window.
			// Bug: all 23 kept because each daily backup fills a unique hourly slot.
			keptIndices: []int{0, 1, 2, 3, 4, 5, 6, 10, 17, 18},
		},
		{
			// Weekly backups with daily slots: each weekly backup has a unique day key,
			// so 10 weekly backups fill 7/7 daily slots + 3 weekly slots = all 10 kept.
			// Daily slots should only cover the most recent 7 days, not span months.
			name:    "DailyAbsorbsWeekly_WeeklyBackupsWithDailySlots_KeepsTooMany",
			backups: backupsEveryWeek(10),
			days:    7,
			weeks:   4,
			// Correct behavior: ~8 kept (4 weekly, some overlap with daily slots that
			// should only cover recent 7 days). Extra daily slots shouldn't retain
			// backups older than 7 days.
			// Bug: all 10 kept because each weekly backup fills a unique day slot.
			keptIndices: []int{0, 1, 2, 3},
		},
		{
			// Monthly backups with many weekly slots: each monthly backup has a unique week key,
			// so 8 monthly backups fill 8/52 weekly slots, all getting kept.
			// Weekly slots should only cover the most recent weeks, not span years.
			name:    "WeeklyAbsorbsMonthly_MonthlyBackupsWithWeeklySlots_KeepsTooMany",
			backups: backupsEveryMonth(8),
			weeks:   52,
			months:  3,
			// Correct behavior: 3 kept (3 monthly slots, weekly should only cover recent 52 weeks
			// but not artificially retain monthly backups).
			// Bug: all 8 kept because each monthly backup fills a unique week slot.
			keptIndices: []int{0, 1, 2},
		},
		{
			// Yearly backups with monthly slots: each yearly backup (on different month of year)
			// has a unique month key, so 5 yearly backups fill 5/12 monthly slots = all 5 kept.
			// Monthly slots should only cover the most recent 12 months, not span decades.
			name:    "MonthlyAbsorbsYearly_YearlyBackupsWithMonthlySlots_KeepsTooMany",
			backups: backupsEveryYear(5),
			months:  12,
			years:   3,
			// Correct behavior: 3 kept (3 yearly slots).
			// Bug: all 5 kept because each yearly backup fills a unique month slot.
			keptIndices: []int{0, 1, 2},
		},

		// Non-adjacent (skip-level) absorption:
		{
			// Weekly backups with hourly slots: each weekly backup has a unique hour key,
			// so 8 weekly backups fill 8/24 hourly slots = all 8 kept.
			name:    "HourlyAbsorbsWeekly_WeeklyBackupsWithHourlySlots_KeepsTooMany",
			backups: backupsEveryWeek(8),
			hours:   24,
			weeks:   4,
			// Correct behavior: 4 kept (4 weekly slots, hourly should only cover recent 24h).
			// Bug: all 8 kept because each weekly backup fills a unique hourly slot.
			keptIndices: []int{0, 1, 2, 3},
		},
		{
			// Monthly backups with daily slots: each monthly backup has a unique day key,
			// so 6 monthly backups fill 6/7 daily slots = all 6 kept.
			name:    "DailyAbsorbsMonthly_MonthlyBackupsWithDailySlots_KeepsTooMany",
			backups: backupsEveryMonth(6),
			days:    7,
			months:  3,
			// Correct behavior: 3 kept (3 monthly slots, daily should only cover recent 7 days).
			// Bug: all 6 kept because each monthly backup fills a unique day slot.
			keptIndices: []int{0, 1, 2},
		},

		// Full-stack (production-like config):
		{
			// Production config with all levels active. Daily backups span 30 days.
			// Hourly slots absorb everything because each daily backup has a unique hour key.
			name:    "FullStack_AllLevelsAbsorb_DailyBackupsWithFullConfig_KeepsTooMany",
			backups: backupsEveryDay(30),
			hours:   24,
			days:    7,
			weeks:   4,
			months:  1,
			years:   1,
			// Correct behavior: ~9 kept (7 daily + ~2 extra weekly, month+year overlap).
			// Bug: 24+ kept because hourly absorbs 24 unique date-hour combos.
			keptIndices: []int{0, 1, 2, 3, 4, 5, 6, 10, 17},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			retainedTiersByBackupID := buildRetainedTiersByBackupID(
				tc.backups,
				tc.hours,
				tc.days,
				tc.weeks,
				tc.months,
				tc.years,
			)

			keptIndexSet := make(map[int]bool, len(tc.keptIndices))
			for _, idx := range tc.keptIndices {
				keptIndexSet[idx] = true
			}

			for i, backup := range tc.backups {
				if keptIndexSet[i] {
					assert.NotEmpty(t, retainedTiersByBackupID[backup.ID],
						"backup at index %d (date=%s) should be kept",
						i, backup.CreatedAt.Format("2006-01-02 15:04"))
				} else {
					assert.Empty(t, retainedTiersByBackupID[backup.ID],
						"backup at index %d (date=%s) should be deleted",
						i, backup.CreatedAt.Format("2006-01-02 15:04"))
				}
			}
		})
	}
}

func Test_CleanByGFS_KeepsCorrectBackupsPerSlot(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsDays:    3,
		RetentionGfsWeeks:   0,
		RetentionGfsMonths:  0,
		RetentionGfsYears:   0,
		StorageID:           &storage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 5 backups on 5 different days; only the 3 newest days should be kept
	var backupIDs []uuid.UUID
	for i := range 5 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(4-i) * 24 * time.Hour).Truncate(24 * time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.False(t, remainingIDs[backupIDs[0]], "Oldest daily backup should be deleted")
	assert.False(t, remainingIDs[backupIDs[1]], "2nd oldest daily backup should be deleted")
	assert.True(t, remainingIDs[backupIDs[2]], "3rd backup should remain")
	assert.True(t, remainingIDs[backupIDs[3]], "4th backup should remain")
	assert.True(t, remainingIDs[backupIDs[4]], "Newest backup should remain")
}

func Test_CleanByGFS_WithWeeklyAndMonthlySlots_KeepsWiderSpread(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsDays:    2,
		RetentionGfsWeeks:   2,
		RetentionGfsMonths:  1,
		RetentionGfsYears:   0,
		StorageID:           &storage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create one backup per week for 6 weeks (each on Monday of that week)
	// GFS should keep: 2 daily (most recent 2 unique days) + 2 weekly + 1 monthly = up to 5 unique
	var createdIDs []uuid.UUID
	for i := range 6 {
		weekOffset := time.Duration(5-i) * 7 * 24 * time.Hour
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    storage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-weekOffset).Truncate(24 * time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
		createdIDs = append(createdIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	// We should have at most 5 backups kept (2 daily + 2 weekly + 1 monthly, but with overlap possible)
	// The exact count depends on how many unique periods are covered
	assert.LessOrEqual(t, len(remainingBackups), 5)
	assert.GreaterOrEqual(t, len(remainingBackups), 1)

	// The two most recent backups should always be retained (daily slots)
	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.True(t, remainingIDs[createdIDs[4]], "Second newest backup should be retained (daily)")
	assert.True(t, remainingIDs[createdIDs[5]], "Newest backup should be retained (daily)")
}

func Test_CleanByGFS_WithHourlySlots_KeepsCorrectBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsHours:   3,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 5 backups spaced 1 hour apart; only the 3 newest hours should be kept
	var backupIDs []uuid.UUID
	for i := range 5 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(4-i) * time.Hour).Truncate(time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(remainingBackups))

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}
	assert.False(t, remainingIDs[backupIDs[0]], "Oldest hourly backup should be deleted")
	assert.False(t, remainingIDs[backupIDs[1]], "2nd oldest hourly backup should be deleted")
	assert.True(t, remainingIDs[backupIDs[2]], "3rd backup should remain")
	assert.True(t, remainingIDs[backupIDs[3]], "4th backup should remain")
	assert.True(t, remainingIDs[backupIDs[4]], "Newest backup should remain")
}

func Test_CleanByGFS_SkipsRecentBackup_WhenNotInKeepSet(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	storage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, storage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), storage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	// Keep only 1 daily slot. We create 2 old backups plus two recent backups on today.
	// Backups are ordered newest-first, so the 15-min-old backup fills the single daily slot.
	// The 30-min-old backup is the same day → not in the GFS keep-set, but it is still recent
	// (within grace period) and must be preserved.
	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsDays:    1,
		StorageID:           &storage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	oldBackup1 := &backups_core_logical.LogicalBackup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-3 * 24 * time.Hour).Truncate(24 * time.Hour),
	}
	oldBackup2 := &backups_core_logical.LogicalBackup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-2 * 24 * time.Hour).Truncate(24 * time.Hour),
	}
	// Newest backup today — will fill the single GFS daily slot.
	newestTodayBackup := &backups_core_logical.LogicalBackup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-15 * time.Minute),
	}
	// Slightly older backup, also today — NOT in GFS keep-set (duplicate day),
	// but within the 60-minute grace period so it must survive.
	recentNotInKeepSet := &backups_core_logical.LogicalBackup{
		ID:           uuid.New(),
		DatabaseID:   database.ID,
		StorageID:    storage.ID,
		Status:       backups_core_logical.BackupStatusCompleted,
		BackupSizeMb: 10,
		CreatedAt:    now.Add(-30 * time.Minute),
	}

	for _, b := range []*backups_core_logical.LogicalBackup{oldBackup1, oldBackup2, newestTodayBackup, recentNotInKeepSet} {
		err = getBackupRepository().Save(b)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}

	assert.False(t, remainingIDs[oldBackup1.ID], "Old backup 1 should be deleted by GFS")
	assert.False(t, remainingIDs[oldBackup2.ID], "Old backup 2 should be deleted by GFS")
	assert.True(
		t,
		remainingIDs[newestTodayBackup.ID],
		"Newest backup fills GFS daily slot and must remain",
	)
	assert.True(
		t,
		remainingIDs[recentNotInKeepSet.ID],
		"Recent backup not in keep-set must be preserved by grace period",
	)
}

func Test_CleanByGFS_With20DailyBackups_KeepsOnlyExpectedCount(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsDays:    7,
		RetentionGfsWeeks:   4,
		RetentionGfsMonths:  1,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 20 daily backups, all older than the 60-minute grace period.
	// backupIDs[0] is oldest (19 days ago), backupIDs[19] is newest (~2h ago).
	var backupIDs []uuid.UUID
	for i := range 20 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 10,
			CreatedAt:    now.Add(-time.Duration(19-i)*24*time.Hour - 2*time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
		backupIDs = append(backupIDs, backup.ID)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	// GFS 7d/4w/1m: 7 daily slots + up to 4 weekly + 1 monthly (with overlap).
	// Theoretical max unique kept: 12. Realistic: ~9 depending on ISO week layout.
	assert.LessOrEqual(t, len(remainingBackups), 12,
		"At most 12 backups should be retained (7d+4w+1m)")
	assert.GreaterOrEqual(t, len(remainingBackups), 7,
		"At least 7 backups should be retained for daily slots")

	remainingIDs := make(map[uuid.UUID]bool)
	for _, backup := range remainingBackups {
		remainingIDs[backup.ID] = true
	}

	// The newest 7 backups (indices 13-19) must always be retained (daily slots)
	for i := 13; i < 20; i++ {
		assert.True(t, remainingIDs[backupIDs[i]],
			"Backup at position %d (one of 7 newest) should be retained", i)
	}
}

func Test_CleanByGFS_WithMultipleBackupsPerDay_KeepsOnlyOnePerDailySlot(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsDays:    7,
		RetentionGfsWeeks:   4,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 3 backups per day for 10 days = 30 total, all beyond grace period.
	// Each day gets backups at base+0h, base+6h, base+12h.
	for day := range 10 {
		for sub := range 3 {
			backup := &backups_core_logical.LogicalBackup{
				ID:           uuid.New(),
				DatabaseID:   database.ID,
				StorageID:    testStorage.ID,
				Status:       backups_core_logical.BackupStatusCompleted,
				BackupSizeMb: 10,
				CreatedAt: now.Add(
					-time.Duration(9-day)*24*time.Hour -
						2*time.Hour -
						time.Duration(2-sub)*6*time.Hour,
				),
			}
			err = getBackupRepository().Save(backup)
			assert.NoError(t, err)
		}
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	// 10 days spans ~2 ISO weeks. GFS 7d/4w: weekly slots overlap with daily.
	// Expected: ~7-9 backups (7 daily + possibly extra weekly for uncovered weeks).
	assert.LessOrEqual(t, len(remainingBackups), 11,
		"Should keep at most 11 backups (7d+4w theoretical max)")
	assert.GreaterOrEqual(t, len(remainingBackups), 7,
		"Should keep at least 7 backups for daily slots")

	// Only 1 backup per calendar day should be retained
	dayCount := make(map[string]int)
	for _, backup := range remainingBackups {
		dayKey := backup.CreatedAt.Format("2006-01-02")
		dayCount[dayKey]++
	}

	for dayKey, count := range dayCount {
		assert.Equal(t, 1, count,
			"Day %s should have exactly 1 retained backup, got %d", dayKey, count)
	}
}

// When backups are created once per day but hourly slots are configured (e.g. 24h),
// each daily backup lands in a unique hour key (different date-hour combo), filling
// hourly slots with old backups that shouldn't be retained by hourly rotation.
// Hourly slots should only cover the most recent hours, not span across weeks.
func Test_CleanByGFS_With24HourlySlotsAnd23DailyBackups_DeletesExcessBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsHours:   24,
		RetentionGfsDays:    7,
		RetentionGfsWeeks:   4,
		RetentionGfsMonths:  12,
		RetentionGfsYears:   3,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	for i := range 23 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 0.02,
			CreatedAt:    now.Add(-time.Duration(22-i)*24*time.Hour - 2*time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	assert.Less(t, len(remainingBackups), 20,
		"GFS with 24h/7d/4w/12m/3y should not retain all 23 daily backups")
	assert.GreaterOrEqual(t, len(remainingBackups), 7,
		"At least 7 backups should be retained for daily slots")
}

// Same scenario as above but with hourly slots disabled (0h). This verifies
// that the daily/weekly/monthly/yearly rotation correctly prunes excess backups
// when hourly slots are not involved.
func Test_CleanByGFS_WithDisabledHourlySlotsAnd23DailyBackups_DeletesExcessBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsHours:   0,
		RetentionGfsDays:    7,
		RetentionGfsWeeks:   4,
		RetentionGfsMonths:  12,
		RetentionGfsYears:   3,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	for i := range 23 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 0.02,
			CreatedAt:    now.Add(-time.Duration(22-i)*24*time.Hour - 2*time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	assert.Less(t, len(remainingBackups), 15,
		"GFS with 0h/7d/4w/12m/3y should not retain all 23 daily backups")
	assert.GreaterOrEqual(t, len(remainingBackups), 7,
		"At least 7 backups should be retained for daily slots")
}

// When weekly backups exist but daily slots are configured, each weekly backup
// has a unique day key, filling daily slots with backups that are weeks apart.
// Daily slots should only cover the most recent days, not span months.
func Test_CleanByGFS_WithDailySlotsAndWeeklyBackups_DeletesExcessBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsDays:    7,
		RetentionGfsWeeks:   4,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 10 weekly backups (1 per week, all >2h old past grace period).
	// With 7d/4w config, correct behavior: ~8 kept (4 weekly + overlap with daily for recent ones).
	// Daily slots should NOT absorb weekly backups that are older than 7 days.
	for i := range 10 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 0.02,
			CreatedAt:    now.Add(-time.Duration(9-i)*7*24*time.Hour - 2*time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	fmt.Printf(
		"[TEST] WithDailySlotsAndWeeklyBackups: %d remaining out of 10\n",
		len(remainingBackups),
	)

	// With weekly backups and 7d/4w config, daily slots should only cover the most recent
	// 7 days. Since backups are 1 week apart, at most 1 backup falls within the daily window.
	// Correct: ~4-5 kept (4 weekly + possibly 1 overlapping daily). Not 7+.
	assert.LessOrEqual(
		t,
		len(remainingBackups),
		5,
		"GFS with 7d/4w should keep at most ~5 weekly backups — daily slots should not absorb weekly backups older than 7 days",
	)
	assert.GreaterOrEqual(t, len(remainingBackups), 4,
		"At least 4 backups should be retained for weekly slots")
}

// When monthly backups exist but weekly slots are configured, each monthly backup
// has a unique week key, filling weekly slots with backups that are months apart.
// Weekly slots should only cover the most recent weeks, not span years.
func Test_CleanByGFS_WithWeeklySlotsAndMonthlyBackups_DeletesExcessBackups(t *testing.T) {
	router := CreateTestRouter()
	owner := users_testing.CreateTestUser(t.Context(), users.RoleMember)
	workspace := workspaces_testing.CreateTestWorkspace(t.Context(), "Test Workspace", owner, router)
	testStorage := storages.CreateTestStorage(workspace.ID)
	notifier := notifiers.CreateTestNotifier(workspace.ID)
	database := databases.CreateTestDatabase(workspace.ID, testStorage, notifier)

	defer func() {
		backups, _ := getBackupRepository().FindByDatabaseID(database.ID)
		for _, backup := range backups {
			getBackupRepository().DeleteByID(backup.ID)
		}

		databases.RemoveTestDatabase(t.Context(), database)
		time.Sleep(50 * time.Millisecond)
		notifiers.RemoveTestNotifier(notifier)
		storages.RemoveTestStorage(t.Context(), testStorage.ID)
		workspaces_testing.RemoveTestWorkspace(t.Context(), workspace, router)
	}()

	interval := createTestInterval()

	backupConfig := &backups_config_logical.LogicalBackupConfig{
		DatabaseID:          database.ID,
		IsBackupsEnabled:    true,
		RetentionPolicyType: backups_config_logical.RetentionPolicyTypeGFS,
		RetentionGfsWeeks:   52,
		RetentionGfsMonths:  3,
		StorageID:           &testStorage.ID,
		BackupInterval:      interval,
	}
	_, err := backups_config_logical.GetBackupConfigService().SaveBackupConfig(t.Context(), backupConfig)
	assert.NoError(t, err)

	now := time.Now().UTC()

	// Create 8 monthly backups (1 per month, all >2h old past grace period).
	// With 52w/3m config, correct behavior: 3 kept (3 monthly slots; weekly should only
	// cover recent 52 weeks but not artificially retain old monthly backups).
	// Bug: all 8 kept because each monthly backup fills a unique weekly slot.
	for i := range 8 {
		backup := &backups_core_logical.LogicalBackup{
			ID:           uuid.New(),
			DatabaseID:   database.ID,
			StorageID:    testStorage.ID,
			Status:       backups_core_logical.BackupStatusCompleted,
			BackupSizeMb: 0.02,
			CreatedAt:    now.AddDate(0, -(7 - i), 0).Add(-2 * time.Hour),
		}
		err = getBackupRepository().Save(backup)
		assert.NoError(t, err)
	}

	cleaner := GetBackupCleaner()
	err = cleaner.cleanByRetentionPolicy(t.Context(), testLogger())
	assert.NoError(t, err)

	remainingBackups, err := getBackupRepository().FindByDatabaseID(database.ID)
	assert.NoError(t, err)

	fmt.Printf(
		"[TEST] WithWeeklySlotsAndMonthlyBackups: %d remaining out of 8\n",
		len(remainingBackups),
	)

	// Weekly slots should not absorb monthly backups. Correct: ~3 kept (3 monthly).
	assert.LessOrEqual(
		t,
		len(remainingBackups),
		5,
		"GFS with 52w/3m should keep at most ~5 backups — weekly slots should not absorb monthly backups",
	)
	assert.GreaterOrEqual(t, len(remainingBackups), 3,
		"At least 3 backups should be retained for monthly slots")
}
