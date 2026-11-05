package mysqlfamily

import (
	"context"
	"io"
	"time"

	"databasus-backend/internal/lib/process"
)

// clientToolKillAfter bounds how long a run waits for more of the tool's output after the tool
// exited: a process the tool left behind can hold that output open.
const clientToolKillAfter = 10 * time.Second

type clientToolRun struct {
	path   string
	args   []string
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

// runClientTool empties MYSQL_PWD, so a password in the backend's own environment cannot override
// the one in the .my.cnf.
func runClientTool(ctx context.Context, run clientToolRun) error {
	return process.Run(ctx, process.Spec{
		Path:       run.path,
		Args:       run.args,
		Env:        []string{"MYSQL_PWD=", "LC_ALL=C.UTF-8", "LANG=C.UTF-8"},
		Dir:        "",
		Stdin:      run.stdin,
		Stdout:     run.stdout,
		Stderr:     run.stderr,
		StopSignal: nil,
		KillAfter:  clientToolKillAfter,
	})
}
