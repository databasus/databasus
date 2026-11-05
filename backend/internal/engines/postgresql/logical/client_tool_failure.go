package logical

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// The exit codes pg_dump and pg_restore report. -1073741819 is 0xC0000005, an access violation.
const (
	exitCodeAccessViolation = -1073741819
	exitCodeGenericError    = 1
	exitCodeConnectionError = 2
)

// clientToolFailure is a pg_dump or pg_restore run that exited with an error. The messages built
// from it are the fail messages users see, so their wording stays as it is.
type clientToolFailure struct {
	toolPath string
	args     []string
	runErr   error
	stderr   string
	logger   *slog.Logger
}

// getExitError returns the tool's own exit error, whose text the messages quote, rather than the
// lib/process wrapping around it.
func (f clientToolFailure) getExitError() (*exec.ExitError, bool) {
	return errors.AsType[*exec.ExitError](f.runErr)
}

func (f clientToolFailure) getGenericMessage() string {
	runErr := f.runErr
	if exitErr, isExitErr := f.getExitError(); isExitErr {
		runErr = exitErr
	}

	return fmt.Sprintf("%s failed: %v – stderr: %s", filepath.Base(f.toolPath), runErr, f.stderr)
}

func (f clientToolFailure) getAccessViolationMessage() string {
	return fmt.Sprintf(
		"%s crashed with access violation (0xC0000005). "+
			"This may indicate incompatible PostgreSQL version, corrupted installation, or connection issues. "+
			"stderr: %s",
		filepath.Base(f.toolPath),
		f.stderr,
	)
}

func (f clientToolFailure) getCommandLine() string {
	return f.toolPath + " " + strings.Join(f.args, " ")
}

func classifyDumpFailure(ctx context.Context, failure clientToolFailure, password string) error {
	exitErr, isExitErr := failure.getExitError()
	if !isExitErr {
		return errors.New(failure.getGenericMessage())
	}

	switch exitCode := exitErr.ExitCode(); {
	case exitCode == exitCodeGenericError && strings.TrimSpace(failure.stderr) == "":
		failure.logger.ErrorContext(ctx, "pg_dump failed with exit status 1 but no stderr output",
			slog.String("pg_bin", failure.toolPath),
			slog.Any("args", failure.args),
			slog.Any("env_vars", []string{
				"PGCLIENTENCODING=UTF8",
				"PGCONNECT_TIMEOUT=" + strconv.Itoa(clientConnectTimeoutSeconds),
				"LC_ALL=C.UTF-8",
				"LANG=C.UTF-8",
			}),
		)

		return fmt.Errorf(
			"%s failed with exit status 1 but provided no error details. "+
				"This often indicates: "+
				"1) Connection timeout or refused connection, "+
				"2) Authentication failure with incorrect credentials, "+
				"3) Database does not exist, "+
				"4) Network connectivity issues, "+
				"5) PostgreSQL server not running. "+
				"Command executed: %s",
			filepath.Base(failure.toolPath),
			failure.getCommandLine(),
		)
	case exitCode == exitCodeAccessViolation:
		failure.logger.ErrorContext(ctx, "postgreSQL tool crashed with access violation",
			slog.String("pg_bin", failure.toolPath),
			slog.String("exit_code", "0xC0000005"),
		)

		return errors.New(failure.getAccessViolationMessage())
	case exitCode == exitCodeGenericError || exitCode == exitCodeConnectionError:
		return classifyDumpConnectionFailure(failure.stderr, password)
	default:
		return errors.New(failure.getGenericMessage())
	}
}

