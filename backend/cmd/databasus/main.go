// Command databasus runs the Databasus server, or one of its console commands when a command flag
// is given. It parses the arguments, prints the result and picks the exit code; everything else
// happens in internal/app.
package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"databasus-backend/internal/app"
	"databasus-backend/internal/lib/config"
	"databasus-backend/internal/lib/shutdown"
)

const (
	serverAddress      = ":4005"
	logFlushTimeout    = 5 * time.Second
	telemetryAnnouncer = "Anonymous telemetry collected (Databasus version, OS/arch, etc.). " +
		"No DB contents, no user data. To disable, set IS_DISABLE_ANONYMOUS_TELEMETRY=true in your .env"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, arguments []string, standardOutput, errorOutput io.Writer) int {
	if len(arguments) > 0 && arguments[0] == healthcheckSubcommand {
		return runHealthcheck(ctx, standardOutput, errorOutput)
	}

	options, err := parseOptions(arguments, errorOutput)
	if err != nil {
		return 2
	}

	command, err := selectCommand(options)
	if err != nil {
		_, _ = fmt.Fprintln(errorOutput, err.Error())

		return 1
	}

	environment, err := app.LoadEnvironment(ctx, config.LoadSpec{
		IsDatabaseRequired:       command.kind != commandProbeStorage,
		RootDirectory:            "",
		PublishedDatabaseDsnPath: "",
	})
	if err != nil {
		return 1
	}

	switch command.kind {
	case commandProbeStorage:
		return probeStorage(ctx, environment, standardOutput, errorOutput)
	case commandServe:
		return serve(ctx, environment, standardOutput)
	case commandConsole:
		return runConsoleCommand(ctx, environment, command.console, standardOutput)
	default:
		return 1
	}
}

func probeStorage(ctx context.Context, environment *app.Environment, standardOutput, errorOutput io.Writer) int {
	defer flushLogs(ctx, environment)

	err := app.ProbeStorage(ctx, environment)
	if err != nil {
		_, _ = fmt.Fprintf(errorOutput, "ERROR: %v\n", err)

		return 1
	}

	_, _ = fmt.Fprintln(standardOutput, "storage test passed")

	return 0
}

func serve(ctx context.Context, environment *app.Environment, standardOutput io.Writer) int {
	defer flushLogs(ctx, environment)

	application, err := app.New(ctx, environment)
	if err != nil {
		environment.Logging.GetLogger().ErrorContext(ctx, "failed to start the application", slog.Any("error", err))

		return 1
	}

	if !environment.Config.IsDisableAnonymousTelemetry {
		_, _ = fmt.Fprintln(standardOutput, telemetryAnnouncer)
	}

	stopSignalCtx, stopListening := shutdown.ListenForStopSignal(ctx)
	defer stopListening()

	err = application.Serve(stopSignalCtx, serverAddress)
	if err != nil {
		environment.Logging.GetLogger().ErrorContext(ctx, "failed to serve", slog.Any("error", err))

		return 1
	}

	return 0
}

func runConsoleCommand(
	ctx context.Context,
	environment *app.Environment,
	command app.ConsoleCommand,
	standardOutput io.Writer,
) int {
	defer flushLogs(ctx, environment)

	application, err := app.New(ctx, environment)
	if err != nil {
		environment.Logging.GetLogger().ErrorContext(ctx, "failed to start the application", slog.Any("error", err))

		return 1
	}

	output, err := application.RunConsoleCommand(ctx, command)
	if err != nil {
		return 1
	}

	if output != "" {
		_, _ = fmt.Fprintln(standardOutput, output)
	}

	return 0
}

// flushLogs runs before exit because the file and OTLP sinks buffer, and the line describing a
// failure is the one an operator needs in the forensic copy.
func flushLogs(ctx context.Context, environment *app.Environment) {
	flushCtx, cancelFlush := context.WithTimeout(context.WithoutCancel(ctx), logFlushTimeout)
	defer cancelFlush()

	_ = environment.Logging.Shutdown(flushCtx)
}
