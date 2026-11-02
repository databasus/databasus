package logical

import (
	"context"
	"io"
	"strconv"
	"time"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/process"
)

const (
	// clientToolKillAfter bounds how long a run waits for more of the tool's output after the tool
	// exited: a process the tool left behind can hold that output open.
	clientToolKillAfter         = 10 * time.Second
	clientConnectTimeoutSeconds = 30
)

// clientToolRun is one start of pg_dump or pg_restore.
type clientToolRun struct {
	path        string
	args        []string
	credentials *pgshared.CredentialTempFiles
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
}

// runClientTool starts the tool in a process group of its own, so an ended ctx stops pg_restore -j
// together with its workers.
func (s *Source) runClientTool(ctx context.Context, run clientToolRun) error {
	return process.Run(ctx, process.Spec{
		Path:       run.path,
		Args:       run.args,
		Env:        s.buildClientToolEnv(run.credentials),
		Dir:        "",
		Stdin:      run.stdin,
		Stdout:     run.stdout,
		Stderr:     run.stderr,
		StopSignal: nil,
		KillAfter:  clientToolKillAfter,
	})
}

// buildClientToolEnv passes the password through the .pgpass file and the certificates through the
// libpq variables, so neither appears in the tool's arguments.
func (s *Source) buildClientToolEnv(credentials *pgshared.CredentialTempFiles) []string {
	env := []string{"PGPASSFILE=" + credentials.PgpassPath}
	env = append(env, pgshared.GetPgHostAddrEnv(s.settings.HostAddr)...)
	env = append(env,
		"PGCLIENTENCODING=UTF8",
		"PGCONNECT_TIMEOUT="+strconv.Itoa(clientConnectTimeoutSeconds),
		"LC_ALL=C.UTF-8",
		"LANG=C.UTF-8",
	)

	sslMode := s.settings.TLS.Mode
	if sslMode == "" {
		sslMode = pgshared.SslModeDisable
	}

	return append(env,
		"PGSSLMODE="+string(sslMode),
		"PGSSLCERT="+credentials.ClientCertPath,
		"PGSSLKEY="+credentials.ClientKeyPath,
		"PGSSLROOTCERT="+credentials.RootCertPath,
		"PGSSLCRL=",
	)
}

func (s *Source) writeCredentialFiles() (*pgshared.CredentialTempFiles, error) {
	return pgshared.WriteCredentialFilesToTempDir(s.settings.GetCredentialSpec(), s.settings.Password)
}
