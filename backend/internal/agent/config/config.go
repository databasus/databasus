package config

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"
)

const configFileName = "databasus-verification.json"

const minRAMMbPerJob = 512

// DefaultVerificationPgImageRepo bundles common extensions so a dump's CREATE
// EXTENSION statements succeed during verification; override via
// verificationPgImageRepo (e.g. a private registry, or "postgres" for stock).
const DefaultVerificationPgImageRepo = "databasus/verification-postgres"

type Config struct {
	DatabasusHost           string `json:"databasusHost"`
	AgentID                 string `json:"agentId"`
	Token                   string `json:"token"`
	MaxCPU                  int    `json:"maxCpu"`
	MaxRAMMb                int    `json:"maxRamMb"`
	MaxDiskGb               int    `json:"maxDiskGb"`
	MaxConcurrentJobs       int    `json:"maxConcurrentJobs"`
	AllowInsecureHTTP       bool   `json:"allowInsecureHttp"`
	VerificationPgImageRepo string `json:"verificationPgImageRepo"`

	flagSources map[string]string
}

func (c *Config) GetVerificationPgImageRepo() string {
	if c.VerificationPgImageRepo == "" {
		return DefaultVerificationPgImageRepo
	}

	return c.VerificationPgImageRepo
}

type Capacity struct {
	MaxCPU            int
	MaxRAMMb          int
	MaxDiskGb         int
	MaxConcurrentJobs int

	CPUPerJob   int
	RAMMbPerJob int
}

// LoadFromJSONAndArgs reads databasus-verification.json into the struct
// and overrides JSON values with any explicitly provided CLI flags.
func (c *Config) LoadFromJSONAndArgs(ctx context.Context, log *slog.Logger, fs *flag.FlagSet, args []string) error {
	c.loadFromJSON(ctx, log)

	bindings := c.flagBindings()
	c.flagSources = sourcesFromConfig(bindings, c)

	appliers := make([]func(*Config, map[string]string), 0, len(bindings))
	for _, binding := range bindings {
		appliers = append(appliers, binding.register(fs))
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	for _, apply := range appliers {
		apply(c, c.flagSources)
	}

	log.InfoContext(ctx, "loading config")
	c.logConfigSources(ctx, log, bindings)
	log.InfoContext(ctx, "config loaded")
	return nil
}

// SaveToJSON writes the current struct to databasus-verification.json.
func (c *Config) SaveToJSON() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFileName, data, 0o600)
}

// LoadFromJSON reads databasus-verification.json without applying CLI flags.
// The detached _run daemon uses this: the parent `start` already validated and
// persisted the config, so the child trusts the file as-is.
func (c *Config) LoadFromJSON(ctx context.Context, log *slog.Logger) {
	c.loadFromJSON(ctx, log)
}

func (c *Config) Validate() (Capacity, error) {
	if c.DatabasusHost == "" {
		return Capacity{}, fmt.Errorf("databasus-host is required")
	}

	if c.AgentID == "" {
		return Capacity{}, fmt.Errorf("agent-id is required")
	}

	if c.Token == "" {
		return Capacity{}, fmt.Errorf("token is required")
	}

	return c.DeriveCapacity()
}

