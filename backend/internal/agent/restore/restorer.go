package restore

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"time"
)

type Restorer struct {
	log *slog.Logger
}

func NewRestorer(log *slog.Logger) *Restorer {
	return &Restorer{log: log}
}

// No host bind mount: the decrypted dump never touches the agent host.
func (r *Restorer) StageBackupViaExec(
	ctx context.Context,
	exec ExecRunner,
	body io.Reader,
	destPath string,
) error {
	result, err := exec.Exec(ctx, []string{"dd", "of=" + destPath, "bs=4M"}, body, nil)
	if err != nil {
		return fmt.Errorf("stage exec: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf(
			"stage dd exited %d: %s", result.ExitCode, tailString(result.Stderr, 2048))
	}

	return nil
}

func (r *Restorer) RunPgRestore(
	ctx context.Context,
	exec ExecRunner,
	spec PgRestoreSpec,
) (Result, error) {
	started := time.Now().UTC()

	conn := spec.Conn

	cmd := []string{
		"pg_restore", "-Fc", "--no-password",
		"--no-owner", "--no-acl",
		"-h", conn.Host, "-p", strconv.Itoa(conn.Port),
		"-U", conn.User, "-d", conn.Database,
		"-j", strconv.Itoa(spec.ParallelJobs),
	}

	// The target is a throwaway freshly-initdb'd database, so every DROP is a no-op except the one
	// that matters: an archive dumped with -n public carries CREATE SCHEMA public, which would
	// otherwise collide with the schema initdb already made (issue #726). TimescaleDB is excluded
	// because --clean would DROP EXTENSION timescaledb, taking the catalog pre_restore needs.
	if !spec.IsTimescaledb {
		cmd = append(cmd, "--clean", "--if-exists")
	}

	cmd = append(cmd, spec.ArchivePath)

	env := []string{"PGPASSWORD=" + conn.Password}

	execResult, err := exec.Exec(ctx, cmd, nil, env)

	result := Result{
		PgRestoreExitCode: execResult.ExitCode,
		DurationMs:        time.Since(started).Milliseconds(),
		StderrTail:        tailString(execResult.Stderr, 8192),
	}

	if err != nil {
		return result, fmt.Errorf("pg_restore exec: %w", err)
	}

	if execResult.ExitCode != 0 {
		return result, fmt.Errorf("%w (code %d)", ErrRestoreFailed, execResult.ExitCode)
	}

	return result, nil
}

func tailString(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}

	return s[len(s)-maxBytes:]
}
