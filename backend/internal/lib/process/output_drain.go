package process

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"databasus-backend/internal/lib/background"
)

// outputDrain copies a tool's standard output into its writer. os/exec would do the same, but its
// WaitDelay also counts the time the writer takes, so a slow storage would fail a finished dump.
// Here the limit after the tool's exit covers only waiting for more output, which is what tells a
// busy writer from a descendant that holds the pipe open and writes nothing.
type outputDrain struct {
	reader    *os.File
	toolEnd   *os.File
	target    io.Writer
	readLimit time.Duration

	isToolExited atomic.Bool
	done         chan struct{}
	copyErr      error
}

func newOutputDrain(target io.Writer, readLimit time.Duration) (*outputDrain, error) {
	reader, toolEnd, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	return &outputDrain{
		reader:       reader,
		toolEnd:      toolEnd,
		target:       target,
		readLimit:    readLimit,
		isToolExited: atomic.Bool{},
		done:         make(chan struct{}),
		copyErr:      nil,
	}, nil
}

// Read refreshes the deadline every time, because a read on an expired deadline fails before it
// looks at the pipe, and a long write can outlast the previous deadline with output still buffered.
func (d *outputDrain) Read(p []byte) (int, error) {
	if d.isToolExited.Load() {
		d.refreshReadDeadline()
	}

	return d.reader.Read(p)
}

// start closes the parent's copy of the tool's end, so the pipe reports EOF once the tool and its
// descendants are gone.
func (d *outputDrain) start(ctx context.Context) {
	_ = d.toolEnd.Close()

	go func() {
		defer close(d.done)
		// A closed read end stops a tool that is still writing, as it does with os/exec.
		defer func() { _ = d.reader.Close() }()

		d.copyErr = background.RunRecovering(ctx, func(context.Context) error {
			_, err := io.Copy(d.target, d)
			if errors.Is(err, os.ErrDeadlineExceeded) {
				return exec.ErrWaitDelay
			}

			return err
		})
	}()
}

func (d *outputDrain) closeBothEnds() {
	_ = d.toolEnd.Close()
	_ = d.reader.Close()
}

func (d *outputDrain) closeReader() {
	_ = d.reader.Close()
}

// startLimitingReads also limits a read that is already blocked.
func (d *outputDrain) startLimitingReads() {
	d.isToolExited.Store(true)
	d.refreshReadDeadline()
}

func (d *outputDrain) refreshReadDeadline() {
	if d.readLimit > 0 {
		_ = d.reader.SetReadDeadline(time.Now().Add(d.readLimit))
	}
}

// attachOutputDrain gives the tool the drain's pipe as Stdout. A writer also passed as Stderr shares
// that pipe, so its writes stay in order and never run concurrently.
func attachOutputDrain(cmd *exec.Cmd, spec Spec) (*outputDrain, error) {
	if !needsOutputDrain(spec.Stdout) {
		return nil, nil
	}

	drain, err := newOutputDrain(spec.Stdout, spec.KillAfter)
	if err != nil {
		return nil, err
	}

	cmd.Stdout = drain.toolEnd
	if isSameWriter(spec.Stdout, spec.Stderr) {
		cmd.Stderr = drain.toolEnd
	}

	return drain, nil
}

func needsOutputDrain(stdout io.Writer) bool {
	if stdout == nil {
		return false
	}

	_, isFile := stdout.(*os.File)

	return !isFile
}

// isSameWriter recovers because comparing two writers of an uncomparable type panics.
func isSameWriter(first, second io.Writer) (isSame bool) {
	defer func() {
		if recover() != nil {
			isSame = false
		}
	}()

	return first == second
}
