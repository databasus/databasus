package internaldb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/process"
)

// internalServerVersion is the major version the image pins for the internal database.
const internalServerVersion = pgshared.Version17

const (
	toolKillAfter       = 10 * time.Second
	maxDiagnosticLength = 2000
	dumpCompression     = "--compress=zstd:5"
)

type DumpSpec struct {
	ClientTools pgshared.ClientTools
	Connection  Connection
	// SnapshotID is a snapshot exported by a transaction that stays open while the dump runs.
	SnapshotID string
	Logger     *slog.Logger
}

// Dump writes the whole internal database as a pg_dump custom-format archive, including goose's
// migration history, without owners or privileges, until pg_dump exits.
func Dump(ctx context.Context, spec DumpSpec, output io.Writer) error {
	if spec.SnapshotID == "" {
		return ErrSnapshotIDRequired
	}

	pgDump := spec.ClientTools.GetExecutable(internalServerVersion, pgshared.ExecutablePgDump)
	args := []string{
		"-Fc",
		"--no-password",
		"--no-owner",
		"--no-privileges",
		"--snapshot=" + spec.SnapshotID,
		dumpCompression,
	}

	spec.Logger.InfoContext(ctx, "dumping the internal database", slog.String("pg_bin", pgDump))

	var stderr bytes.Buffer

	err := process.Run(ctx, process.Spec{
		Path:       pgDump,
		Args:       args,
		UnsetEnv:   nil,
		Env:        spec.Connection.toEnv(),
		Dir:        "",
		Stdin:      nil,
		Stdout:     output,
		Stderr:     &stderr,
		StopSignal: nil,
		KillAfter:  toolKillAfter,
	})
	if err != nil {
		return fmt.Errorf("%w: %w: %s", ErrDumpFailed, err, truncate(stderr.String()))
	}

	return nil
}

func truncate(diagnostic string) string {
	if len(diagnostic) <= maxDiagnosticLength {
		return diagnostic
	}

	return diagnostic[:maxDiagnosticLength]
}
