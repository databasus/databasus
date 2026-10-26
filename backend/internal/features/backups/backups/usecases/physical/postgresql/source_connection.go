package usecases_physical_postgresql

import (
	"context"

	"github.com/jackc/pgx/v5"

	pgphysical "databasus-backend/internal/engines/postgresql/physical"
	"databasus-backend/internal/modules/databases"
)

// openInspectionConn opens an ordinary connection to the source with the settings the caller resolved,
// the local end of its tunnel included.
func openInspectionConn(
	ctx context.Context,
	sourceDB databases.PostgresqlPhysicalConnectionSettings,
) (*pgx.Conn, error) {
	return pgphysical.NewSource(sourceDB.Engine).OpenInspectionConn(ctx)
}