func classifyDumpConnectionFailure(stderr, password string) error {
	switch {
	case containsIgnoreCase(stderr, "pg_hba.conf"):
		return fmt.Errorf(
			"PostgreSQL connection rejected by server configuration (pg_hba.conf). "+
				"The server may not allow connections from your IP address or may require different authentication settings. "+
				"stderr: %s",
			stderr,
		)
	case containsIgnoreCase(stderr, "no password supplied") || containsIgnoreCase(stderr, "fe_sendauth"):
		return fmt.Errorf(
			"PostgreSQL authentication failed - no password supplied. "+
				"The server asked for a password, but the password file given to pg_dump held none for this connection. "+
				"Password length: %d, Password empty: %v. "+
				"stderr: %s",
			len(password),
			password == "",
			stderr,
		)
	case containsIgnoreCase(stderr, "ssl") && containsIgnoreCase(stderr, "connection"):
		return fmt.Errorf(
			"PostgreSQL SSL connection failed. "+
				"The server may require SSL encryption or have SSL configuration issues. "+
				"stderr: %s",
			stderr,
		)
	case containsIgnoreCase(stderr, "connection") && containsIgnoreCase(stderr, "refused"):
		return fmt.Errorf(
			"PostgreSQL connection refused. "+
				"Check if the server is running and accessible from your network. "+
				"stderr: %s",
			stderr,
		)
	case containsIgnoreCase(stderr, "authentication") || containsIgnoreCase(stderr, "password"):
		return fmt.Errorf("PostgreSQL authentication failed. Check username and password. stderr: %s", stderr)
	case containsIgnoreCase(stderr, "timeout"):
		return fmt.Errorf(
			"PostgreSQL connection timeout. The server may be unreachable or overloaded. stderr: %s",
			stderr,
		)
	default:
		return fmt.Errorf("PostgreSQL connection or authentication error. stderr: %s", stderr)
	}
}

// restoreDatabaseNames name the databases in the message about a missing target database.
type restoreDatabaseNames struct {
	source string
	target string
}

func classifyRestoreFailure(failure clientToolFailure, databaseNames restoreDatabaseNames) error {
	exitErr, isExitErr := failure.getExitError()
	if !isExitErr {
		return errors.New(failure.getGenericMessage())
	}

	switch exitCode := exitErr.ExitCode(); {
	case exitCode == exitCodeGenericError && strings.TrimSpace(failure.stderr) == "":
		return fmt.Errorf(
			"%s failed with exit status 1 but provided no error details. "+
				"This often indicates: "+
				"1) Connection timeout or refused connection, "+
				"2) Authentication failure with incorrect credentials, "+
				"3) Database does not exist, "+
				"4) Network connectivity issues, "+
				"5) PostgreSQL server not running, "+
				"6) Backup file is corrupted or incompatible. "+
				"Command executed: %s",
			filepath.Base(failure.toolPath),
			failure.getCommandLine(),
		)
	case exitCode == exitCodeAccessViolation:
		return errors.New(failure.getAccessViolationMessage())
	case exitCode == exitCodeGenericError || exitCode == exitCodeConnectionError:
		return classifyRestoreConnectionFailure(failure, databaseNames)
	default:
		return errors.New(failure.getGenericMessage())
	}
}

func classifyRestoreConnectionFailure(failure clientToolFailure, databaseNames restoreDatabaseNames) error {
	stderr := failure.stderr

	switch {
	case containsIgnoreCase(stderr, "pg_hba.conf"):
		return fmt.Errorf("PostgreSQL connection rejected by server configuration (pg_hba.conf). stderr: %s", stderr)
	case containsIgnoreCase(stderr, "no password supplied") || containsIgnoreCase(stderr, "fe_sendauth"):
		return fmt.Errorf("PostgreSQL authentication failed - no password supplied. stderr: %s", stderr)
	case containsIgnoreCase(stderr, "ssl") && containsIgnoreCase(stderr, "connection"):
		return fmt.Errorf("PostgreSQL SSL connection failed. stderr: %s", stderr)
	case containsIgnoreCase(stderr, "connection") && containsIgnoreCase(stderr, "refused"):
		return fmt.Errorf(
			"PostgreSQL connection refused. Check if the server is running and accessible. stderr: %s",
			stderr,
		)
	case containsIgnoreCase(stderr, "authentication") || containsIgnoreCase(stderr, "password"):
		return fmt.Errorf("PostgreSQL authentication failed. Check username and password. stderr: %s", stderr)
	case containsIgnoreCase(stderr, "timeout"):
		return fmt.Errorf("PostgreSQL connection timeout. stderr: %s", stderr)
	case containsIgnoreCase(stderr, "database") && containsIgnoreCase(stderr, "does not exist"):
		//nolint:staticcheck // a fail message users see and match, so its wording stays
		return fmt.Errorf(
			"Target database does not exist (backup db %s, not found %s). "+
				"Create the database before restoring. stderr: %s",
			getNameOrUnknown(databaseNames.source),
			getNameOrUnknown(databaseNames.target),
			stderr,
		)
	default:
		return errors.New(failure.getGenericMessage())
	}
}

func getNameOrUnknown(name string) string {
	if name == "" {
		return "unknown"
	}

	return name
}

func containsIgnoreCase(text, substring string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substring))
}
