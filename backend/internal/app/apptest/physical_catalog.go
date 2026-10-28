package apptest

import (
	"testing"

	"databasus-backend/internal/lib/database"
)

func (a *App) insertPhysicalCatalogRow(tb testing.TB, table string, row any) {
	tb.Helper()
	db, err := database.Open(tb.Context(), a.databaseDsn)
	if err != nil {
		tb.Fatalf("open the physical catalog database: %v", err)
	}
	defer closeConnections(db)
	if err := db.WithContext(tb.Context()).Table(table).Create(row).Error; err != nil {
		tb.Fatalf("insert a row in %s: %v", table, err)
	}
}
