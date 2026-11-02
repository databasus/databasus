package mongodb

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"databasus-backend/internal/lib/process"
)

const (
	// clientToolKillAfter bounds how long a run waits for more of the tool's output after the tool
	// exited: a process the tool left behind can hold that output open.
	clientToolKillAfter = 10 * time.Second
	uriArgPrefix        = "--uri="
	maskedURIArg        = uriArgPrefix + "mongodb://***:***@***"
	// maxToolParallelism caps the parallel collections of a dump and the insertion workers of a
	// restore, balancing speed against the load on the server.
	maxToolParallelism = 16
)

type clientToolRun struct {
	path   string
	args   []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// runClientTool starts mongodump or mongorestore in a process group of its own, so an ended ctx
// stops the tool together with its children.
func runClientTool(ctx context.Context, run clientToolRun) error {
	return process.Run(ctx, process.Spec{
		Path:       run.path,
		Args:       run.args,
		Env:        []string{"LC_ALL=C.UTF-8", "LANG=C.UTF-8"},
		Dir:        "",
		Stdin:      run.stdin,
		Stdout:     run.stdout,
		Stderr:     run.stderr,
		StopSignal: nil,
		KillAfter:  clientToolKillAfter,
	})
}

// maskURIArgs hides the URI, which carries the password, from the log.
func maskURIArgs(args []string) []string {
	maskedArgs := make([]string, len(args))

	for i, arg := range args {
		if strings.HasPrefix(arg, uriArgPrefix) {
			maskedArgs[i] = maskedURIArg
		} else {
			maskedArgs[i] = arg
		}
	}

	return maskedArgs
}

// buildToolFailure wraps the tool's own exit error, whose text the fail messages have always
// quoted, rather than the lib/process wrapping around it.
func buildToolFailure(toolPath string, runErr error, stderr string) error {
	if exitErr, isExitErr := errors.AsType[*exec.ExitError](runErr); isExitErr {
		runErr = exitErr
	}

	if stderr != "" {
		return fmt.Errorf("%s failed: %w\nstderr: %s", filepath.Base(toolPath), runErr, stderr)
	}

	return fmt.Errorf("%s failed: %w", filepath.Base(toolPath), runErr)
}

func containsIgnoreCase(text, substring string) bool {
	return strings.Contains(strings.ToLower(text), strings.ToLower(substring))
}
