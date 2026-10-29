package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"databasus-backend/internal/agent/config"
)

type configWarningOutput struct {
	log    *slog.Logger
	stderr io.Writer
}

func warnConfigPermissions(ctx context.Context, warning config.PermissionWarning, output configWarningOutput) {
	chmodCommand := "chmod 600 " + warning.Path
	_, _ = fmt.Fprintf(output.stderr,
		"WARNING: other users can read the agent token\nFile: %s\nMode: %04o\nOwner: %s\nError: %v\nFix: %s\n",
		warning.Path, warning.Mode, warning.Owner, warning.Err, chmodCommand)
	attributes := []slog.Attr{
		slog.String("path", warning.Path),
		slog.String("mode", fmt.Sprintf("%04o", warning.Mode)),
		slog.String("owner", warning.Owner),
		slog.Any("error", warning.Err),
		slog.String("risk", "other users can read the agent token"),
		slog.String("chmod_command", chmodCommand),
	}
	if warning.OwnerUID >= 0 && warning.OwnerUID != warning.AgentUID {
		chownCommand := "sudo chown " + warning.AgentUser + " " + warning.Path
		_, _ = fmt.Fprintf(output.stderr, "Change ownership: %s\n", chownCommand)
		attributes = append(attributes, slog.String("chown_command", chownCommand))
	}
	output.log.LogAttrs(ctx, slog.LevelWarn, "warning: agent config file is readable by other users", attributes...)
}
