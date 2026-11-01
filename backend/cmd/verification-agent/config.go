package main

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"databasus-backend/internal/agent/config"
)

func loadConfig(ctx context.Context, options configLoadOptions) (*config.Config, error) {
	path, err := filepath.Abs("databasus-verification.json")
	if err != nil {
		return nil, err
	}

	warning, err := config.EnsurePrivateFile(path, options.chmod)
	if err != nil {
		return nil, err
	}
	if warning != nil {
		warnConfigPermissions(ctx, *warning, configWarningOutput{log: options.log, stderr: options.stderr})
	}

	cfg := &config.Config{}
	if options.flags == nil {
		cfg.LoadFromJSON(ctx, options.log)
		return cfg, nil
	}
	if err := cfg.LoadFromJSONAndArgs(ctx, options.log, options.flags, options.args); err != nil {
		return nil, err
	}
	return cfg, nil
}

type configLoadOptions struct {
	flags  *flag.FlagSet
	args   []string
	log    *slog.Logger
	stderr io.Writer
	chmod  func(string, os.FileMode) error
}
