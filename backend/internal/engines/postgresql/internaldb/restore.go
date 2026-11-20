package internaldb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/process"
)

type RestoreSpec struct {
	ClientTools pgshared.ClientTools
	// Connection names the empty staging database the archive restores into.
	Connection Connection
	Logger     *slog.Logger
}

// Restore reads a custom-format archive from input into the staging database in one transaction,
// so any restore error leaves the staging database empty.
func Restore(ctx context.Context, spec RestoreSpec, input io.Reader) error {
	if err := validateStagingName(spec.Connection.Database); err != nil {
		return err
	}

	pgRestore := spec.ClientTools.GetExecutable(internalServerVersion, pgshared.ExecutablePgRestore)
	args := []string{
		"--no-password",
		"--no-owner",
		"--no-privileges",
		"--single-transaction",
		"--exit-on-error",
		"-d", spec.Connection.Database,
	}

	spec.Logger.InfoContext(ctx, "restoring a Databasus backup into the staging database",
		slog.String("pg_bin", pgRestore), slog.String("staging_database", spec.Connection.Database))

	var stderr bytes.Buffer

	err := process.Run(ctx, process.Spec{
		Path:       pgRestore,
		Args:       args,
		UnsetEnv:   nil,
		Env:        spec.Connection.toEnv(),
		Dir:        "",
		Stdin:      input,
		Stdout:     io.Discard,
		Stderr:     &stderr,
		StopSignal: nil,
		KillAfter:  toolKillAfter,
	})
	if err != nil {
		return fmt.Errorf("%w: %w: %s", ErrRestoreFailed, err, truncate(stderr.String()))
	}

	return nil
}
