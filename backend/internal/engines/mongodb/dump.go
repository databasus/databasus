package mongodb

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"strconv"

	"databasus-backend/internal/lib/namelist"
)

type DumpSpec struct {
	ClientTools        ClientTools
	ExcludeCollections []string
	// ParallelCollections above 1 dumps that many collections at once, capped at 16.
	ParallelCollections int
	Logger              *slog.Logger
}

// Dump writes the database as a gzip-compressed mongodump archive into output until mongodump
// exits. The archive is compressed by mongodump itself, so the caller stores it as it is.
func (s *Source) Dump(ctx context.Context, spec DumpSpec, output io.Writer) error {
	if s.settings.Database == "" {
		return ErrDumpDatabaseNameRequired
	}

	mongodump := spec.ClientTools.GetExecutable(ExecutableMongodump)
	args := s.buildDumpArgs(spec)

	spec.Logger.InfoContext(ctx, "executing MongoDB backup command",
		slog.String("command", mongodump),
		slog.Any("args", maskURIArgs(args)),
	)

	var stderr bytes.Buffer

	err := runClientTool(ctx, clientToolRun{
		path:   mongodump,
		args:   args,
		stdin:  nil,
		stdout: output,
		stderr: &stderr,
	})
	if err != nil {
		return buildToolFailure(mongodump, err, stderr.String())
	}

	return nil
}

func (s *Source) buildDumpArgs(spec DumpSpec) []string {
	args := []string{uriArgPrefix + s.settings.BuildConnectionURI(), "--archive", "--gzip"}

	for _, excludedCollection := range namelist.NormalizeUniqueNames(spec.ExcludeCollections) {
		args = append(args, "--excludeCollection="+excludedCollection)
	}

	parallelCollections := max(1, min(spec.ParallelCollections, maxToolParallelism))
	if parallelCollections > 1 {
		args = append(args, "--numParallelCollections="+strconv.Itoa(parallelCollections))
	}

	return args
}
