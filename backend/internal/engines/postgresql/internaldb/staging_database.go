package internaldb

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	// StagingPrefix starts the name of every database a recovery stages into, so cleanup and the
	// publication swap never touch another database.
	StagingPrefix = "databasus_staging_"
	// PreviousPrefix starts the name the replaced application database keeps until its recovery
	// completes.
	PreviousPrefix = "databasus_previous_"
)

// CreateStagingDatabase creates an empty database on the server of maintenance. A stale database
// of the same name is dropped first.
func CreateStagingDatabase(ctx context.Context, maintenance Connection, name string) error {
	if err := validateStagingName(name); err != nil {
		return err
	}

	return runMaintenance(ctx, maintenance, []string{
		"DROP DATABASE IF EXISTS " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)",
		"CREATE DATABASE " + pgx.Identifier{name}.Sanitize() + " TEMPLATE template0 ENCODING 'UTF8'",
	})
}

// DropStagingDatabase removes a staging database, ending its sessions.
func DropStagingDatabase(ctx context.Context, maintenance Connection, name string) error {
	if err := validateStagingName(name); err != nil {
		return err
	}

	return runMaintenance(ctx, maintenance, []string{
		"DROP DATABASE IF EXISTS " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)",
	})
}

// DropPreviousDatabase removes the application database a completed recovery replaced.
func DropPreviousDatabase(ctx context.Context, maintenance Connection, name string) error {
	if !strings.HasPrefix(name, PreviousPrefix) || len(name) == len(PreviousPrefix) {
		return ErrPreviousNameInvalid
	}

	return runMaintenance(ctx, maintenance, []string{
		"DROP DATABASE IF EXISTS " + pgx.Identifier{name}.Sanitize() + " WITH (FORCE)",
	})
}

// SwapSpec names the databases one publication renames: Active becomes Previous and Staging
// becomes Active.
type SwapSpec struct {
	Active   string
	Staging  string
	Previous string
}

// SwapDatabases publishes a staging database under the active name in one transaction, so the
// server never lacks the active database and a crash leaves either the old or the new one under
// that name. The maintenance connection must reach another database, because a database with
// sessions cannot be renamed.
func SwapDatabases(ctx context.Context, maintenance Connection, spec SwapSpec) error {
	if err := validateStagingName(spec.Staging); err != nil {
		return err
	}

	if !strings.HasPrefix(spec.Previous, PreviousPrefix) || maintenance.Database == spec.Active {
		return ErrPreviousNameInvalid
	}

	connection, err := pgx.Connect(ctx, maintenance.ConnString())
	if err != nil {
		return fmt.Errorf("connect to the internal database server: %w", err)
	}
	defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

	err = pgx.BeginFunc(ctx, connection, func(transaction pgx.Tx) error {
		for _, statement := range []string{
			"ALTER DATABASE " + pgx.Identifier{spec.Active}.Sanitize() + " RENAME TO " +
				pgx.Identifier{spec.Previous}.Sanitize(),
			"ALTER DATABASE " + pgx.Identifier{spec.Staging}.Sanitize() + " RENAME TO " +
				pgx.Identifier{spec.Active}.Sanitize(),
		} {
			if _, err := transaction.Exec(ctx, statement); err != nil {
				return fmt.Errorf("run %q: %w", statement, err)
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("swap the staging database in: %w", err)
	}

	return nil
}

// ListDatabases returns the databases on the server whose names start with prefix, so a restart
// can release the staging databases no recovery owns any longer.
func ListDatabases(ctx context.Context, maintenance Connection, prefix string) ([]string, error) {
	connection, err := pgx.Connect(ctx, maintenance.ConnString())
	if err != nil {
		return nil, fmt.Errorf("connect to the internal database server: %w", err)
	}
	defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

	rows, err := connection.Query(ctx, "SELECT datname FROM pg_database WHERE starts_with(datname, $1)", prefix)
	if err != nil {
		return nil, fmt.Errorf("list the databases: %w", err)
	}

	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("list the databases: %w", err)
	}

	return names, nil
}

func runMaintenance(ctx context.Context, maintenance Connection, statements []string) error {
	connection, err := pgx.Connect(ctx, maintenance.ConnString())
	if err != nil {
		return fmt.Errorf("connect to the internal database server: %w", err)
	}
	defer func() { _ = connection.Close(context.WithoutCancel(ctx)) }()

	for _, statement := range statements {
		if _, err := connection.Exec(ctx, statement); err != nil {
			return fmt.Errorf("run %q: %w", statement, err)
		}
	}

	return nil
}

func validateStagingName(name string) error {
	if !strings.HasPrefix(name, StagingPrefix) || len(name) == len(StagingPrefix) {
		return ErrStagingNameInvalid
	}

	return nil
}
