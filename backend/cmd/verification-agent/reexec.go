package main

import (
	"context"
	"log/slog"
	"os"
	"syscall"
)

func reexecAfterUpgrade(log *slog.Logger) error {
	selfPath, err := os.Executable()
	if err != nil {
		log.ErrorContext(context.Background(), "failed to resolve executable for re-exec", slog.Any("error", err))
		return err
	}

	log.InfoContext(context.Background(), "re-executing after upgrade...")

	// Replaces the process with the agent's own upgraded executable.
	//nolint:forbidigo // Re-exec inherits the process environment rather than reading a setting.
	// nosemgrep: go.lang.security.audit.dangerous-syscall-exec.dangerous-syscall-exec
	if err := syscall.Exec(selfPath, os.Args, os.Environ()); err != nil {
		log.ErrorContext(context.Background(), "failed to re-exec after upgrade", slog.Any("error", err))
		return err
	}
	return nil
}
