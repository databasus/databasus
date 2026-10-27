package mysqlfamily

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
)

// DumpRun is one dump through mysqldump or mariadb-dump, as the engine package describes it.
type DumpRun struct {
	// Product names the server in the fail messages: MySQL or MariaDB.
	Product      string
	DumpToolPath string
	Connection   ClientConnection
	DatabaseName string
	// Args follow the defaults file and the negotiated network compression.
	Args                  []string
	CompressionCandidates NetworkCompressionCandidates
	Logger                *slog.Logger
}

// Dump writes the database as SQL into output until the tool exits. The SQL is uncompressed; the
// engine packages tell the caller to compress what it stores.
func Dump(ctx context.Context, run DumpRun, output io.Writer) error {
	configFile, err := writeClientConfigFile(run.Connection)
	if err != nil {
		return fmt.Errorf("failed to create .my.cnf: %w", err)
	}
	defer configFile.remove()

	compressionArgs := negotiateNetworkCompression(ctx, compressionProbe{
		product:      run.Product,
		dumpToolPath: run.DumpToolPath,
		configFile:   configFile,
		databaseName: run.DatabaseName,
		candidates:   run.CompressionCandidates,
		logger:       run.Logger,
	})

	args := append([]string{configFile.getDefaultsFileArg()}, compressionArgs...)
	args = append(args, run.Args...)

	run.Logger.InfoContext(ctx, "executing backup command",
		slog.String("product", run.Product),
		slog.String("dump_bin", run.DumpToolPath),
		slog.Any("args", args),
	)

	var stderr bytes.Buffer

	err = runClientTool(ctx, clientToolRun{
		path:   run.DumpToolPath,
		args:   args,
		stdin:  nil,
		stdout: output,
		stderr: &stderr,
	})
	if err != nil {
		return classifyDumpFailure(clientToolFailure{
			product:  run.Product,
			toolPath: run.DumpToolPath,
			runErr:   err,
			stderr:   stderr.String(),
		})
	}

	return nil
}
