package testdb

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"sync"
	"time"

	"gorm.io/gorm"
)

const abandonedDatabaseAge = time.Hour

var (
	testDatabaseNamePattern = regexp.MustCompile(`^databasus_test_([0-9]+)_[0-9a-f]{8}$`)

	abandonedDatabasesCleanup sync.Once

	// The tests of the cleanup itself move its clock forward and narrow it to databases they
	// created, so the cleanup they trigger never drops a database of another run.
	processCleanupScopeForTest *cleanupScope
)

type cleanupScope struct {
	now time.Time
	// An empty list puts every test database in scope.
	databaseNames []string
}

// A process killed before its cleanups ran, or a process whose database lives as long as the
// process, leaves its databases behind. Every process removes them once, on its first request.
func removeAbandonedDatabasesOncePerProcess(ctx context.Context, systemDb *gorm.DB) error {
	var err error

	abandonedDatabasesCleanup.Do(func() {
		scope := cleanupScope{now: time.Now(), databaseNames: nil}
		if processCleanupScopeForTest != nil {
			scope = *processCleanupScopeForTest
		}

		err = removeAbandonedDatabases(ctx, systemDb, scope)
	})

	return err
}

// removeAbandonedDatabases drops the test databases in scope that were created over an hour before
// the scope's time and have no connections. It takes no lock: a neighbouring process may drop the
// same database first, and a database someone connects to in the meantime refuses the drop, so
// every drop error is ignored.
func removeAbandonedDatabases(ctx context.Context, systemDb *gorm.DB, scope cleanupScope) error {
	var unconnectedNames []string

	err := systemDb.WithContext(ctx).Raw(`
		SELECT datname FROM pg_database
		WHERE datname LIKE 'databasus\_test\_%'
		  AND NOT EXISTS (SELECT 1 FROM pg_stat_activity WHERE pg_stat_activity.datname = pg_database.datname)
	`).Scan(&unconnectedNames).Error
	if err != nil {
		return fmt.Errorf("list abandoned test databases: %w", err)
	}

	for _, databaseName := range unconnectedNames {
		if len(scope.databaseNames) > 0 && !slices.Contains(scope.databaseNames, databaseName) {
			continue
		}

		createdAt, isTestDatabase := getTestDatabaseCreationTime(databaseName)
		if !isTestDatabase || scope.now.Sub(createdAt) < abandonedDatabaseAge {
			continue
		}

		_ = systemDb.WithContext(ctx).Exec("DROP DATABASE IF EXISTS " + databaseName).Error
	}

	return nil
}

func getTestDatabaseCreationTime(databaseName string) (time.Time, bool) {
	nameMatch := testDatabaseNamePattern.FindStringSubmatch(databaseName)
	if nameMatch == nil {
		return time.Time{}, false
	}

	createdAtSeconds, err := strconv.ParseInt(nameMatch[1], 10, 64)
	if err != nil {
		return time.Time{}, false
	}

	return time.Unix(createdAtSeconds, 0), true
}
