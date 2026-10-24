// Package storage hands the database pool internal/app opened to the code that has not moved into
// internal/modules yet. A test binary built with the testhelpers tag opens it on first use instead.
package storage

import (
	"sync"
	"sync/atomic"

	"gorm.io/gorm"
)

var installedDb atomic.Pointer[gorm.DB]

var getDbWithoutApp = sync.OnceValue(loadWithoutApp)

func Install(db *gorm.DB) {
	installedDb.Store(db)
}

func GetDb() *gorm.DB {
	if db := installedDb.Load(); db != nil {
		return db
	}

	return getDbWithoutApp()
}
