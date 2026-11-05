package restore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"databasus-backend/internal/agent/dbconn"
	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
)

func (r *Restorer) RunTimescalePreRestore(ctx context.Context, conn dbconn.Conn) error {
	return execStatements(ctx, conn, pgrestore.GetTimescaleDBPreRestoreStatements()...)
}

func (r *Restorer) RunTimescalePostRestore(ctx context.Context, conn dbconn.Conn) error {
	return execStatements(ctx, conn, pgrestore.GetTimescaleDBPostRestoreStatements()...)
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
