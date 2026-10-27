package mongodb

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
)

type RestoreSpec struct {
	ClientTools ClientTools
	// SourceDatabaseName is the database the archive was dumped from; its collections are renamed
	// into the target database when the names differ.
	SourceDatabaseName string
	// InsertionWorkers above 1 inserts into each collection with that many workers, capped at 16.
	InsertionWorkers int
	Logger           *slog.Logger
}

// Restore feeds the gzip-compressed mongodump archive read from input into mongorestore, which
// drops each collection before restoring it.
func (s *Source) Restore(ctx context.Context, spec RestoreSpec, input io.Reader) error {
	if s.settings.Database == "" {
		return ErrRestoreDatabaseNameRequired
	}

	mongorestore := spec.ClientTools.GetExecutable(ExecutableMongorestore)
	args := s.buildRestoreArgs(spec)

	spec.Logger.InfoContext(ctx, "executing MongoDB restore command",
		slog.String("command", mongorestore),
		slog.Any("args", maskURIArgs(args)),
	)

	var stderr bytes.Buffer

	err := runClientTool(ctx, clientToolRun{
		path:   mongorestore,
		args:   args,
		stdin:  input,
		stdout: nil,
		stderr: &stderr,
	})
	if err != nil {
		return classifyRestoreFailure(mongorestore, err, stderr.String())
	}

	return nil
}

func (s *Source) buildRestoreArgs(spec RestoreSpec) []string {
	targetDatabaseName := s.settings.Database

	args := []string{uriArgPrefix + s.settings.BuildRestoreURI(), "--archive", "--gzip", "--drop"}

	if spec.SourceDatabaseName != "" && spec.SourceDatabaseName != targetDatabaseName {
		args = append(args, "--nsFrom="+spec.SourceDatabaseName+".*", "--nsTo="+targetDatabaseName+".*")
	} else {
		args = append(args, "--nsInclude="+targetDatabaseName+".*")
	}

	insertionWorkers := max(1, min(spec.InsertionWorkers, maxToolParallelism))
	if insertionWorkers > 1 {
		args = append(args, "--numInsertionWorkersPerCollection="+strconv.Itoa(insertionWorkers))
	}

	return args
}

func classifyRestoreFailure(mongorestore string, runErr error, stderr string) error {
	switch {
	case containsIgnoreCase(stderr, "authentication failed"):
		return fmt.Errorf("MongoDB authentication failed. Check username and password. stderr: %s", stderr)
	case containsIgnoreCase(stderr, "connection refused") || containsIgnoreCase(stderr, "server selection error"):
		return fmt.Errorf(
			"MongoDB connection refused. Check if the server is running and accessible. stderr: %s", stderr,
		)
	case containsIgnoreCase(stderr, "timeout"):
		return fmt.Errorf("MongoDB connection timeout. stderr: %s", stderr)
	default:
		return buildToolFailure(mongorestore, runErr, stderr)
	}
}
