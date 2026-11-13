package containers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

type dockerExec struct {
	ContainerID string
	Command     []string
	Stdin       io.ReadCloser
}

type dockerExecOutput struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// runDockerExec streams Stdin into the command, which testcontainers' Exec cannot do. Image import
// pipes gigabytes through it, so nothing is buffered on the host.
func runDockerExec(ctx context.Context, docker *client.Client, spec dockerExec) (dockerExecOutput, error) {
	created, err := docker.ExecCreate(ctx, spec.ContainerID, client.ExecCreateOptions{
		AttachStdin:  spec.Stdin != nil,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          spec.Command,
	})
	if err != nil {
		return dockerExecOutput{}, fmt.Errorf("create exec %q: %w", spec.Command, err)
	}

	attached, err := docker.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return dockerExecOutput{}, fmt.Errorf("attach exec %q: %w", spec.Command, err)
	}
	defer attached.Close()
	cancellationDone := make(chan struct{})
	stopCancellation := context.AfterFunc(ctx, func() {
		attached.Close()
		if spec.Stdin != nil {
			_ = spec.Stdin.Close()
		}
		close(cancellationDone)
	})
	defer func() {
		if !stopCancellation() {
			<-cancellationDone
		}
	}()

	stdinDone := make(chan error, 1)
	if spec.Stdin != nil {
		go func() {
			_, copyErr := io.Copy(attached.Conn, spec.Stdin)
			stdinDone <- errors.Join(copyErr, attached.CloseWrite())
		}()
	} else {
		stdinDone <- nil
	}

	var output dockerExecOutput
	var stdout, stderr bytes.Buffer
	_, readErr := stdcopy.StdCopy(&stdout, &stderr, attached.Reader)
	output.Stdout = stdout.Bytes()
	output.Stderr = stderr.Bytes()
	if readErr != nil {
		attached.Close()
		if spec.Stdin != nil {
			_ = spec.Stdin.Close()
		}
	}

	inputErr := <-stdinDone
	if ctx.Err() != nil {
		return output, fmt.Errorf("exec %q: %w", spec.Command, ctx.Err())
	}

	if readErr != nil {
		return output, fmt.Errorf("read exec %q output: %w", spec.Command, readErr)
	}

	if inputErr != nil {
		return output, fmt.Errorf("stream exec %q input: %w", spec.Command, inputErr)
	}

	inspection, err := docker.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
	if err != nil {
		return output, fmt.Errorf("inspect exec %q: %w", spec.Command, err)
	}

	output.ExitCode = inspection.ExitCode

	return output, nil
}
