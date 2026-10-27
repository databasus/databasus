package mysqlfamily

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// The exit codes mysqldump and mariadb-dump report for a failed connection or query.
const (
	exitCodeGenericError    = 1
	exitCodeConnectionError = 2
)

// clientToolFailure is a dump or restore tool run that exited with an error. The messages built
// from it are the fail messages users see, so their wording stays as it is.
type clientToolFailure struct {
	product  string
	toolPath string
	runErr   error
	stderr   string
}

func (f clientToolFailure) getGenericMessage() string {
	runErr := f.runErr
	if exitErr, isExitErr := errors.AsType[*exec.ExitError](f.runErr); isExitErr {
		runErr = exitErr
	}

	return fmt.Sprintf("%s failed: %v – stderr: %s", filepath.Base(f.toolPath), runErr, f.stderr)
}

func classifyDumpFailure(failure clientToolFailure) error {
	exitErr, isExitErr := errors.AsType[*exec.ExitError](failure.runErr)
	if !isExitErr {
		return errors.New(failure.getGenericMessage())
	}

	exitCode := exitErr.ExitCode()
	if exitCode != exitCodeGenericError && exitCode != exitCodeConnectionError {
		return errors.New(failure.getGenericMessage())
	}

	product, stderr := failure.product, failure.stderr

	switch {
	case containsIgnoreCase(stderr, "access denied"):
		return fmt.Errorf("%s access denied. Check username and password. stderr: %s", product, stderr)
	case containsIgnoreCase(stderr, "can't connect") || containsIgnoreCase(stderr, "connection refused"):
		return fmt.Errorf(
			"%s connection refused. Check if the server is running and accessible. stderr: %s", product, stderr,
		)
	case isCompressionRejection(stderr):
		return fmt.Errorf(
			"%s rejected the network compression algorithm that had just passed the "+
				"pre-flight handshake probe. Compression is re-probed on every run, so retry "+
				"the backup. stderr: %s",
			product,
			stderr,
		)
	case containsIgnoreCase(stderr, "unknown database"):
		return fmt.Errorf("%s database does not exist. stderr: %s", product, stderr)
	case containsIgnoreCase(stderr, "ssl"):
		return fmt.Errorf("%s SSL connection failed. stderr: %s", product, stderr)
	case containsIgnoreCase(stderr, "timeout"):
		return fmt.Errorf("%s connection timeout. stderr: %s", product, stderr)
	default:
		return fmt.Errorf("%s connection or authentication error. stderr: %s", product, stderr)
	}
}

func classifyRestoreFailure(failure clientToolFailure, run RestoreRun) error {
	product, stderr := failure.product, failure.stderr

	switch {
	case containsIgnoreCase(stderr, "access denied"):
		return fmt.Errorf("%s access denied. Check username and password. stderr: %s", product, stderr)
	case containsIgnoreCase(stderr, "can't connect") || containsIgnoreCase(stderr, "connection refused"):
		return fmt.Errorf(
			"%s connection refused. Check if the server is running and accessible. stderr: %s", product, stderr,
		)
	case containsIgnoreCase(stderr, "unknown database"):
		sourceDatabaseName := run.SourceDatabaseName
		if sourceDatabaseName == "" {
			sourceDatabaseName = "unknown"
		}

		return fmt.Errorf(
			"target database does not exist (backup db %s). Create the database before restoring. stderr: %s",
			sourceDatabaseName,
			stderr,
		)
	case containsIgnoreCase(stderr, "ssl"):
		return fmt.Errorf("%s SSL connection failed. stderr: %s", product, stderr)
	case containsIgnoreCase(stderr, "timeout"):
		return fmt.Errorf("%s connection timeout. stderr: %s", product, stderr)
	}

	if run.ClassifyEngineFailure != nil {
		if engineFailure := run.ClassifyEngineFailure(stderr); engineFailure != nil {
			return engineFailure
		}
	}

	return errors.New(failure.getGenericMessage())
}

// isCompressionRejection reads error 2066, CR_COMPRESSION_WRONGLY_CONFIGURED, which the client
// raises when the endpoint cannot negotiate the requested algorithm.
func isCompressionRejection(stderr string) bool {
	return containsIgnoreCase(stderr, "compression algorithm") || containsIgnoreCase(stderr, "2066")
}

func containsIgnoreCase(text, substring string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substring))
}
