package logical

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"

	pgshared "databasus-backend/internal/engines/postgresql/shared"
	"databasus-backend/internal/lib/namelist"
)

const dumpCompressionLevel = 5

type DumpSpec struct {
	ClientTools      pgshared.ClientTools
	Version          pgshared.Version
	IncludeSchemas   []string
	ExcludeTables    []string
	ExcludeTableData []string
	Logger           *slog.Logger
}

// Dump writes the database as a pg_dump custom-format archive into output until pg_dump exits. The
// archive is compressed by pg_dump itself, so the caller stores it as it is.
func (s *Source) Dump(ctx context.Context, spec DumpSpec, output io.Writer) error {
	if s.settings.Database == "" {
		return ErrDumpDatabaseNameRequired
	}

	credentials, err := s.writeCredentialFiles()
	if err != nil {
		return fmt.Errorf("failed to create credential files: %w", err)
	}
	defer credentials.Remove()

	pgDump := spec.ClientTools.GetExecutable(spec.Version, pgshared.ExecutablePgDump)
	args := s.buildDumpArgs(spec)

	spec.Logger.InfoContext(
		ctx,
		"executing PostgreSQL backup command",
		slog.String("pg_bin", pgDump),
		slog.Any("args", args),
	)

	var stderr bytes.Buffer

	err = s.runClientTool(ctx, clientToolRun{
		path:        pgDump,
		args:        args,
		credentials: credentials,
		stdin:       nil,
		stdout:      output,
		stderr:      &stderr,
	})
	if err != nil {
		return classifyDumpFailure(ctx, clientToolFailure{
			toolPath: pgDump,
			args:     args,
			runErr:   err,
			stderr:   stderr.String(),
			logger:   spec.Logger,
		}, s.settings.Password)
	}

	return nil
}

func (s *Source) buildDumpArgs(spec DumpSpec) []string {
	args := []string{
		"-Fc",
		"--no-password",
		"-h", s.settings.Host,
		"-p", strconv.Itoa(s.settings.Port),
		"-U", s.settings.Username,
		"-d", pgshared.BuildDatabaseNameConninfo(s.settings.Database),
		"--verbose",
	}

	for _, includedSchema := range namelist.NormalizeUniqueNames(spec.IncludeSchemas) {
		args = append(args, "-n", includedSchema)
	}

	for _, excludedTable := range namelist.NormalizeUniqueNames(spec.ExcludeTables) {
		args = append(args, "--exclude-table="+excludedTable)
	}

	for _, dataExcludedTable := range namelist.NormalizeUniqueNames(spec.ExcludeTableData) {
		args = append(args, "--exclude-table-data="+dataExcludedTable)
	}

	return append(args, getDumpCompressionArgs(spec.Version)...)
}

// pg_dump learned zstd in PostgreSQL 16, so older versions compress with gzip.
func getDumpCompressionArgs(version pgshared.Version) []string {
	isZstdUnsupported := version == pgshared.Version12 || version == pgshared.Version13 ||
		version == pgshared.Version14 || version == pgshared.Version15
	if isZstdUnsupported {
		return []string{"-Z", strconv.Itoa(dumpCompressionLevel)}
	}

	return []string{fmt.Sprintf("--compress=zstd:%d", dumpCompressionLevel)}
}
