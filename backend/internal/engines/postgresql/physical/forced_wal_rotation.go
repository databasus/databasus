package physical

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// pgRowQuerier is satisfied by both *pgx.Conn and pgx.Tx, so a probe can run on a
// plain connection or inside the provisioning transaction that just created the role.
type pgRowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Reading the privilege instead of calling pg_switch_wal() keeps this safe to run on
// every connection test: a successful call would archive a padded WAL segment.
func canForceWalRotation(ctx context.Context, querier pgRowQuerier, roleName string) (bool, error) {
	var canExecute bool

	if err := querier.QueryRow(
		ctx,
		`SELECT has_function_privilege($1, 'pg_switch_wal()', 'EXECUTE')`,
		roleName,
	).Scan(&canExecute); err != nil {
		return false, fmt.Errorf("failed to check pg_switch_wal privilege: %w", err)
	}

	return canExecute, nil
}

// A source that will not confer this cannot be used for continuous WAL streaming, but
// it still serves FULL and incremental backups, so a refusal is an outcome rather than a
// failure. The GRANT runs in a savepoint because a refused statement would otherwise
// abort the transaction that created the role.
func grantWalSwitchIfPermitted(ctx context.Context, tx pgx.Tx, username string) (isGranted bool, err error) {
	savepoint, err := tx.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to open savepoint for the pg_switch_wal grant: %w", err)
	}

	quotedUsername := pgx.Identifier{username}.Sanitize()

	_, err = savepoint.Exec(ctx,
		fmt.Sprintf(`GRANT EXECUTE ON FUNCTION pg_switch_wal() TO %s`, quotedUsername))
	if err != nil {
		if rollbackErr := savepoint.Rollback(ctx); rollbackErr != nil {
			return false, fmt.Errorf("failed to roll back the refused pg_switch_wal grant: %w", rollbackErr)
		}

		if classifyServerError(err) == serverErrorPermissionDenied {
			return false, nil
		}

		return false, fmt.Errorf("failed to grant EXECUTE on pg_switch_wal: %w", err)
	}

	if err := savepoint.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit the pg_switch_wal grant: %w", err)
	}

	// The catalog, not the statement's exit status, is what the connection test later
	// reads, so provisioning and testing cannot disagree about the same role.
	return canForceWalRotation(ctx, tx, username)
}
