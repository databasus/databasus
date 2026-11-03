package app

import (
	"errors"
	"fmt"
	"sync/atomic"

	"gorm.io/gorm"
)

const internalDatabaseQueryCounterCallback = "app:count_internal_database_queries"

type databaseStatementProcessor interface {
	Register(name string, fn func(*gorm.DB)) error
	Remove(name string) error
}

// CountInternalDatabaseQueries counts the statements this application sends while run works. Registering a
// GORM callback is not safe during other queries, so a caller runs it with no background work going.
func (a *App) CountInternalDatabaseQueries(run func()) (int64, error) {
	callbacks := a.db.Callback()
	processors := []databaseStatementProcessor{
		callbacks.Query(), callbacks.Row(), callbacks.Raw(),
		callbacks.Create(), callbacks.Update(), callbacks.Delete(),
	}

	var queryCount atomic.Int64

	for _, processor := range processors {
		err := processor.Register(internalDatabaseQueryCounterCallback, func(*gorm.DB) { queryCount.Add(1) })
		if err != nil {
			return 0, fmt.Errorf("register the internal database query counter: %w", err)
		}
	}

	run()

	var removalErrors []error
	for _, processor := range processors {
		removalErrors = append(removalErrors, processor.Remove(internalDatabaseQueryCounterCallback))
	}

	if err := errors.Join(removalErrors...); err != nil {
		return 0, fmt.Errorf("remove the internal database query counter: %w", err)
	}

	return queryCount.Load(), nil
}
