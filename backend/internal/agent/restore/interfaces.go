package restore

import (
	"context"
	"io"
)

type ExecRunner interface {
	Exec(ctx context.Context, cmd []string, stdin io.Reader, env []string) (ExecResult, error)
}
