package clienttools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"databasus-backend/internal/lib/process"
)

// clientVersionTimeout bounds one --version call. A client that has not answered by then is as
// unusable as one that is missing.
const clientVersionTimeout = 10 * time.Second

type Result struct {
	Bundle Bundle
	Errors []error
}

// Checker runs every client of every bundle once and keeps the verdict per bin directory. The
// container probes the health endpoint every 30 seconds, and starting every shipped binary on each
// probe is how a loaded host reports a false unhealthy, so later checks reread the files but reuse
// the verdict.
type Checker struct {
	bundles []Bundle

	mu       sync.Mutex
	verdicts map[string][]error
}

func NewChecker(bundles []Bundle) *Checker {
	return &Checker{bundles: bundles, verdicts: make(map[string][]error)}
}

// Check verifies that every bundle holds its commands and that each one starts and reports a
// version on the release line the bundle is registered under.
func (c *Checker) Check(ctx context.Context) []Result {
	results := make([]Result, 0, len(c.bundles))

	for _, bundle := range c.bundles {
		results = append(results, Result{Bundle: bundle, Errors: c.checkBundle(ctx, bundle)})
	}

	return results
}

// GetHealthError joins the failures of the fatal bundles, or returns nil when they all pass.
func (c *Checker) GetHealthError(ctx context.Context) error {
	var failures []string

	for _, result := range c.Check(ctx) {
		if !result.Bundle.IsFatal {
			continue
		}

		for _, err := range result.Errors {
			failures = append(failures, fmt.Sprintf("%s %s: %s", result.Bundle.Engine, result.Bundle.Version, err))
		}
	}

	if len(failures) == 0 {
		return nil
	}

	return fmt.Errorf("%w: %s", ErrClientToolsBroken, strings.Join(failures, "; "))
}

func (c *Checker) checkBundle(ctx context.Context, bundle Bundle) []error {
	if fileErrors := CheckBinDir(bundle.BinDir, bundle.RequiredCommands); len(fileErrors) > 0 {
		return fileErrors
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if verdict, hasVerdict := c.verdicts[bundle.BinDir]; hasVerdict {
		return verdict
	}

	verdict := runClients(ctx, bundle)

	// A check cut short by its caller says nothing about the binaries, so the next one runs them again.
	if ctx.Err() == nil {
		c.verdicts[bundle.BinDir] = verdict
	}

	return verdict
}

// runClients starts every required command with --version. Presence and the executable bit say
// nothing about a missing shared library, which is the failure a new bundle is most likely to
// introduce.
func runClients(ctx context.Context, bundle Bundle) []error {
	var clientErrors []error

	for _, command := range bundle.RequiredCommands {
		if err := runClient(ctx, bundle, filepath.Join(bundle.BinDir, command)); err != nil {
			clientErrors = append(clientErrors, err)
		}
	}

	return clientErrors
}

func runClient(ctx context.Context, bundle Bundle, commandPath string) error {
	clientCtx, cancel := context.WithTimeout(ctx, clientVersionTimeout)
	defer cancel()

	output, err := process.CombinedOutput(clientCtx, process.Spec{Path: commandPath, Args: []string{"--version"}})
	if err != nil {
		return fmt.Errorf("client command cannot run: %s: %w: %s", commandPath, err, strings.TrimSpace(string(output)))
	}

	if bundle.ReleaseLine == "" {
		return nil
	}

	reportedVersion, err := bundle.ParseVersion(string(output))
	if err != nil {
		return fmt.Errorf("%s: %w", commandPath, err)
	}

	if !isVersionOnReleaseLine(reportedVersion, bundle.ReleaseLine) {
		return fmt.Errorf(
			"%w: %s reports version %s, which is not on the %s release line its bundle is registered under",
			ErrForeignClientVersion,
			commandPath,
			reportedVersion,
			bundle.ReleaseLine,
		)
	}

	return nil
}

func (r Result) IsPassed() bool {
	return len(r.Errors) == 0
}
