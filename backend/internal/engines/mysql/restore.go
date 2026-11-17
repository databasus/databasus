package mysql

import (
	"context"
	"io"
	"log/slog"
	"strconv"

	"databasus-backend/internal/engines/mysqlfamily"
)

type RestoreSpec struct {
	Version            Version
	SourceDatabaseName string
	Logger             *slog.Logger
}

// Restore feeds the SQL read from input into the mysql client.
func (s *Source) Restore(ctx context.Context, spec RestoreSpec, input io.Reader) error {
	if s.settings.Database == "" {
		return ErrRestoreDatabaseNameRequired
	}

	if err := s.settings.Validate(); err != nil {
		return err
	}

	mysqlClient, err := s.clientTools.GetExecutable(spec.Version, ExecutableMysql)
	if err != nil {
		return err
	}

	return mysqlfamily.Restore(ctx, mysqlfamily.RestoreRun{
		Product:               "MySQL",
		ClientToolPath:        mysqlClient,
		Connection:            s.getClientConnection(),
		Args:                  s.buildRestoreArgs(),
		SourceDatabaseName:    spec.SourceDatabaseName,
		ClassifyEngineFailure: nil,
		Logger:                spec.Logger,
	}, input)
}

func (s *Source) buildRestoreArgs() []string {
	return []string{
		"--host=" + s.settings.Host,
		"--port=" + strconv.Itoa(s.settings.Port),
		"--user=" + s.settings.Username,
		"--verbose",
		"--max-allowed-packet=1G",
		s.getSSLModeArg(),
		"--database=" + s.settings.Database,
	}
}
