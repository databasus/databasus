package mysqlfamily

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
)

// RestoreRun is one restore through the mysql or mariadb client, as the engine package describes it.
type RestoreRun struct {
	// Product names the server in the fail messages: MySQL or MariaDB.
	Product            string
	ClientToolPath     string
	Connection         ClientConnection
	Args               []string
	SourceDatabaseName string
	// ClassifyEngineFailure reads the failures only one engine reports, after the shared ones, and
	// returns nil for anything else. It may be nil.
	ClassifyEngineFailure func(stderr string) error
	Logger                *slog.Logger
}

// Restore feeds the SQL read from input into the client's standard input.
func Restore(ctx context.Context, run RestoreRun, input io.Reader) error {
	configFile, err := writeClientConfigFile(run.Connection)
	if err != nil {
		return fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer configFile.remove()

	args := append([]string{configFile.getDefaultsFileArg()}, run.Args...)

	run.Logger.InfoContext(ctx, "executing restore command",
		slog.String("product", run.Product),
		slog.String("client_bin", run.ClientToolPath),
		slog.Any("args", args),
	)

	var stderr bytes.Buffer

	err = runClientTool(ctx, clientToolRun{
		path:   run.ClientToolPath,
		args:   args,
		stdin:  input,
		stdout: nil,
		stderr: &stderr,
	})
	if err != nil {
		return classifyRestoreFailure(clientToolFailure{
			product:  run.Product,
			toolPath: run.ClientToolPath,
			runErr:   err,
			stderr:   stderr.String(),
		}, run)
	}

	return nil
}
