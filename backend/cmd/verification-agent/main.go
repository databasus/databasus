package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"databasus-backend/internal/agent/api"
	"databasus-backend/internal/agent/logging"
	"databasus-backend/internal/agent/start"
	"databasus-backend/internal/agent/upgrade"
)

//nolint:gochecknoglobals // The release build sets version metadata through the linker; runtime code never changes it.
var Version = "dev"

func main() { os.Exit(runCommand(os.Args[1:])) }

func runCommand(args []string) int {
	if len(args) == 0 {
		printUsage()
		return 1
	}
	switch args[0] {
	case "version":
		fmt.Println(Version)
		return 0
	case "start", "run", "_run", "stop", "status":
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printUsage()
		return 1
	}
	log, closeLog := newLogger()
	defer closeLog()
	switch args[0] {
	case "start":
		return runStart(args[1:], log)
	case "run":
		return runAgent(args[1:], log)
	case "_run":
		return runDaemon(args[1:], log)
	case "stop":
		return runStop(log)
	case "status":
		return runStatus()
	}
	return 1
}

func runStart(args []string, log *slog.Logger) int {
	fs := flag.NewFlagSet("start", flag.ExitOnError)

	isSkipUpdate := fs.Bool("skip-update", false, "Skip auto-update check")

	cfg, err := loadConfig(
		context.Background(),
		configLoadOptions{flags: fs, args: args, log: log, stderr: os.Stderr, chmod: os.Chmod},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := cfg.SaveToJSON(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
	}

	isDev := checkIsDevelopment()
	if err := runUpdateCheck(cfg.DatabasusHost, *isSkipUpdate, isDev, log); err != nil {
		return 1
	}

	pid, err := start.Start(context.Background(), cfg, log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Printf("Agent started in background (PID %d)\n", pid)
	return 0
}

func runAgent(args []string, log *slog.Logger) int {
	fs := flag.NewFlagSet("run", flag.ExitOnError)

	isSkipUpdate := fs.Bool("skip-update", false, "Skip auto-update check")

	cfg, err := loadConfig(
		context.Background(),
		configLoadOptions{flags: fs, args: args, log: log, stderr: os.Stderr, chmod: os.Chmod},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := cfg.SaveToJSON(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
	}

	isDev := checkIsDevelopment()
	if err := runUpdateCheck(cfg.DatabasusHost, *isSkipUpdate, isDev, log); err != nil {
		return 1
	}

	if err := start.RunDaemon(context.Background(), cfg, Version, isDev, log); err != nil {
		if errors.Is(err, upgrade.ErrUpgradeRestart) {
			return reexecExitCode(log)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func runDaemon(args []string, log *slog.Logger) int {
	fs := flag.NewFlagSet("_run", flag.ExitOnError)

	if err := fs.Parse(args); err != nil {
		return 1
	}

	cfg, err := loadConfig(context.Background(), configLoadOptions{log: log, stderr: os.Stderr, chmod: os.Chmod})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := start.RunDaemon(context.Background(), cfg, Version, checkIsDevelopment(), log); err != nil {
		if errors.Is(err, upgrade.ErrUpgradeRestart) {
			return reexecExitCode(log)
		}

		log.ErrorContext(context.Background(), "agent exited with error", slog.Any("error", err))
		return 1
	}
	return 0
}

func runStop(log *slog.Logger) int {
	if err := start.Stop(context.Background(), log); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func runStatus() int {
	status, err := start.GetDaemonStatus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	fmt.Println(status)
	return 0
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: databasus-verification-agent <command> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  start    Start the verification agent in the background (daemon)")
	fmt.Fprintln(os.Stderr, "  run      Run the verification agent in the foreground (systemd / container)")
	fmt.Fprintln(os.Stderr, "  stop     Stop a running agent")
	fmt.Fprintln(os.Stderr, "  status   Show agent status")
	fmt.Fprintln(os.Stderr, "  version  Print agent version")
}

func runUpdateCheck(host string, isSkipUpdate, isDev bool, log *slog.Logger) error {
	if isSkipUpdate {
		return nil
	}

	if host == "" {
		return nil
	}

	apiClient := api.NewClient(api.Deps{Host: host, Token: "", AgentID: "", Logger: log})

	isUpgraded, err := upgrade.New(upgrade.Deps{APIClient: apiClient, Logger: log}).
		CheckAndUpdate(context.Background(), Version, isDev)
	if err != nil {
		log.ErrorContext(context.Background(), "auto-update failed", slog.Any("error", err))
		return err
	}

	if isUpgraded {
		return reexecAfterUpgrade(log)
	}
	return nil
}

func checkIsDevelopment() bool {
	dir, err := os.Getwd()
	if err != nil {
		return false
	}

	for range 3 {
		if data, err := os.ReadFile(filepath.Join(dir, ".env")); err == nil {
			return parseEnvMode(data)
		}

		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return false
		}

		dir = filepath.Dir(dir)
	}

	return false
}

func parseEnvMode(data []byte) bool {
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) == "ENV_MODE" {
			return strings.TrimSpace(parts[1]) == "development"
		}
	}

	return false
}

func newLogger() (*slog.Logger, func()) {
	log, closer, err := logging.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}
	return log, func() {
		if closer != nil {
			_ = closer.Close()
		}
	}
}

func reexecExitCode(log *slog.Logger) int {
	_ = reexecAfterUpgrade(log)
	return 1
}
