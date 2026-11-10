package restore

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"databasus-backend/internal/engines/postgresql/logical/pgrestore"
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
			"stage dd exited %d: %s", result.ExitCode, tailString(result.Stderr, 2048),
		)
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

	args := pgrestore.BuildArgs(pgrestore.ArgsSpec{
		Host:                conn.Host,
		Port:                conn.Port,
		User:                conn.User,
		Database:            conn.Database,
		ParallelJobs:        spec.ParallelJobs,
		IsTimescaleDB:       spec.IsTimescaledb,
		IsRestoreOwnership:  false,
		IsRestorePrivileges: false,
		IsVerbose:           false,
		TocListPath:         "",
		ArchivePath:         spec.ArchivePath,
	})
	cmd := append([]string{"pg_restore"}, args...)

	env := []string{"PGPASSWORD=" + conn.Password}

	execResult, err := exec.Exec(ctx, cmd, nil, env)

	result := Result{
		FailureDiagnostic: pgrestore.GetFailureDiagnostic(
			pgrestore.FailureDiagnosticSpec{
				Output:      execResult.Stderr,
				Secrets:     append(append([]string(nil), spec.DiagnosticSecrets...), conn.Password),
				IsTruncated: false,
			},
		),
		PgRestoreExitCode: execResult.ExitCode,
		DurationMs:        time.Since(started).Milliseconds(),
		StderrTail:        tailString(execResult.Stderr, 8192),
		// The verification container may lack source extensions; dependent data errors still fail.
		HasOnlyIgnorableErrors: pgrestore.HasOnlyIgnorableErrors(execResult.Stderr,
			pgrestore.PreexistingPublicSchema, pgrestore.UnavailableExtension),
		UnavailableExtensions: pgrestore.GetUnavailableExtensions(execResult.Stderr),
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
