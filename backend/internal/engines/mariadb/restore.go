package mariadb

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"databasus-backend/internal/engines/mysqlfamily"
)

type RestoreSpec struct {
	Version            Version
	SourceDatabaseName string
	// IsSkipGaleraDisable keeps Galera replication on for the restore session, for managed
	// clusters that deny the SUPER privilege that turning it off needs.
	IsSkipGaleraDisable bool
	Logger              *slog.Logger
}

// Restore feeds the SQL read from input into the mariadb client.
func (s *Source) Restore(ctx context.Context, spec RestoreSpec, input io.Reader) error {
	if s.settings.Database == "" {
		return ErrRestoreDatabaseNameRequired
	}

	mariadbClient, err := s.clientTools.GetExecutable(spec.Version, ExecutableMariadb)
	if err != nil {
		return err
	}

	return mysqlfamily.Restore(ctx, mysqlfamily.RestoreRun{
		Product:               "MariaDB",
		ClientToolPath:        mariadbClient,
		Connection:            s.getClientConnection(),
		Args:                  s.buildRestoreArgs(spec),
		SourceDatabaseName:    spec.SourceDatabaseName,
		ClassifyEngineFailure: classifyGaleraFailure,
		Logger:                spec.Logger,
	}, input)
}

func (s *Source) buildRestoreArgs(spec RestoreSpec) []string {
	args := []string{
		"--host=" + s.settings.Host,
		"--port=" + strconv.Itoa(s.settings.Port),
		"--user=" + s.settings.Username,
		"--verbose",
	}

	// Galera replication is turned off for the restore session, so a large restore does not fail
	// with "Maximum writeset size exceeded". wsrep_on exists from MariaDB 10.1 in every build, and
	// it is a no-op on a server without Galera.
	if spec.Version != Version55 && !spec.IsSkipGaleraDisable {
		args = append(args, "--init-command=SET SESSION wsrep_on=OFF")
	}

	args = append(args, "--max-allowed-packet=1G")
	args = append(args, s.getSSLArgs()...)

	return append(args, s.settings.Database)
}

func classifyGaleraFailure(stderr string) error {
	if !strings.Contains(strings.ToLower(stderr), "writeset size exceeded") {
		return nil
	}

	return fmt.Errorf(
		"MariaDB Galera Cluster writeset size limit exceeded. "+
			"Try increasing wsrep_max_ws_size on your cluster nodes. stderr: %s",
		stderr,
	)
}