func (c *Config) DeriveCapacity() (Capacity, error) {
	if c.MaxConcurrentJobs < 1 {
		return Capacity{}, fmt.Errorf(
			"max-concurrent-jobs must be >= 1 (got %d)", c.MaxConcurrentJobs)
	}

	if c.MaxCPU < 1 {
		return Capacity{}, fmt.Errorf("max-cpu must be >= 1 (got %d)", c.MaxCPU)
	}

	if c.MaxDiskGb < 1 {
		return Capacity{}, fmt.Errorf("max-disk-gb must be >= 1 (got %d)", c.MaxDiskGb)
	}

	if c.MaxRAMMb < minRAMMbPerJob {
		return Capacity{}, fmt.Errorf(
			"max-ram-mb must be >= %d (got %d)", minRAMMbPerJob, c.MaxRAMMb)
	}

	cpuPerJob := c.MaxCPU / c.MaxConcurrentJobs
	ramMbPerJob := c.MaxRAMMb / c.MaxConcurrentJobs

	if cpuPerJob < 1 {
		return Capacity{}, fmt.Errorf(
			"max-cpu (%d) split across max-concurrent-jobs (%d) yields < 1 CPU per job; "+
				"lower max-concurrent-jobs or raise max-cpu",
			c.MaxCPU, c.MaxConcurrentJobs)
	}

	if ramMbPerJob < minRAMMbPerJob {
		return Capacity{}, fmt.Errorf(
			"max-ram-mb (%d) split across max-concurrent-jobs (%d) yields %d MB per job, "+
				"below the %d MB floor; lower max-concurrent-jobs or raise max-ram-mb",
			c.MaxRAMMb, c.MaxConcurrentJobs, ramMbPerJob, minRAMMbPerJob)
	}

	return Capacity{
		MaxCPU:            c.MaxCPU,
		MaxRAMMb:          c.MaxRAMMb,
		MaxDiskGb:         c.MaxDiskGb,
		MaxConcurrentJobs: c.MaxConcurrentJobs,
		CPUPerJob:         cpuPerJob,
		RAMMbPerJob:       ramMbPerJob,
	}, nil
}

// ValidateTransport enforces the http/https gate before any goroutine starts.
// The per-agent token and the decrypted backup stream both cross this link, so
// plain HTTP is allowed only with explicit operator consent.
func (c *Config) ValidateTransport(ctx context.Context, log *slog.Logger, isStdinTTY bool, in *bufio.Reader) error {
	parsed, err := url.Parse(c.DatabasusHost)
	if err != nil {
		return fmt.Errorf("databasus-host is not a valid URL: %w", err)
	}

	switch parsed.Scheme {
	case "https":
		return nil

	case "http":
		return c.consentToInsecureHTTP(ctx, log, isStdinTTY, in)

	default:
		return fmt.Errorf(
			"databasus-host must start with https:// or http:// (got scheme %q)", parsed.Scheme)
	}
}

func (c *Config) loadFromJSON(ctx context.Context, log *slog.Logger) {
	data, err := os.ReadFile(configFileName)
	if err != nil {
		if os.IsNotExist(err) {
			log.InfoContext(ctx, "no databasus-verification.json found, will create on save")
			return
		}

		log.WarnContext(ctx, "failed to read databasus-verification.json", slog.Any("error", err))

		return
	}

	if err := json.Unmarshal(data, c); err != nil {
		log.WarnContext(ctx, "failed to parse databasus-verification.json", slog.Any("error", err))

		return
	}

	log.InfoContext(ctx, "configuration loaded", slog.Any("path", configFileName))
}

func (c *Config) consentToInsecureHTTP(ctx context.Context, log *slog.Logger, isStdinTTY bool, in *bufio.Reader) error {
	if c.AllowInsecureHTTP {
		log.WarnContext(ctx, "databasus-host is plain HTTP; transport is unencrypted")

		return nil
	}

	if !isStdinTTY {
		return fmt.Errorf(
			"refusing to use plain HTTP over a non-interactive connection: " +
				"switch --databasus-host to https:// or pass --allow-insecure-http " +
				"to accept unencrypted transport")
	}

	fmt.Fprint(os.Stderr,
		"WARNING: connecting to the primary over plain HTTP, not HTTPS. "+
			"The agent token and decrypted backup data are sent unencrypted. "+
			"Continue? [y/N] ")

	answer, _ := in.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer != "y" && answer != "yes" {
		return fmt.Errorf("aborted: plain HTTP transport declined by operator")
	}

	return nil
}

func maskSensitive(value string) string {
	if value == "" {
		return "(not set)"
	}

	visibleLen := max(len(value)/4, 1)

	return value[:visibleLen] + "***"
}
