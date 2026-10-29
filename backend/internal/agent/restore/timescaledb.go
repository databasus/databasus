package restore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/agent/dbconn"
)

// TimescaleDB restore procedure run against the throwaway restore target.
//
// timescaledb_pre_restore() sets a database-level GUC (ALTER DATABASE ... SET
// timescaledb.restoring='on') that the separate in-container pg_restore connection inherits, and
// stops background workers. timescaledb_post_restore() resets it and restarts the workers, returning
// the database to normal mode — it must run before the verifier collects stats. The connection used
// here is the agent-host verifier conn (published port); pg_restore uses the in-container conn, but
// both reach the same database, so the GUC carries across.

// RunTimescalePreRestore installs the extension (a freshly initialised target may lack it) and
// enters restoring mode.
func (r *Restorer) RunTimescalePreRestore(ctx context.Context, conn dbconn.Conn) error {
	return execStatements(ctx, conn,
		"CREATE EXTENSION IF NOT EXISTS timescaledb",
		"SELECT timescaledb_pre_restore()",
	)
}

// RunTimescalePostRestore leaves restoring mode and restarts background workers.
func (r *Restorer) RunTimescalePostRestore(ctx context.Context, conn dbconn.Conn) error {
	return execStatements(ctx, conn, "SELECT timescaledb_post_restore()")
}

func execStatements(ctx context.Context, conn dbconn.Conn, statements ...string) error {
	pgConn, err := pgx.Connect(ctx, conn.GetDSN())
	if err != nil {
		return fmt.Errorf("connect restore target: %w", err)
	}
	defer func() { _ = pgConn.Close(ctx) }()

	for _, statement := range statements {
		if _, err := pgConn.Exec(ctx, statement); err != nil {
			return fmt.Errorf("run %q: %w", statement, err)
		}
	}

	return nil
}
